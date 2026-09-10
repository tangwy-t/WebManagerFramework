package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/dto/response"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/app"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"

	"go.uber.org/zap"
)

// ── 上传参数默认值:与 v006 的 sys_config 种子值一致,配置读取失败时的兜底 ──
const (
	defaultFileUploadPath    = "./uploads"
	defaultFileUploadMaxSize = 10 << 20 // 10MB
	defaultFileModule        = "default"
)

// FileRepositoryInterface 文件服务所需的数据访问方法集(消费方接口)。
type FileRepositoryInterface interface {
	FindPage(ctx context.Context, q *request.FileQuery) ([]entity.SysFile, int64, error)
	FindByID(ctx context.Context, id uint64) (*entity.SysFile, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]entity.SysFile, error)
	CreateBatch(ctx context.Context, files []*entity.SysFile) error
	Rename(ctx context.Context, id uint64, name string) error
	DeleteByIDs(ctx context.Context, ids []uint64) error
	Stats(ctx context.Context, weekStart time.Time) (*entity.FileStatsRow, error)
}

// FileConfigProvider 上传参数配置读取接口(消费方定义,由 ConfigService 实现)。
type FileConfigProvider interface {
	GetString(ctx context.Context, key, defaultVal string) string
	GetInt(ctx context.Context, key string, defaultVal int) int
}

// FileService 文件管理服务:元数据 CRUD + 本地存储读写。
type FileService struct {
	repo       FileRepositoryInterface
	cfg        FileConfigProvider
	logger     logger.LoggerInterface
	thumbCache *thumbLRU
}

// NewFileService constructs a FileService with the given dependencies.
func NewFileService(repo FileRepositoryInterface, cfg FileConfigProvider, logger logger.LoggerInterface) *FileService {
	return &FileService{repo: repo, cfg: cfg, logger: logger, thumbCache: newThumbLRU()}
}

// List 分页查询文件列表。
func (s *FileService) List(ctx context.Context, q *request.FileQuery) (*app.PageResponse, error) {
	files, total, err := s.repo.FindPage(ctx, q)
	if err != nil {
		return nil, err
	}
	items := make([]response.FileResp, 0, len(files))
	for _, f := range files {
		items = append(items, toFileResp(f))
	}
	return app.NewPageResponse(items, total, q.GetPage(), q.GetPageSize()), nil
}

// Stats 返回文件概览统计(总数/总大小/近7天新增/分类计数)。
func (s *FileService) Stats(ctx context.Context) (*response.FileStatsResp, error) {
	row, err := s.repo.Stats(ctx, time.Now().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}
	return &response.FileStatsResp{
		Total:          row.Total,
		TotalSize:      row.TotalSize,
		WeekUploads:    row.WeekUploads,
		CategoryCounts: row.CategoryCounts(),
	}, nil
}

// Upload 校验并落盘上传文件,随后批量写入元数据。
// 所有文件先整体校验(大小/扩展名),再逐个落盘;任一步失败回滚已写入的物理文件,
// 保证磁盘与数据库一致。物理文件名随机生成,杜绝路径注入与同名覆盖。
func (s *FileService) Upload(ctx context.Context, headers []*multipart.FileHeader) ([]response.FileResp, error) {
	if len(headers) == 0 {
		return nil, apperror.BadRequest("未接收到文件")
	}

	maxSize := s.cfg.GetInt(ctx, "sys.file.upload.maxSize", defaultFileUploadMaxSize)
	allowedExts := parseAllowedExts(s.cfg.GetString(ctx, "sys.file.upload.allowedExts", ""))
	basePath := s.cfg.GetString(ctx, "sys.file.upload.path", defaultFileUploadPath)

	type pendingFile struct {
		header *multipart.FileHeader
		ext    string
		mime   string
	}
	pending := make([]pendingFile, 0, len(headers))
	for _, h := range headers {
		if h.Size > int64(maxSize) {
			return nil, apperror.BadRequest(fmt.Sprintf("%s 超过上传大小限制(%s)", h.Filename, formatFileBytes(int64(maxSize))))
		}
		ext := strings.ToLower(filepath.Ext(h.Filename))
		if len(allowedExts) > 0 && !allowedExts[ext] {
			return nil, apperror.BadRequest(fmt.Sprintf("%s 的文件类型 %s 不在允许上传范围内", h.Filename, ext))
		}
		pending = append(pending, pendingFile{header: h, ext: ext, mime: detectFileMime(h)})
	}

	if err := os.MkdirAll(basePath, 0o755); err != nil {
		s.logger.Error("create upload dir failed", zap.String("path", basePath), zap.Error(err))
		return nil, apperror.Internal("创建上传目录失败", err)
	}

	dateDir := time.Now().Format("2006/01")
	savedKeys := make([]string, 0, len(pending))
	rollback := func() {
		for _, rel := range savedKeys {
			_ = os.Remove(filepath.Join(basePath, rel))
		}
	}

	files := make([]*entity.SysFile, 0, len(pending))
	for _, p := range pending {
		relKey := filepath.Join(dateDir, randomFileKey()+p.ext)
		fullPath := filepath.Join(basePath, relKey)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			rollback()
			return nil, apperror.Internal("创建上传目录失败", err)
		}

		n, err := saveMultipart(p.header, fullPath, int64(maxSize))
		if err != nil {
			rollback()
			return nil, err
		}

		savedKeys = append(savedKeys, relKey)
		files = append(files, &entity.SysFile{
			Name:         p.header.Filename,
			OriginalName: p.header.Filename,
			Path:         relKey,
			Size:         ptr.To(n),
			MimeType:     ptr.To(p.mime),
			Ext:          ptr.To(p.ext),
			Module:       ptr.To(defaultFileModule),
			StorageType:  "local",
		})
	}

	if err := s.repo.CreateBatch(ctx, files); err != nil {
		rollback()
		return nil, err
	}

	items := make([]response.FileResp, 0, len(files))
	for _, f := range files {
		items = append(items, toFileResp(*f))
	}
	return items, nil
}

// Rename 更新文件显示名(Trim 后校验,不改变物理路径与扩展名)。
func (s *FileService) Rename(ctx context.Context, id uint64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return apperror.BadRequest("文件名不能为空")
	}
	if strings.ContainsAny(name, `/\`) {
		return apperror.BadRequest("文件名不能包含路径分隔符")
	}
	file, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if file == nil {
		return apperror.NotFound("文件不存在")
	}
	return s.repo.Rename(ctx, id, name)
}

// DeleteMany 批量删除文件元数据并清理本地磁盘文件。
func (s *FileService) DeleteMany(ctx context.Context, ids []uint64) error {
	files, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return apperror.NotFound("文件不存在")
	}

	basePath := s.cfg.GetString(ctx, "sys.file.upload.path", defaultFileUploadPath)
	for _, f := range files {
		if f.StorageType != "local" {
			continue
		}
		if err := os.Remove(filepath.Join(basePath, f.Path)); err != nil && !os.IsNotExist(err) {
			s.logger.Warn("remove file failed", zap.String("path", f.Path), zap.Error(err))
		}
	}
	return s.repo.DeleteByIDs(ctx, ids)
}

// OpenFile 打开文件用于下载/预览,返回实体与磁盘绝对路径。
func (s *FileService) OpenFile(ctx context.Context, id uint64) (*entity.SysFile, string, error) {
	file, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if file == nil {
		return nil, "", apperror.NotFound("文件不存在")
	}
	if file.StorageType != "local" {
		return nil, "", apperror.Internal(fmt.Sprintf("存储类型 %s 暂不支持在线访问", file.StorageType))
	}
	basePath := s.cfg.GetString(ctx, "sys.file.upload.path", defaultFileUploadPath)
	fullPath := filepath.Join(basePath, file.Path)
	if _, err := os.Stat(fullPath); err != nil {
		return nil, "", apperror.NotFound("文件已丢失")
	}
	return file, fullPath, nil
}

// ── 工具函数 ──────────────────────────────────────────────────────────

func toFileResp(f entity.SysFile) response.FileResp {
	resp := response.FileResp{
		ID:           f.ID,
		Name:         f.Name,
		OriginalName: f.OriginalName,
		StorageType:  f.StorageType,
		ModuleID:     f.ModuleID,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
	if f.Size != nil {
		resp.Size = *f.Size
	}
	if f.MimeType != nil {
		resp.MimeType = *f.MimeType
	}
	if f.Ext != nil {
		resp.Ext = *f.Ext
		resp.Category = entity.CategoryOfExt(*f.Ext)
	} else {
		resp.Category = entity.FileCategoryOther
	}
	if f.Module != nil {
		resp.Module = *f.Module
	}
	if f.CreatedBy != nil {
		resp.CreatedBy = *f.CreatedBy
	}
	return resp
}

// parseAllowedExts 解析逗号分隔的扩展名白名单(小写、含点);空配置 = 不限制。
func parseAllowedExts(raw string) map[string]bool {
	allowed := make(map[string]bool)
	for _, ext := range strings.Split(raw, ",") {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		allowed[ext] = true
	}
	return allowed
}

// randomFileKey 生成 16 字节随机 hex 存储键(crypto/rand,零额外依赖)。
func randomFileKey() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败属极端情形:时间戳兜底,保证可用性优先。
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// saveMultipart 将 multipart 文件写入目标路径,双重校验大小上限。
func saveMultipart(header *multipart.FileHeader, dstPath string, maxSize int64) (int64, error) {
	src, err := header.Open()
	if err != nil {
		return 0, apperror.BadRequest("读取上传文件失败")
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return 0, apperror.Internal("写入上传文件失败", err)
	}
	// 一旦文件创建成功,后续任何失败都必须删除已写文件:O_EXCL 建出的
	// 半成品/超限文件若残留,调用方的 rollback 只会清理「已登记 savedKeys」,
	// 未登记的本文件会永久残留为孤儿(磁盘泄漏)。故在此统一兜底清理。
	removeOnError := true
	defer func() {
		if removeOnError {
			_ = os.Remove(dstPath)
		}
	}()

	n, copyErr := io.Copy(dst, io.LimitReader(src, maxSize+1))
	closeErr := dst.Close()
	if copyErr != nil {
		return 0, apperror.Internal("写入上传文件失败", copyErr)
	}
	if closeErr != nil {
		return 0, apperror.Internal("写入上传文件失败", closeErr)
	}
	if n > maxSize {
		return 0, apperror.BadRequest(fmt.Sprintf("%s 超过上传大小限制(%s)", header.Filename, formatFileBytes(maxSize)))
	}
	removeOnError = false
	return n, nil
}

// detectFileMime 取 multipart 声明的 Content-Type,缺省按扩展名兜底。
func detectFileMime(header *multipart.FileHeader) string {
	if mimeType := header.Header.Get("Content-Type"); mimeType != "" && mimeType != "application/octet-stream" {
		return mimeType
	}
	return mime.TypeByExtension(strings.ToLower(filepath.Ext(header.Filename)))
}

// formatFileBytes 将字节数格式化为可读文案(用于错误提示)。
func formatFileBytes(size int64) string {
	const unit = 1024
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	i := 0
	for value >= unit && i < len(units)-1 {
		value /= unit
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", size, units[i])
	}
	return fmt.Sprintf("%.1f %s", value, units[i])
}

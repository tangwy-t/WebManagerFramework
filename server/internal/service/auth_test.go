package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/contextkeys"
	"github.com/tangwy-t/webmanager-server/internal/pkg/crypto"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"gorm.io/gorm"
)

// ── 测试替身 ────────────────────────────────────────────
// stubAuthRepo 满足 AuthRepositoryInterface;仅 FindByID 被测,其余一律零值返回。
type stubAuthRepo struct {
	findByIDUser *entity.SysUser
	findByIDErr  error
	findMenuPermsFn func(context.Context) ([]string, error)
}

func (m *stubAuthRepo) FindByUsername(context.Context, string) (*entity.SysUser, error) {
	return nil, nil
}
func (m *stubAuthRepo) FindByID(_ context.Context, _ uint64) (*entity.SysUser, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDUser, nil
}
func (m *stubAuthRepo) GetRoleCodes(context.Context, uint64) ([]string, error) {
	return nil, nil
}
func (m *stubAuthRepo) FindMenuPerms(ctx context.Context) ([]string, error) {
	if m.findMenuPermsFn == nil {
		return nil, nil
	}
	return m.findMenuPermsFn(ctx)
}
func (m *stubAuthRepo) GetUserPermissions(context.Context, uint64) ([]string, error) {
	return nil, nil
}
func (m *stubAuthRepo) GetUserDataScope(context.Context, uint64) (int8, uint64, error) {
	return 0, 0, nil
}
func (m *stubAuthRepo) GetUserRoleScope(context.Context, uint64) int8 { return 0 }
func (m *stubAuthRepo) UpdatePassword(context.Context, uint64, string, *string) error {
	return nil
}
func (m *stubAuthRepo) UpdateLoginInfo(context.Context, uint64, string) error { return nil }
func (m *stubAuthRepo) UpdateProfile(context.Context, uint64, *string, *string, *string) error {
	return nil
}
func (m *stubAuthRepo) UpdateAvatar(context.Context, uint64, string) error { return nil }
func (m *stubAuthRepo) CountUserLogins(context.Context, uint64) (int64, error) {
	return 0, nil
}
func (m *stubAuthRepo) FindUserLoginLogsSince(context.Context, uint64, time.Time) ([]entity.SysLoginLog, error) {
	return nil, nil
}
func (m *stubAuthRepo) GetDeptName(context.Context, uint64) (string, error) { return "", nil }

// newVerifyService 构造仅 VerifyPassword 依赖的极简 AuthService(其余依赖零值)。
func newVerifyService(repo AuthRepositoryInterface) *AuthService {
	return NewAuthService(nil, repo, logger.NewNop(), nil, nil, nil, "")
}

func newStubUser(t *testing.T, password string) *entity.SysUser {
	t.Helper()
	hash, salt, err := crypto.HashPassword(password, 10)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	return &entity.SysUser{Password: hash, PasswordSalt: &salt}
}

func wantCode(t *testing.T, err error, code int) {
	t.Helper()
	var ae *apperror.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("err = %v, 不是 AppError", err)
	}
	if ae.Code != code {
		t.Fatalf("err.Code = %d, want %d", ae.Code, code)
	}
}

func TestAuthServiceVerifyPasswordOK(t *testing.T) {
	svc := newVerifyService(&stubAuthRepo{findByIDUser: newStubUser(t, "admin123")})
	ctx := contextkeys.WithUserID(context.Background(), 1)

	got, err := svc.VerifyPassword(ctx, &request.VerifyPasswordReq{Password: "admin123"})
	if err != nil {
		t.Fatalf("VerifyPassword err: %v", err)
	}
	if got == nil || !got.Valid {
		t.Fatalf("resp = %+v, want Valid=true", got)
	}
}

func TestAuthServiceVerifyPasswordWrong(t *testing.T) {
	svc := newVerifyService(&stubAuthRepo{findByIDUser: newStubUser(t, "admin123")})
	ctx := contextkeys.WithUserID(context.Background(), 1)

	_, err := svc.VerifyPassword(ctx, &request.VerifyPasswordReq{Password: "wrong-pass"})
	wantCode(t, err, apperror.CodeBadRequest)
}

func TestAuthServiceVerifyPasswordUserNotFound(t *testing.T) {
	svc := newVerifyService(&stubAuthRepo{findByIDErr: gorm.ErrRecordNotFound})
	ctx := contextkeys.WithUserID(context.Background(), 1)

	_, err := svc.VerifyPassword(ctx, &request.VerifyPasswordReq{Password: "x"})
	wantCode(t, err, apperror.CodeNotFound)
}

func TestAuthServiceVerifyPasswordUnauthorized(t *testing.T) {
	svc := newVerifyService(&stubAuthRepo{})
	// 不注入 userID
	_, err := svc.VerifyPassword(context.Background(), &request.VerifyPasswordReq{Password: "x"})
	wantCode(t, err, apperror.CodeUnauthorized)
}

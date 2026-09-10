package service

import (
	"context"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/dto/response"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/contextkeys"
	"github.com/tangwy-t/webmanager-server/internal/pkg/crypto"

	"go.uber.org/zap"
)

func (s *AuthService) ChangePassword(ctx context.Context, req *request.ChangePasswordReq, accessToken string) error {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return apperror.Unauthorized("未登录")
	}
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return translateNotFound(err, "用户不存在")
	}
	salt := ""
	if user.PasswordSalt != nil {
		salt = *user.PasswordSalt
	}
	if !crypto.VerifyPassword(req.OldPassword, user.Password, salt) {
		return apperror.BadRequest("旧密码不正确")
	}
	hashed, newSalt, err := crypto.HashPassword(req.NewPassword, s.cfgProv.GetInt(ctx, "sys.auth.bcryptCost", 10))
	if err != nil {
		return apperror.Internal("密码加密失败")
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hashed), &newSalt); err != nil {
		return err
	}
	// Revoke all existing sessions so that old tokens are invalidated after
	// password change. The current access token is passed in so it dies
	// immediately instead of staying valid (up to its 2h expiry) alongside the
	// possibly stolen one it was reported through.
	if err := s.sessionStore.RevokeAll(ctx, userID, accessToken); err != nil {
		// Error:改密(常因疑似泄露)后旧 token 残留至 TTL——安全敏感窗口。
		// DB 密码已更新不能回滚,故记录 Error 并继续,窗口由运维告警跟进。
		s.logger.Error("failed to revoke sessions after password change", zap.Uint64("userId", userID), zap.Error(err))
	}
	return nil
}

// VerifyPassword checks the provided password against the current user's
// login password. Used by the lock screen unlock flow; intentionally does
// NOT touch tokens or sessions (unlock is a pure front-door check, real
// session security remains with logout / password change).
func (s *AuthService) VerifyPassword(ctx context.Context, req *request.VerifyPasswordReq) (*response.VerifyPasswordResp, error) {
	userID, ok := contextkeys.UserIDFromCtx(ctx)
	if !ok {
		return nil, apperror.Unauthorized("未登录")
	}
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, translateNotFound(err, "用户不存在")
	}
	salt := ""
	if user.PasswordSalt != nil {
		salt = *user.PasswordSalt
	}
	if !crypto.VerifyPassword(req.Password, user.Password, salt) {
		return nil, apperror.BadRequest("密码不正确")
	}
	return &response.VerifyPasswordResp{Valid: true}, nil
}

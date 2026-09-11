package service

import (
	"context"
	"testing"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/contextkeys"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
)

type fakePwdRepo struct {
	user            *entity.SysUser
	updatedPassword string
	mustChange      *bool
}

func (f *fakePwdRepo) FindByID(ctx context.Context, id uint64) (*entity.SysUser, error) {
	return f.user, nil
}
func (f *fakePwdRepo) UpdatePassword(ctx context.Context, id uint64, pwd string, salt *string) error {
	f.updatedPassword = pwd
	return nil
}
func (f *fakePwdRepo) SetMustChangePassword(ctx context.Context, id uint64, m bool) error {
	f.mustChange = &m
	return nil
}

func TestPasswordService_ChangePassword_RejectsWeak(t *testing.T) {
	svc := NewPasswordService(nil, &fakePwdRepo{}, nil, logger.NewNop())
	ctx := contextkeys.WithUserID(context.Background(), 1)
	err := svc.ChangePassword(ctx, &request.ChangePasswordReq{
		OldPassword: "old",
		NewPassword: "short",
	}, "")
	if err == nil {
		t.Fatal("weak password accepted")
	}
}
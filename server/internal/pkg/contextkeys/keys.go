package contextkeys

import "context"

type contextKey string

const TraceID contextKey = "traceId"

// ─── 身份与授权上下文键 ─────────────────────────────────────────────
// 这些键由 HTTP 中间件(middleware.Auth)与 WebSocket 客户端(ws.Client)
// 写入 request/context,供 service、repository(GORM 审计回调)读取。
// 集中定义在此处,避免业务层反向依赖 middleware(传输层)。
const (
	UserID    contextKey = "userID"
	DataScope contextKey = "dataScope"
	DeptID    contextKey = "deptID"
)

// TraceIDFromCtx 从 context.Context 中提取 traceId。
// 返回空字符串和 false 如果未找到。
func TraceIDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(TraceID).(string)
	return id, ok
}

// WithTraceID 将 traceId 注入 context.Context。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceID, traceID)
}

// UserIDFromCtx 提取认证用户 ID;未认证返回 0,false。
func UserIDFromCtx(ctx context.Context) (uint64, bool) {
	id, ok := ctx.Value(UserID).(uint64)
	return id, ok
}

// WithUserID 注入认证用户 ID(供中间件、WS 客户端与测试使用)。
func WithUserID(ctx context.Context, userID uint64) context.Context {
	return context.WithValue(ctx, UserID, userID)
}

// DataScopeFromCtx 提取用户数据范围等级(1-5);缺失返回 0,false。
func DataScopeFromCtx(ctx context.Context) (int8, bool) {
	ds, ok := ctx.Value(DataScope).(int8)
	return ds, ok
}

// WithDataScope 注入数据范围等级。
func WithDataScope(ctx context.Context, level int8) context.Context {
	return context.WithValue(ctx, DataScope, level)
}

// DeptIDFromCtx 提取用户部门 ID;缺失返回 0,false。
func DeptIDFromCtx(ctx context.Context) (uint64, bool) {
	did, ok := ctx.Value(DeptID).(uint64)
	return did, ok
}

// WithDeptID 注入部门 ID。
func WithDeptID(ctx context.Context, deptID uint64) context.Context {
	return context.WithValue(ctx, DeptID, deptID)
}

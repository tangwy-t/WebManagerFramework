package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
)

// captureOpLogService captures operation log entries created by the middleware.
type captureOpLogService struct {
	logs chan *entity.SysOperationLog
}

func (s *captureOpLogService) Create(ctx context.Context, log *entity.SysOperationLog) error {
	s.logs <- log
	return nil
}

// setupOpLogRouter builds a test engine with the same middleware order as production:
// Recovery (outer) → OperationLogMiddleware → route handler.
func setupOpLogRouter() (*gin.Engine, *captureOpLogService) {
	gin.SetMode(gin.TestMode)
	svc := &captureOpLogService{logs: make(chan *entity.SysOperationLog, 1)}
	log := logger.NewNop()

	r := gin.New()
	r.Use(Recovery(log))
	r.Use(OperationLogMiddleware(svc, log))
	return r, svc
}

// waitLog reads one captured log entry, failing the test on timeout (save is async).
func waitLog(t *testing.T, svc *captureOpLogService) *entity.SysOperationLog {
	t.Helper()
	select {
	case entry := <-svc.logs:
		return entry
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for operation log entry")
		return nil
	}
}

// TestOperationLogMiddlewareRecordsSuccess verifies a successful envelope
// (HTTP 200 + code 0) is stored with code 0 and no error message.
func TestOperationLogMiddlewareRecordsSuccess(t *testing.T) {
	r, svc := setupOpLogRouter()
	r.POST("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": nil})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ok", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}

	entry := waitLog(t, svc)
	if entry.Code != apperror.CodeOK {
		t.Fatalf("expected code %d, got %d", apperror.CodeOK, entry.Code)
	}
	if entry.ErrorMsg != nil {
		t.Fatalf("expected nil errorMsg for success, got %q", *entry.ErrorMsg)
	}
	if entry.RequestParams == nil || !strings.Contains(*entry.RequestParams, `"name"`) {
		t.Fatalf("expected request params to be recorded, got %v", entry.RequestParams)
	}
}

// TestOperationLogMiddlewareRecordsBusiness400 verifies that a handler error
// (HTTP 400 envelope with business code) stores the business code and message.
func TestOperationLogMiddlewareRecordsBusiness400(t *testing.T) {
	r, svc := setupOpLogRouter()
	r.POST("/fail", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40000, "msg": "参数错误", "data": nil})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/fail", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d", w.Code)
	}

	entry := waitLog(t, svc)
	if entry.Code != apperror.CodeBadRequest {
		t.Fatalf("expected code %d, got %d", apperror.CodeBadRequest, entry.Code)
	}
	if entry.ErrorMsg == nil || *entry.ErrorMsg != "参数错误" {
		t.Fatalf("expected errorMsg 参数错误, got %v", entry.ErrorMsg)
	}
	if entry.RequestMethod == nil || *entry.RequestMethod != http.MethodPost {
		t.Fatalf("expected POST request method, got %v", entry.RequestMethod)
	}
}

// TestOperationLogMiddlewareBusinessCodeWinsOverHTTPStatus verifies the core
// guarantee for this feature: the business code in the envelope decides the
// result even when the HTTP status is 200 (the framework's common case).
func TestOperationLogMiddlewareBusinessCodeWinsOverHTTPStatus(t *testing.T) {
	r, svc := setupOpLogRouter()
	r.POST("/fail-http200", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 40000, "msg": "参数错误", "data": nil})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/fail-http200", strings.NewReader(`{"x":1}`)))

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}

	entry := waitLog(t, svc)
	if entry.Code != apperror.CodeBadRequest {
		t.Fatalf("expected code %d (business code wins over HTTP 200), got %d", apperror.CodeBadRequest, entry.Code)
	}
	if entry.ErrorMsg == nil || *entry.ErrorMsg != "参数错误" {
		t.Fatalf("expected errorMsg 参数错误, got %v", entry.ErrorMsg)
	}
}

// TestOperationLogMiddlewareRecordsPanic verifies that a panic in the handler
// (500 via Recovery) is recorded with code 50000 and the panic information,
// and that the client still receives the 500 response from Recovery.
func TestOperationLogMiddlewareRecordsPanic(t *testing.T) {
	r, svc := setupOpLogRouter()
	r.POST("/boom", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/boom", strings.NewReader(`{"x":1}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500 from Recovery, got %d", w.Code)
	}

	entry := waitLog(t, svc)
	if entry.Code != apperror.CodeInternal {
		t.Fatalf("expected code %d, got %d", apperror.CodeInternal, entry.Code)
	}
	// The response body is written by Recovery after the panic, so errorMsg
	// should carry the panic information captured by the middleware.
	if entry.ErrorMsg == nil || !strings.Contains(*entry.ErrorMsg, "panic: boom") {
		t.Fatalf("expected errorMsg to contain panic info, got %v", entry.ErrorMsg)
	}
}

// TestOperationLogMiddlewareSkipsGET verifies GET requests are not logged.
func TestOperationLogMiddlewareSkipsGET(t *testing.T) {
	r, svc := setupOpLogRouter()
	r.GET("/get", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/get", nil))

	select {
	case <-svc.logs:
		t.Fatal("expected no operation log for GET request")
	case <-time.After(150 * time.Millisecond):
		// pass
	}
}
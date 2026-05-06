package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/service"
)

// ── Stub EnrollmentService ──────────────────────────────────────────────────

type stubEnrollmentService struct {
	createResult     *service.EnrollResult
	createErr        error
	statusEnrollment *domain.Enrollment
	statusActivity   *domain.Activity
	statusOrder      *domain.Order
	statusErr        error
	listResult       []domain.Enrollment
	listTotal        int64
	listErr          error
	lastUserID       uint64
	lastActivityID   uint64
	lastEnrollmentID uint64
	lastPage         int
	lastPageSize     int
	cancelErr        error
}

func (s *stubEnrollmentService) Create(userID, activityID uint64) (*service.EnrollResult, error) {
	s.lastUserID = userID
	s.lastActivityID = activityID
	if s.createErr != nil {
		return nil, s.createErr
	}
	return s.createResult, nil
}

func (s *stubEnrollmentService) GetStatus(enrollmentID, userID uint64) (*domain.Enrollment, *domain.Activity, *domain.Order, error) {
	s.lastEnrollmentID = enrollmentID
	s.lastUserID = userID
	if s.statusErr != nil {
		return nil, nil, nil, s.statusErr
	}
	return s.statusEnrollment, s.statusActivity, s.statusOrder, nil
}

func (s *stubEnrollmentService) ListByUser(userID uint64, page, pageSize int) ([]domain.Enrollment, int64, error) {
	s.lastUserID = userID
	s.lastPage = page
	s.lastPageSize = pageSize
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.listResult, s.listTotal, nil
}

func (s *stubEnrollmentService) Cancel(ctx context.Context, enrollmentID, userID uint64) error {
	s.lastEnrollmentID = enrollmentID
	s.lastUserID = userID
	return s.cancelErr
}

// ── Test Cases ──────────────────────────────────────────────────────────────

func TestEnrollmentHandler_Create_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{
		createResult: &service.EnrollResult{
			EnrollmentID: 456,
			Status:       "SUCCESS",
			OrderNo:      "ORD202604060001",
		},
	}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Create(c)
	})

	reqBody := map[string]uint64{"activity_id": 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["enrollment_id"].(float64) != 456 {
		t.Errorf("want enrollment_id=456, got %v", data["enrollment_id"])
	}
	if stub.lastUserID != 99 {
		t.Errorf("want userID=99, got %d", stub.lastUserID)
	}
	if stub.lastActivityID != 123 {
		t.Errorf("want activityID=123, got %d", stub.lastActivityID)
	}
}

func TestEnrollmentHandler_Create_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/enrollments", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestEnrollmentHandler_Create_MissingActivityID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/enrollments", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestEnrollmentHandler_Create_AlreadyEnrolled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{createErr: service.ErrAlreadyEnrolled}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Create(c)
	})

	reqBody := map[string]uint64{"activity_id": 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestEnrollmentHandler_Create_StockInsufficient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{createErr: service.ErrStockInsufficient}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Create(c)
	})

	reqBody := map[string]uint64{"activity_id": 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 1101 {
		t.Errorf("want code=1101, got %v", resp["code"])
	}
}

func TestEnrollmentHandler_Create_ActivityNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{createErr: service.ErrActivityNotFound}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Create(c)
	})

	reqBody := map[string]uint64{"activity_id": 999}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestEnrollmentHandler_Create_EnrollmentClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{createErr: service.ErrEnrollmentClosed}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Create(c)
	})

	reqBody := map[string]uint64{"activity_id": 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestEnrollmentHandler_Create_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{createErr: errors.New("db error")}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.POST("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Create(c)
	})

	reqBody := map[string]uint64{"activity_id": 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestEnrollmentHandler_GetStatus_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	stub := &stubEnrollmentService{
		statusEnrollment: &domain.Enrollment{
			ID:         456,
			ActivityID: 123,
			Status:     "SUCCESS",
			EnrolledAt: now,
		},
		statusActivity: &domain.Activity{Title: "Test Activity"},
		statusOrder:    &domain.Order{OrderNo: "ORD202604060001"},
	}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.GET("/api/v1/enrollments/:id/status", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.GetStatus(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrollments/456/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["enrollment_id"].(float64) != 456 {
		t.Errorf("want enrollment_id=456, got %v", data["enrollment_id"])
	}
	if data["activity_title"] != "Test Activity" {
		t.Errorf("want activity_title, got %v", data["activity_title"])
	}
	if data["order_no"] != "ORD202604060001" {
		t.Errorf("want order_no, got %v", data["order_no"])
	}
}

func TestEnrollmentHandler_GetStatus_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.GET("/api/v1/enrollments/:id/status", h.GetStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrollments/abc/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestEnrollmentHandler_GetStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{statusErr: service.ErrEnrollNotFound}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.GET("/api/v1/enrollments/:id/status", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.GetStatus(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrollments/999/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestEnrollmentHandler_List_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{
		listResult: []domain.Enrollment{{ID: 1, ActivityID: 123}},
		listTotal:  1,
	}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.GET("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrollments?page=2&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if stub.lastPage != 2 {
		t.Errorf("want page=2, got %d", stub.lastPage)
	}
	if stub.lastPageSize != 10 {
		t.Errorf("want pageSize=10, got %d", stub.lastPageSize)
	}
}

func TestEnrollmentHandler_List_DefaultPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{
		listResult: []domain.Enrollment{},
		listTotal:  0,
	}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.GET("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrollments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if stub.lastPage != 1 {
		t.Errorf("want default page=1, got %d", stub.lastPage)
	}
	if stub.lastPageSize != 20 {
		t.Errorf("want default pageSize=20, got %d", stub.lastPageSize)
	}
}

func TestEnrollmentHandler_List_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubEnrollmentService{listErr: errors.New("db error")}
	h := NewEnrollmentHandler(stub)

	r := gin.New()
	r.GET("/api/v1/enrollments", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrollments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

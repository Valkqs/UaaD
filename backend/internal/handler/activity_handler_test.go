package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/repository"
	"github.com/uaad/backend/internal/service"
)

// ── Stub ActivityService ────────────────────────────────────────────────────

type stubActivityService struct {
	createResult       uint64
	createErr          error
	updateErr          error
	publishResult      *domain.Activity
	publishErr         error
	listResult         []domain.Activity
	listTotal          int64
	listErr            error
	detailResult       *domain.Activity
	detailErr          error
	stockRemaining     int
	stockMaxCapacity   int
	stockErr           error
	merchantListResult []domain.Activity
	merchantListErr    error
	lastMerchantID     uint64
	lastActivityID     uint64
	lastFilter         repository.ActivityFilter
	lastPage           int
	lastPageSize       int
}

func (s *stubActivityService) Create(merchantID uint64, req service.CreateActivityReq) (uint64, error) {
	s.lastMerchantID = merchantID
	if s.createErr != nil {
		return 0, s.createErr
	}
	return s.createResult, nil
}

func (s *stubActivityService) Update(activityID, merchantID uint64, req service.UpdateActivityReq) error {
	s.lastActivityID = activityID
	s.lastMerchantID = merchantID
	return s.updateErr
}

func (s *stubActivityService) Publish(activityID, merchantID uint64) (*domain.Activity, error) {
	s.lastActivityID = activityID
	s.lastMerchantID = merchantID
	if s.publishErr != nil {
		return nil, s.publishErr
	}
	return s.publishResult, nil
}

func (s *stubActivityService) List(filter repository.ActivityFilter, page, pageSize int) ([]domain.Activity, int64, error) {
	s.lastFilter = filter
	s.lastPage = page
	s.lastPageSize = pageSize
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.listResult, s.listTotal, nil
}

func (s *stubActivityService) Detail(id uint64) (*domain.Activity, error) {
	s.lastActivityID = id
	if s.detailErr != nil {
		return nil, s.detailErr
	}
	return s.detailResult, nil
}

func (s *stubActivityService) Stock(id uint64) (remaining int, maxCapacity int, err error) {
	s.lastActivityID = id
	if s.stockErr != nil {
		return 0, 0, s.stockErr
	}
	return s.stockRemaining, s.stockMaxCapacity, nil
}

func (s *stubActivityService) MerchantList(merchantID uint64) ([]domain.Activity, error) {
	s.lastMerchantID = merchantID
	if s.merchantListErr != nil {
		return nil, s.merchantListErr
	}
	return s.merchantListResult, nil
}

// ── Test Cases ──────────────────────────────────────────────────────────────

func TestActivityHandler_Create_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{createResult: 123}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.POST("/api/v1/activities", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Create(c)
	})

	reqBody := map[string]interface{}{
		"title":           "Test Activity",
		"description":     "Test Description",
		"location":        "Test Location",
		"category":        "CONCERT",
		"max_capacity":    100,
		"price":           99.0,
		"enroll_open_at":  time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		"enroll_close_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"activity_at":     time.Now().Add(48 * time.Hour).Format(time.RFC3339),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["activity_id"].(float64) != 123 {
		t.Errorf("want activity_id=123, got %v", data["activity_id"])
	}
	if stub.lastMerchantID != 99 {
		t.Errorf("want merchantID=99, got %d", stub.lastMerchantID)
	}
}

func TestActivityHandler_Create_InvalidTimeRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{createErr: service.ErrInvalidTimeRange}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.POST("/api/v1/activities", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Create(c)
	})

	reqBody := map[string]interface{}{
		"title":           "Test",
		"description":     "Test",
		"location":        "Test",
		"category":        "CONCERT",
		"max_capacity":    100,
		"price":           99.0,
		"enroll_open_at":  time.Now().Format(time.RFC3339),
		"enroll_close_at": time.Now().Format(time.RFC3339),
		"activity_at":     time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestActivityHandler_Create_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.POST("/api/v1/activities", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestActivityHandler_Update_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.PUT("/api/v1/activities/:id", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Update(c)
	})

	title := "Updated Title"
	reqBody := map[string]interface{}{"title": title}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/123", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if stub.lastActivityID != 123 {
		t.Errorf("want activityID=123, got %d", stub.lastActivityID)
	}
}

func TestActivityHandler_Update_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.PUT("/api/v1/activities/:id", h.Update)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/abc", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestActivityHandler_Update_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{updateErr: service.ErrActivityNotFound}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.PUT("/api/v1/activities/:id", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Update(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/123", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestActivityHandler_Update_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{updateErr: service.ErrNotActivityOwner}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.PUT("/api/v1/activities/:id", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Update(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/123", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestActivityHandler_Update_Published(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{updateErr: service.ErrActivityPublished}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.PUT("/api/v1/activities/:id", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Update(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/123", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestActivityHandler_Publish_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{
		publishResult: &domain.Activity{ID: 123, Status: "PUBLISHED", MaxCapacity: 100},
	}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.PUT("/api/v1/activities/:id/publish", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Publish(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/123/publish", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["status"] != "PUBLISHED" {
		t.Errorf("want status=PUBLISHED, got %v", data["status"])
	}
}

func TestActivityHandler_Publish_InvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{publishErr: service.ErrInvalidActivityState}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.PUT("/api/v1/activities/:id/publish", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.Publish(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/123/publish", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestActivityHandler_List_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{
		listResult: []domain.Activity{{ID: 1, Title: "Activity 1"}},
		listTotal:  1,
	}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.GET("/api/v1/activities", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities?page=1&page_size=20&category=CONCERT&sort=hot", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if stub.lastFilter.Category != "CONCERT" {
		t.Errorf("want category=CONCERT, got %s", stub.lastFilter.Category)
	}
	if stub.lastFilter.Sort != "hot" {
		t.Errorf("want sort=hot, got %s", stub.lastFilter.Sort)
	}
}

func TestActivityHandler_Detail_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{
		detailResult: &domain.Activity{
			ID:          123,
			Title:       "Test Activity",
			MaxCapacity: 100,
			EnrollCount: 30,
		},
		stockRemaining:   70,
		stockMaxCapacity: 100,
	}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.GET("/api/v1/activities/:id", h.Detail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["stock_remaining"].(float64) != 70 {
		t.Errorf("want stock_remaining=70, got %v", data["stock_remaining"])
	}
}

func TestActivityHandler_Detail_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{detailErr: service.ErrActivityNotFound}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.GET("/api/v1/activities/:id", h.Detail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestActivityHandler_Stock_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{
		stockRemaining:   50,
		stockMaxCapacity: 100,
	}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.GET("/api/v1/activities/:id/stock", h.Stock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/123/stock", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["stock_remaining"].(float64) != 50 {
		t.Errorf("want stock_remaining=50, got %v", data["stock_remaining"])
	}
}

func TestActivityHandler_MerchantList_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{
		merchantListResult: []domain.Activity{{ID: 1, Title: "My Activity"}},
	}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.GET("/api/v1/activities/merchant", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.MerchantList(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/merchant", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if stub.lastMerchantID != 99 {
		t.Errorf("want merchantID=99, got %d", stub.lastMerchantID)
	}
}

func TestActivityHandler_MerchantList_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubActivityService{merchantListErr: errors.New("db error")}
	h := NewActivityHandler(stub)

	r := gin.New()
	r.GET("/api/v1/activities/merchant", func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		h.MerchantList(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/merchant", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

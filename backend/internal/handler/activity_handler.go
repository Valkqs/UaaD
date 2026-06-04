package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/uaad/backend/internal/repository"
	"github.com/uaad/backend/internal/service"
	"github.com/uaad/backend/pkg/response"
)

// ActivityHandler handles activity-related HTTP requests.
type ActivityHandler struct {
	svc service.ActivityService
}

// NewActivityHandler creates a new ActivityHandler.
func NewActivityHandler(svc service.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

// Create handles POST /api/v1/activities.
func (h *ActivityHandler) Create(c *gin.Context) {
	var req service.CreateActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	merchantID := getUserID(c)
	activityID, err := h.svc.Create(merchantID, req)
	if err != nil {
		switch err {
		case service.ErrInvalidTimeRange:
			response.BadRequest(c, "时间参数无效: enroll_open_at < enroll_close_at < activity_at")
		default:
			response.InternalError(c, "创建活动失败")
		}
		return
	}

	response.Created(c, gin.H{
		"activity_id": activityID,
		"status":      "DRAFT",
	})
}

// Update handles PUT /api/v1/activities/:id.
func (h *ActivityHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动 ID")
		return
	}

	var req service.UpdateActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	merchantID := getUserID(c)
	if err := h.svc.Update(id, merchantID, req); err != nil {
		switch err {
		case service.ErrActivityNotFound:
			response.NotFound(c, "活动不存在")
		case service.ErrNotActivityOwner:
			response.Forbidden(c, "无权修改此活动")
		case service.ErrActivityPublished:
			response.BadRequest(c, "活动已上架，不可修改库存和开票时间")
		case service.ErrInvalidTimeRange:
			response.BadRequest(c, "时间参数无效")
		default:
			response.InternalError(c, "更新活动失败")
		}
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Preheat handles PUT /api/v1/activities/:id/preheat.
func (h *ActivityHandler) Preheat(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动 ID")
		return
	}

	merchantID := getUserID(c)
	activity, err := h.svc.Preheat(id, merchantID)
	if err != nil {
		switch err {
		case service.ErrActivityNotFound:
			response.NotFound(c, "活动不存在")
		case service.ErrNotActivityOwner:
			response.Forbidden(c, "无权操作此活动")
		case service.ErrInvalidActivityState:
			response.BadRequest(c, "当前状态不允许设为预热（仅 DRAFT 可进入 PREHEAT）")
		default:
			response.InternalError(c, "设为预热失败")
		}
		return
	}

	response.Success(c, gin.H{
		"activity_id": activity.ID,
		"status":      activity.Status,
	})
}

// Publish handles PUT /api/v1/activities/:id/publish.
func (h *ActivityHandler) Publish(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动 ID")
		return
	}

	merchantID := getUserID(c)
	activity, err := h.svc.Publish(id, merchantID)
	if err != nil {
		switch err {
		case service.ErrActivityNotFound:
			response.NotFound(c, "活动不存在")
		case service.ErrNotActivityOwner:
			response.Forbidden(c, "无权操作此活动")
		case service.ErrInvalidActivityState:
			response.BadRequest(c, "当前状态不允许上架（仅 DRAFT/PREHEAT 可上架）")
		default:
			response.InternalError(c, "上架失败")
		}
		return
	}

	response.Success(c, gin.H{
		"activity_id":    activity.ID,
		"status":         activity.Status,
		"stock_in_cache": activity.MaxCapacity,
	})
}

// List handles GET /api/v1/activities.
func (h *ActivityHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.ActivityFilter{
		Category: c.Query("category"),
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
		Sort:     c.DefaultQuery("sort", "recent"),
	}

	activities, total, err := h.svc.List(filter, page, pageSize)
	if err != nil {
		response.InternalError(c, "获取活动列表失败")
		return
	}

	response.Paginated(c, activities, total, page, pageSize)
}

// Detail handles GET /api/v1/activities/:id.
func (h *ActivityHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动 ID")
		return
	}

	activity, err := h.svc.Detail(id)
	if err != nil {
		response.NotFound(c, "活动不存在")
		return
	}

	remaining, _, err := h.svc.Stock(id)
	if err != nil {
		remaining = activity.MaxCapacity - int(activity.EnrollCount)
		if remaining < 0 {
			remaining = 0
		}
	}

	response.Success(c, gin.H{
		"activity_id":     activity.ID,
		"title":           activity.Title,
		"description":     activity.Description,
		"cover_url":       activity.CoverURL,
		"location":        activity.Location,
		"latitude":        activity.Latitude,
		"longitude":       activity.Longitude,
		"category":        activity.Category,
		"tags":            activity.Tags,
		"max_capacity":    activity.MaxCapacity,
		"price":           activity.Price,
		"enroll_open_at":  activity.EnrollOpenAt,
		"enroll_close_at": activity.EnrollCloseAt,
		"activity_at":     activity.ActivityAt,
		"status":          activity.Status,
		"enroll_count":    activity.EnrollCount,
		"stock_remaining": remaining,
		"created_by":      activity.CreatedBy,
	})
}

// Stock handles GET /api/v1/activities/:id/stock.
func (h *ActivityHandler) Stock(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的活动 ID")
		return
	}

	remaining, maxCapacity, err := h.svc.Stock(id)
	if err != nil {
		response.NotFound(c, "活动不存在")
		return
	}

	response.Success(c, gin.H{
		"activity_id":     id,
		"stock_remaining": remaining,
		"max_capacity":    maxCapacity,
	})
}

// MerchantList handles GET /api/v1/activities/merchant.
func (h *ActivityHandler) MerchantList(c *gin.Context) {
	merchantID := getUserID(c)
	activities, err := h.svc.MerchantList(merchantID)
	if err != nil {
		response.InternalError(c, "获取商户活动列表失败")
		return
	}

	response.Success(c, activities)
}

// getUserID extracts user_id from gin.Context (set by JWTAuth middleware).
func getUserID(c *gin.Context) uint64 {
	val, _ := c.Get("user_id")
	id, _ := val.(uint64)
	return id
}

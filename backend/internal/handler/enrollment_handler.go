package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/uaad/backend/internal/service"
	"github.com/uaad/backend/pkg/response"
)

// EnrollmentHandler handles enrollment-related HTTP requests.
type EnrollmentHandler struct {
	svc service.EnrollmentService
}

// NewEnrollmentHandler creates a new EnrollmentHandler.
func NewEnrollmentHandler(svc service.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{svc: svc}
}

// EnrollRequest represents the JSON body for creating an enrollment.
type EnrollRequest struct {
	ActivityID uint64 `json:"activity_id" binding:"required"`
}

// Create handles POST /api/v1/enrollments.
func (h *EnrollmentHandler) Create(c *gin.Context) {
	var req EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	userID := getUserID(c)
	result, err := h.svc.Create(userID, req.ActivityID)
	if err != nil {
		switch err {
		case service.ErrAlreadyEnrolled:
			response.Conflict(c, "您已对该活动提交报名，无需重复操作")
		case service.ErrStockInsufficient:
			c.JSON(200, gin.H{
				"code":    1101,
				"message": "库存不足，该活动已售罄",
				"data": gin.H{
					"activity_id":     req.ActivityID,
					"stock_remaining": 0,
				},
			})
		case service.ErrActivityNotFound:
			response.NotFound(c, "活动不存在")
		case service.ErrEnrollmentClosed:
			response.BadRequest(c, "报名通道未开放或已关闭")
		default:
			response.InternalError(c, "报名失败，请稍后重试")
		}
		return
	}

	data := gin.H{
		"status":         result.Status,
		"queue_position": result.QueuePosition,
	}
	if result.EnrollmentID != 0 {
		data["enrollment_id"] = result.EnrollmentID
	}
	if result.OrderNo != "" {
		data["order_no"] = result.OrderNo
	}
	response.Accepted(c, data)
}

// GetStatus handles GET /api/v1/enrollments/:id/status.
func (h *EnrollmentHandler) GetStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的报名 ID")
		return
	}

	userID := getUserID(c)
	enrollment, activity, order, err := h.svc.GetStatus(id, userID)
	if err != nil {
		response.NotFound(c, "报名记录不存在")
		return
	}

	data := gin.H{
		"enrollment_id": enrollment.ID,
		"activity_id":   enrollment.ActivityID,
		"status":        enrollment.Status,
		"submitted_at":  enrollment.EnrolledAt,
	}
	if enrollment.QueuePosition != nil {
		data["queue_position"] = *enrollment.QueuePosition
	}
	if activity != nil {
		data["activity_title"] = activity.Title
	}
	if enrollment.Status == "SUCCESS" && order != nil {
		data["order_no"] = order.OrderNo
		data["finalized_at"] = enrollment.FinalizedAt
	}

	response.Success(c, data)
}

// List handles GET /api/v1/enrollments.
func (h *EnrollmentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID := getUserID(c)
	enrollments, total, err := h.svc.ListByUser(userID, page, pageSize)
	if err != nil {
		response.InternalError(c, "获取报名列表失败")
		return
	}

	response.Paginated(c, enrollments, total, page, pageSize)
}

// Cancel handles POST /api/v1/enrollments/:id/cancel.
func (h *EnrollmentHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的报名 ID")
		return
	}

	userID := getUserID(c)
	if err := h.svc.Cancel(c.Request.Context(), id, userID); err != nil {
		switch err {
		case service.ErrEnrollNotFound:
			response.NotFound(c, "报名记录不存在")
		case service.ErrEnrollNotCancelable:
			response.BadRequest(c, "当前状态不可取消")
		default:
			response.InternalError(c, "取消报名失败")
		}
		return
	}

	response.Success(c, gin.H{"enrollment_id": id, "status": "CANCELLED"})
}

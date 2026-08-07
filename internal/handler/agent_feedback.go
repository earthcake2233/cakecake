package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
)

type postAgentFeedbackReq struct {
	MessageID uint64 `json:"message_id"`
	Feedback  string `json:"feedback"`
}

// PostAgentFeedback records or toggles a user's like/dislike on an AI message.
// PostAgentFeedback godoc
// @Summary      Feedback on an AI assistant message
// @Tags         DM
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /dm/agent/feedback [post]
func (a *API) PostAgentFeedback(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req postAgentFeedbackReq
	if err := c.ShouldBindJSON(&req); err != nil || req.MessageID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if a.Agent == nil {
		resp.Err(c, http.StatusServiceUnavailable, errcode.CodeInternalError)
		return
	}
	if err := a.Agent.SetMessageFeedback(c.Request.Context(), req.MessageID, uid, req.Feedback); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	resp.OK(c, gin.H{})
}

// AdminListAgentFeedbacks lists feedback rows with the rated message content.
// AdminListAgentFeedbacks godoc
// @Summary      List agent message feedback
// @Tags         Admin
// @Produce      json
// @Param        limit  query int false "page size"
// @Param        offset query int false "offset"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/agent-feedbacks [get]
func (a *API) AdminListAgentFeedbacks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	rows, err := a.Agent.ListAgentFeedbacksWithContent(c.Request.Context(), limit, offset)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"items": rows})
}

package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
)

// JSON writes the unified API response (Rule R-API-1).
func JSON(c *gin.Context, httpStatus int, code int, data interface{}) {
	if data == nil {
		data = nil
	}
	c.JSON(httpStatus, gin.H{
		"code": code,
		"msg":  errcode.GetMsg(code),
		"data": data,
	})
}

// OK writes 200 with code 0.
func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, errcode.CodeSuccess, data)
}

// Err writes a JSON body with non-zero business code. HTTP status may differ.
func Err(c *gin.Context, httpStatus int, code int) {
	JSON(c, httpStatus, code, nil)
}

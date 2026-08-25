package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parsePositiveID(c *gin.Context, param, label string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(param), 10, 32)
	if err != nil || value == 0 {
		BadRequest(c, "无效的"+label)
		return 0, false
	}
	return uint(value), true
}

func authenticatedUserID(c *gin.Context) (uint, bool) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return 0, false
	}
	return userID, true
}

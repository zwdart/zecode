package handler

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统信息类接口
type SystemHandler struct {
	Version   string
	BuildTime string
}

func NewSystemHandler(version, buildTime string) *SystemHandler {
	return &SystemHandler{Version: version, BuildTime: buildTime}
}

// Zing 健康检查
//
//	GET /zing
func (h *SystemHandler) Zing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
		"now":     time.Now(),
		"ip":      c.ClientIP(), // Gin 内置：自动解析 X-Forwarded-For / X-Real-IP
		"pid":     os.Getpid(),
		"time":    h.BuildTime,
		"git":     h.Version,
	})
}

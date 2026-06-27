package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/scanner"

	"github.com/gin-gonic/gin"
)

// ScannerHandler 库扫描 HTTP handler
type ScannerHandler struct {
	svc *scanner.Service
}

// NewScannerHandler 构造
func NewScannerHandler(svc *scanner.Service) *ScannerHandler {
	return &ScannerHandler{svc: svc}
}

// GetConfig 获取扫描配置（CMS）
func (h *ScannerHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetScanConfig(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": cfg})
}

// UpdateConfig 更新扫描配置（CMS）
func (h *ScannerHandler) UpdateConfig(c *gin.Context) {
	var req scanner.UpdateScanConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	cfg, err := h.svc.UpdateScanConfig(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": cfg})
}

// Scan 手动触发扫描
func (h *ScannerHandler) Scan(c *gin.Context) {
	root := c.Query("root")
	result, err := h.svc.RunScan(c.Request.Context(), root)
	if err != nil {
		respondError(c, apperr.Wrap(err, apperr.CodeInternal, "扫描失败"))
		return
	}
	c.JSON(200, gin.H{"data": result})
}

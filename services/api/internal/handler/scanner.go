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

// Scan 手动触发扫描
func (h *ScannerHandler) Scan(c *gin.Context) {
	root := c.Query("root")

	var result *scanner.ScanResult
	var err error
	if root != "" {
		result, err = h.svc.ScanRoot(c.Request.Context(), root)
	} else {
		result, err = h.svc.ScanAll(c.Request.Context())
	}

	if err != nil {
		respondError(c, apperr.Wrap(err, apperr.CodeInternal, "扫描失败"))
		return
	}
	c.JSON(200, gin.H{"data": result})
}

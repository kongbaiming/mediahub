package downloader

import (
	"encoding/base64"
	"strings"

	"github.com/mediahub/api/internal/apperr"
)

// NormalizeDownloadURL 将 thunder/qqdl 等封装链接解码为 qBittorrent 可用的 magnet/http URL
func NormalizeDownloadURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.BadRequest("URL 不能为空")
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "thunder://"):
		return decodeAAZZBase64(raw[len("thunder://"):])
	case strings.HasPrefix(lower, "qqdl://"):
		return decodeAAZZBase64(raw[len("qqdl://"):])
	default:
		return raw, nil
	}
}

// decodeAAZZBase64 迅雷 thunder/qqdl：Base64(AA + 真实URL + ZZ)
func decodeAAZZBase64(encoded string) (string, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return "", apperr.BadRequest("迅雷链接为空")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return "", apperr.BadRequest("迅雷链接 Base64 解码失败")
	}
	if len(data) < 4 {
		return "", apperr.BadRequest("迅雷链接格式无效")
	}

	inner := strings.TrimSpace(string(data[2 : len(data)-2]))
	if inner == "" {
		return "", apperr.BadRequest("迅雷链接内容为空")
	}

	// 极少数嵌套 thunder
	lower := strings.ToLower(inner)
	if strings.HasPrefix(lower, "thunder://") || strings.HasPrefix(lower, "qqdl://") {
		return NormalizeDownloadURL(inner)
	}

	return inner, nil
}

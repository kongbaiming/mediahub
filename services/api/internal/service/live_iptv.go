package service

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/live"
	"github.com/mediahub/api/pkg/httpclient"
)

const iptvProbeTimeout = 8 * time.Second

// validateIPTVSourceURL 校验 IPTV 源地址
func validateIPTVSourceURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.Validation("请填写 IPTV 流地址")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", apperr.Validation("IPTV 流地址格式无效")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", apperr.Validation("IPTV 流地址须为 http 或 https")
	}
	if !IsSafePublicURL(u) {
		return "", apperr.Validation("IPTV 流地址不允许访问内网地址")
	}
	return u.String(), nil
}

// IsSafePublicURL 是否允许反代的公网 URL（防 SSRF）
func IsSafePublicURL(u *url.URL) bool {
	return isSafeFetchURL(u)
}

func isSafeFetchURL(u *url.URL) bool {
	host := strings.Trim(u.Hostname(), "[]")
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	return true
}

// resolveMediaURL 将 playlist 内相对地址解析为绝对 URL
func resolveMediaURL(ref, base string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", apperr.Validation("空的媒体地址")
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, nil
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(refURL).String(), nil
}

// probeSourceURL 探测 IPTV 源是否可访问
func probeSourceURL(ctx context.Context, sourceURL string) bool {
	ctx, cancel := context.WithTimeout(ctx, iptvProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, sourceURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "MediaHub-Live-Proxy/1.0")
	resp, err := httpclient.External.Do(req)
	if err != nil {
		// 部分源不支持 HEAD，改用 GET 只读首字节
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return false
		}
		req.Header.Set("Range", "bytes=0-0")
		req.Header.Set("User-Agent", "MediaHub-Live-Proxy/1.0")
		resp, err = httpclient.External.Do(req)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// syncIPTVRooms 探测 IPTV 源并同步状态
func (s *LiveService) syncIPTVRooms(ctx context.Context, rooms []live.Room) {
	now := time.Now()
	for i := range rooms {
		room := &rooms[i]
		if !room.IsIPTV() || room.SourceURL == "" || room.Status == live.StatusEnded {
			continue
		}
		available := probeSourceURL(ctx, room.SourceURL)
		changed := false
		switch {
		case available && room.Status != live.StatusLive:
			room.Status = live.StatusLive
			if room.StartedAt == nil {
				room.StartedAt = &now
			}
			room.EndedAt = nil
			changed = true
		case !available && room.Status == live.StatusLive:
			room.Status = live.StatusIdle
			room.EndedAt = &now
			changed = true
		}
		if changed {
			_ = s.repo.Update(ctx, room)
		}
	}
}

// IPTVUpstreamProxyPrefix 生成 IPTV 子资源反代前缀
func IPTVUpstreamProxyPrefix(roomID string) string {
	return "/api/v1/live/rooms/" + roomID + "/upstream?u="
}

// RewriteIPTVM3U8 将 playlist 内媒体地址改写为 API 反代地址
func RewriteIPTVM3U8(content, roomID, sourceBase string) (string, error) {
	proxyPrefix := IPTVUpstreamProxyPrefix(roomID)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if uri := extractM3UAttribute(trimmed, "URI"); uri != "" {
				resolved, err := resolveMediaURL(uri, sourceBase)
				if err != nil {
					return "", err
				}
				u, err := url.Parse(resolved)
				if err != nil || !IsSafePublicURL(u) {
					continue
				}
				lines[i] = strings.Replace(trimmed, uri, proxyPrefix+url.QueryEscape(resolved), 1)
			}
			continue
		}
		resolved, err := resolveMediaURL(trimmed, sourceBase)
		if err != nil {
			return "", err
		}
		u, err := url.Parse(resolved)
		if err != nil || !IsSafePublicURL(u) {
			continue
		}
		lines[i] = proxyPrefix + url.QueryEscape(resolved)
	}
	return strings.Join(lines, "\n"), nil
}

func extractM3UAttribute(line, key string) string {
	prefix := key + "=\""
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(prefix):]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

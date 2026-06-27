package handler

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mediahub/api/internal/apperr"
)

// PathAlias 宿主机路径 → 容器内挂载路径（如 /volume1/media → /media）
type PathAlias struct {
	Prefix  string
	Replace string
}

// BuildPathAliases 根据 NAS 挂载配置生成路径别名
func BuildPathAliases(containerMedia, containerDownload, hostMedia, hostDownload string) []PathAlias {
	var out []PathAlias
	if p := aliasPair(hostMedia, containerMedia); p != nil {
		out = append(out, *p)
	}
	if p := aliasPair(hostDownload, containerDownload); p != nil {
		out = append(out, *p)
	}
	return out
}

func aliasPair(host, container string) *PathAlias {
	host = filepath.Clean(strings.TrimSpace(host))
	container = filepath.Clean(strings.TrimSpace(container))
	if host == "" || container == "" || host == container {
		return nil
	}
	return &PathAlias{Prefix: host, Replace: container}
}

// BuildPathAliasesFromEnv 读取 NAS_MEDIA_HOST / NAS_DOWNLOADS_HOST
func BuildPathAliasesFromEnv(containerMedia, containerDownload string) []PathAlias {
	return BuildPathAliases(
		containerMedia,
		containerDownload,
		os.Getenv("NAS_MEDIA_HOST"),
		os.Getenv("NAS_DOWNLOADS_HOST"),
	)
}

// ResolveMediaPath 将路径规范化为容器内可访问路径（导出供媒资校验使用）
func ResolveMediaPath(raw string, aliases []PathAlias) string {
	return resolveStreamPath(raw, aliases)
}

// ValidateStoragePath 校验存储路径是否在允许的根目录下，返回规范化后的路径
func ValidateStoragePath(raw string, allowedRoots []string, aliases []PathAlias) (string, error) {
	clean := resolveStreamPath(raw, aliases)
	if clean == "" || clean == "." {
		return "", apperr.Validation("storage_path 不能为空")
	}
	roots := normalizeAllowedRoots(allowedRoots)
	if !isPathUnderRoots(clean, roots) {
		return "", apperr.Validation("storage_path 必须在媒资或下载目录下")
	}
	return clean, nil
}

// resolveStreamPath 将请求路径规范化为容器内可访问路径
func resolveStreamPath(raw string, aliases []PathAlias) string {
	clean := filepath.Clean(strings.TrimSpace(raw))
	for _, a := range aliases {
		if a.Prefix == "" || a.Replace == "" {
			continue
		}
		if clean == a.Prefix || strings.HasPrefix(clean, a.Prefix+string(os.PathSeparator)) {
			rel := strings.TrimPrefix(clean, a.Prefix)
			rel = strings.TrimPrefix(rel, string(os.PathSeparator))
			if rel == "" {
				return a.Replace
			}
			return filepath.Join(a.Replace, rel)
		}
	}
	return clean
}

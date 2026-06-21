package mediafile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const minPlayableBytes = 1024 * 1024 // 1MB，过滤 qBit 占位/未完成文件

// IsPlayable 检查本地视频文件是否可读且头部有效
func IsPlayable(path string) (bool, string) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "文件不存在"
		}
		return false, "无法访问文件"
	}
	if info.IsDir() {
		return false, "路径是目录，请选择具体视频文件"
	}
	if info.Size() < minPlayableBytes {
		return false, fmt.Sprintf("文件过小（%d 字节），可能尚未下载完成", info.Size())
	}

	f, err := os.Open(path)
	if err != nil {
		return false, "无法打开文件"
	}
	defer f.Close()

	buf := make([]byte, 12)
	n, err := f.Read(buf)
	if err != nil || n < 4 {
		return false, "文件为空或已损坏"
	}

	// Matroska / WebM EBML
	if buf[0] == 0x1A && buf[1] == 0x45 && buf[2] == 0xDF && buf[3] == 0xA3 {
		return true, ""
	}
	// MP4 / MOV
	if n >= 8 && string(buf[4:8]) == "ftyp" {
		return true, ""
	}
	// AVI
	if n >= 12 && string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "AVI " {
		return true, ""
	}
	// MPEG-TS
	if buf[0] == 0x47 {
		return true, ""
	}

	return false, "不是有效的视频文件，可能仍在下载中或已损坏"
}

// ShouldSkipScan 扫描时跳过的路径（隐藏文件、qBit 临时文件等）
func ShouldSkipScan(path string) bool {
	base := filepath.Base(path)
	if base == "" || strings.HasPrefix(base, ".") {
		return true
	}
	lower := strings.ToLower(path)
	for _, suffix := range []string{".!qb", ".part", ".downloading", ".tmp"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

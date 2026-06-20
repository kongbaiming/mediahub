package handler

import "strconv"

// atoi 字符串转 int，失败返回默认值
func atoi(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

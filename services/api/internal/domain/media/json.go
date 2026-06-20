package media

import "encoding/json"

// 避免在 model.go 里直接 import encoding/json（保持简洁），单独文件放辅助

func jsonMarshal(v any) ([]byte, error)     { return json.Marshal(v) }
func jsonUnmarshal(b []byte, v any) error   { return json.Unmarshal(b, v) }

package layout

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

func marshalJSON(v any) (driver.Value, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func unmarshalJSON(dst any, src any) error {
	if src == nil {
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("layout: unsupported json scan type %T", src)
	}
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return json.Unmarshal(b, dst)
}

// Value 写入 jsonb
func (c LayoutConfig) Value() (driver.Value, error) {
	if c.Rows == nil {
		c.Rows = []Row{}
	}
	return marshalJSON(c)
}

// Scan 从 jsonb 读取
func (c *LayoutConfig) Scan(src any) error {
	if src == nil {
		*c = LayoutConfig{Rows: []Row{}}
		return nil
	}
	if err := unmarshalJSON(c, src); err != nil {
		return err
	}
	if c.Rows == nil {
		c.Rows = []Row{}
	}
	return nil
}

// TrafficSplit AB 测试权重
type TrafficSplit map[string]int

// Value 写入 jsonb
func (t TrafficSplit) Value() (driver.Value, error) {
	if len(t) == 0 {
		return nil, nil
	}
	return marshalJSON(map[string]int(t))
}

// Scan 从 jsonb 读取
func (t *TrafficSplit) Scan(src any) error {
	if src == nil {
		*t = nil
		return nil
	}
	var m map[string]int
	if err := unmarshalJSON(&m, src); err != nil {
		return err
	}
	*t = TrafficSplit(m)
	return nil
}

// Value 写入 jsonb
func (d DynamicRules) Value() (driver.Value, error) {
	return marshalJSON(d)
}

// Scan 从 jsonb 读取
func (d *DynamicRules) Scan(src any) error {
	if src == nil {
		return nil
	}
	return unmarshalJSON(d, src)
}

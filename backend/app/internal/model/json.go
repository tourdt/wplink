package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JSONMap map[string]interface{}
type JSONStringSlice []string

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("不支持的 JSONB 数据类型")
	}

	if len(bytes) == 0 {
		*m = JSONMap{}
		return nil
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		return err
	}
	*m = JSONMap(decoded)
	return nil
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]interface{}(m))
}

func (s *JSONStringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = JSONStringSlice{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("不支持的 JSONB 数组数据类型")
	}

	if len(bytes) == 0 {
		*s = JSONStringSlice{}
		return nil
	}

	var decoded []string
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		return err
	}
	*s = JSONStringSlice(decoded)
	return nil
}

func (s JSONStringSlice) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(s))
}

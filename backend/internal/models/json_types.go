package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONMap and JSONList implement sql.Scanner + driver.Valuer so we can read
// and write MySQL JSON columns directly as Go maps/slices without a full
// ORM. Both marshal to `null`/`{}`/`[]` cleanly for JSON API responses.

type JSONMap map[string]interface{}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("JSONMap: unsupported scan type")
	}
	if len(bytes) == 0 {
		*m = JSONMap{}
		return nil
	}
	return json.Unmarshal(bytes, m)
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

type JSONList []interface{}

func (l *JSONList) Scan(value interface{}) error {
	if value == nil {
		*l = JSONList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("JSONList: unsupported scan type")
	}
	if len(bytes) == 0 {
		*l = JSONList{}
		return nil
	}
	return json.Unmarshal(bytes, l)
}

func (l JSONList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal(l)
	return string(b), err
}

// StringList is a convenience for JSON string arrays (e.g. technologies)
// with a typed Go representation for services that want []string directly.
type StringList []string

func (l *StringList) Scan(value interface{}) error {
	if value == nil {
		*l = StringList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("StringList: unsupported scan type")
	}
	if len(bytes) == 0 {
		*l = StringList{}
		return nil
	}
	return json.Unmarshal(bytes, l)
}

func (l StringList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal(l)
	return string(b), err
}

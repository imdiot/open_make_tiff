package exiftool

import (
	"errors"
	"fmt"
	"strconv"
)

// ErrKeyNotFound is returned when a requested key does not exist.
var ErrKeyNotFound = errors.New("key not found")

// Metadata represents file metadata extracted by exiftool.
type Metadata struct {
	File   string
	Fields map[string]any
}

// NewMetadata creates an empty Metadata instance.
func NewMetadata(file string) Metadata {
	return Metadata{
		File:   file,
		Fields: make(map[string]any),
	}
}

// GetString returns a field as string. Returns ErrKeyNotFound if missing.
func (m Metadata) GetString(k string) (string, error) {
	v, found := m.Fields[k]
	if !found || v == nil {
		return "", ErrKeyNotFound
	}
	return toString(v), nil
}

// GetInt returns a field as int64. Returns ErrKeyNotFound if missing.
func (m Metadata) GetInt(k string) (int64, error) {
	v, found := m.Fields[k]
	if !found || v == nil {
		return 0, ErrKeyNotFound
	}
	switch v := v.(type) {
	case string:
		return strconv.ParseInt(v, 10, 64)
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64)
	}
}

// GetFloat returns a field as float64. Returns ErrKeyNotFound if missing.
func (m Metadata) GetFloat(k string) (float64, error) {
	v, found := m.Fields[k]
	if !found || v == nil {
		return 0, ErrKeyNotFound
	}
	switch v := v.(type) {
	case string:
		return strconv.ParseFloat(v, 64)
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	default:
		return strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
	}
}

// GetStrings returns a field as []string. Returns ErrKeyNotFound if missing.
func (m Metadata) GetStrings(k string) ([]string, error) {
	v, found := m.Fields[k]
	if !found || v == nil {
		return nil, ErrKeyNotFound
	}
	switch v := v.(type) {
	case []any:
		res := make([]string, len(v))
		for i, item := range v {
			res[i] = toString(item)
		}
		return res, nil
	default:
		return []string{toString(v)}, nil
	}
}

// SetString sets a string field.
func (m *Metadata) SetString(k string, v string) {
	m.Fields[k] = v
}

// SetInt sets an integer field.
func (m *Metadata) SetInt(k string, v int64) {
	m.Fields[k] = v
}

// SetFloat sets a float field.
func (m *Metadata) SetFloat(k string, v float64) {
	m.Fields[k] = v
}

// SetStrings sets a string list field.
func (m *Metadata) SetStrings(k string, v []string) {
	t := make([]any, len(v))
	for i, s := range v {
		t[i] = s
	}
	m.Fields[k] = t
}

// Clear deletes a field (nil value means delete on write).
func (m *Metadata) Clear(k string) {
	m.Fields[k] = nil
}

func toString(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

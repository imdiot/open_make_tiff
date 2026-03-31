package exiftool

import (
	"errors"
	"testing"
)

func TestMetadataGetString(t *testing.T) {
	m := NewMetadata("test.jpg")
	m.Fields["Title"] = "Hello"
	m.Fields["Count"] = float64(42)
	m.Fields["Nil"] = nil

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr error
	}{
		{"string value", "Title", "Hello", nil},
		{"float value", "Count", "42", nil},
		{"missing key", "Missing", "", ErrKeyNotFound},
		{"nil value", "Nil", "", ErrKeyNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.GetString(tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetString(%q) error = %v, want %v", tt.key, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("GetString(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestMetadataGetInt(t *testing.T) {
	m := NewMetadata("test.jpg")
	m.Fields["Count"] = float64(42)
	m.Fields["StrNum"] = "100"
	m.Fields["IntVal"] = int64(7)

	tests := []struct {
		name    string
		key     string
		want    int64
		wantErr error
	}{
		{"float value", "Count", 42, nil},
		{"string value", "StrNum", 100, nil},
		{"int64 value", "IntVal", 7, nil},
		{"missing key", "Missing", 0, ErrKeyNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.GetInt(tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetInt(%q) error = %v, want %v", tt.key, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("GetInt(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

func TestMetadataGetFloat(t *testing.T) {
	m := NewMetadata("test.jpg")
	m.Fields["Ratio"] = float64(1.5)
	m.Fields["StrFloat"] = "3.14"

	got, err := m.GetFloat("Ratio")
	if err != nil || got != 1.5 {
		t.Errorf("GetFloat(Ratio) = %v, %v, want 1.5, nil", got, err)
	}

	got, err = m.GetFloat("StrFloat")
	if err != nil || got != 3.14 {
		t.Errorf("GetFloat(StrFloat) = %v, %v, want 3.14, nil", got, err)
	}

	_, err = m.GetFloat("Missing")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("GetFloat(Missing) error = %v, want ErrKeyNotFound", err)
	}
}

func TestMetadataGetStrings(t *testing.T) {
	m := NewMetadata("test.jpg")
	m.Fields["Tags"] = []interface{}{"a", "b", "c"}
	m.Fields["Single"] = "only"

	got, err := m.GetStrings("Tags")
	if err != nil || len(got) != 3 {
		t.Errorf("GetStrings(Tags) = %v, %v", got, err)
	}

	got, err = m.GetStrings("Single")
	if err != nil || len(got) != 1 || got[0] != "only" {
		t.Errorf("GetStrings(Single) = %v, %v", got, err)
	}

	_, err = m.GetStrings("Missing")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("GetStrings(Missing) error = %v, want ErrKeyNotFound", err)
	}
}

func TestMetadataSetters(t *testing.T) {
	m := NewMetadata("test.jpg")

	m.SetString("Title", "hello")
	got, _ := m.GetString("Title")
	if got != "hello" {
		t.Errorf("SetString failed, got %q", got)
	}

	m.SetInt("Count", 42)
	gotInt, _ := m.GetInt("Count")
	if gotInt != 42 {
		t.Errorf("SetInt failed, got %d", gotInt)
	}

	m.SetFloat("Ratio", 1.5)
	gotFloat, _ := m.GetFloat("Ratio")
	if gotFloat != 1.5 {
		t.Errorf("SetFloat failed, got %v", gotFloat)
	}

	m.SetStrings("Tags", []string{"x", "y"})
	gotTags, _ := m.GetStrings("Tags")
	if len(gotTags) != 2 {
		t.Errorf("SetStrings failed, got %v", gotTags)
	}

	m.Clear("Title")
	_, err := m.GetString("Title")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("Clear failed, err = %v, want ErrKeyNotFound", err)
	}
}

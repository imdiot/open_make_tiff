// Package build holds product metadata parsed from wails.json, shared by the
// CLI and GUI entry points so main does not have to unpack individual fields.
package build

import "encoding/json"

// Info is the product metadata under wails.json's "info" key.
type Info struct {
	ProductName    string `json:"productName"`
	ProductVersion string `json:"productVersion"`
}

// Parse decodes wails.json bytes and returns the embedded Info.
func Parse(data []byte) (Info, error) {
	var c struct{ Info Info `json:"info"` }
	err := json.Unmarshal(data, &c)
	return c.Info, err
}

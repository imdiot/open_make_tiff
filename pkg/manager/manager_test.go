package manager

import "testing"

func TestConfig_load(t *testing.T) {
	cfg := newConfig()
	t.Log(cfg.load())
}

package handlers

import (
	"testing"

	"cube/internal/config"
)

func TestConfigGet(t *testing.T) {
	env := newTestEnv(t)
	var got config.Config
	decodeData(t, getJSON(t, env.url("/api/config")), &got)

	if got.DataDir != env.ws.Cfg.DataDir {
		t.Errorf("dataDir 不符, got %q", got.DataDir)
	}
}

package agenthook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInstallZcodeEnablesHookRunnerAndNestsEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	unmarked := "/opt/roborev agent-hook run --agent zcode"
	unowned := "/opt/user-hook agent-hook run --agent zcode"
	fixture, err := json.Marshal(map[string]any{
		"plugins": map[string]any{"enabledPlugins": map[string]any{"demo": true}},
		"hooks": map[string]any{
			"enabled": false,
			"events": map[string]any{"Stop": []any{map[string]any{
				"hooks": []any{
					map[string]any{"type": "command", "command": unmarked},
					map[string]any{"type": "command", "command": unowned},
				},
			}}},
		},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, fixture, 0o600))
	opts := InstallOptions{
		Agent: "zcode", Command: unmarked,
		ConfigPath: path, Timeout: 10 * time.Second,
	}
	var first, second bytes.Buffer

	require.NoError(t, RunInstall(opts, &first))
	require.NoError(t, RunInstall(opts, &second))

	assert.Contains(t, first.String(), "installed ZCode agent hooks")
	assert.Contains(t, second.String(), "ZCode agent hooks already installed")
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(body, &root))
	plugins, ok := root["plugins"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, plugins["enabledPlugins"], "demo")
	hooks, ok := root["hooks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, hooks["enabled"])
	events, ok := hooks["events"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, events, 3)
	assert.Contains(t, string(body), ZcodeShellMatcher)
	assert.Contains(t, string(body), agentHookMarker)
	assert.Contains(t, string(body), unmarked)
	assert.Contains(t, string(body), unowned)
	installed := 0
	for _, rawEntries := range events {
		for _, rawEntry := range rawEntries.([]any) {
			for _, rawHandler := range rawEntry.(map[string]any)["hooks"].([]any) {
				command := rawHandler.(map[string]any)["command"].(string)
				if strings.Contains(command, agentHookMarker) {
					installed++
				}
			}
		}
	}
	assert.Equal(t, 3, installed)
}

func TestRunDumpZcodeDoesNotWriteSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	source := []byte(`{"custom":"preserved"}`)
	require.NoError(t, os.WriteFile(path, source, 0o600))
	var stdout bytes.Buffer

	err := RunDump(InstallOptions{
		Agent: "zcode", Command: "/opt/roborev agent-hook run --agent zcode",
		ConfigPath: path, Timeout: 10 * time.Second,
	}, &stdout)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), `"custom": "preserved"`)
	assert.Contains(t, stdout.String(), "agent-hook run --agent zcode")
	assert.Contains(t, stdout.String(), `"enabled": true`)
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, source, unchanged)
}

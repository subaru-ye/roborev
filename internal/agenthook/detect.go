package agenthook

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"slices"
	"strings"

	kitagenthook "go.kenn.io/kit/agenthook"
)

func Installed(agent kitagenthook.Agent, path string) (bool, error) {
	result, err := kitagenthook.PlanUninstall(agent, path, agentHookMarker)
	if err != nil {
		return false, err
	}
	return result.Changed, nil
}

func InstalledForAgent(path, agent string) (bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(agent))
	if _, ok := localProfile(normalized); ok {
		body, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		var root any
		if err := json.Unmarshal(body, &root); err != nil {
			return false, err
		}
		return containsOwnedHook(root, kitagenthook.Agent(normalized)), nil
	}
	profile, err := kitagenthook.ParseAgent(agent)
	if err != nil {
		return false, fmt.Errorf("unsupported agent %q", agent)
	}
	return Installed(profile, path)
}

// containsOwnedHook scans a parsed hook configuration for a roborev-owned
// hook command selecting agent, regardless of the nesting level. Local
// profiles keep hooks at harness-specific depths (ZCode nests event arrays
// under hooks.events), so a recursive search avoids duplicating each shape.
func containsOwnedHook(value any, agent kitagenthook.Agent) bool {
	switch typed := value.(type) {
	case []any:
		return slices.ContainsFunc(typed, func(child any) bool {
			return containsOwnedHook(child, agent)
		})
	case map[string]any:
		if command, ok := typed["command"].(string); ok && isOwnedHookCommand(command, agent) {
			return true
		}
		for _, child := range typed {
			if containsOwnedHook(child, agent) {
				return true
			}
		}
	}
	return false
}

func isOwnedHookCommand(command string, agent kitagenthook.Agent) bool {
	selected, err := commandAgent(command)
	return err == nil && selected == agent
}

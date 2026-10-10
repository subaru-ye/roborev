package agenthook

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kitagenthook "go.kenn.io/kit/agenthook"
)

// ZCode matches tool events against the Bash tool name exactly like Claude
// Code; its matcher is a case-sensitive regular expression.
const ZcodeShellMatcher = "Bash"

// DefaultZcodeHooksPath returns ZCode's user-scope configuration file. Hook
// registrations live under the top-level hooks key as
// hooks.events.<Event>, and hooks must also set enabled: true because
// configuration-file hooks are disabled by default.
func DefaultZcodeHooksPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".zcode", "cli", "config.json")
}

func planZcodeInstall(opts InstallOptions) (kitagenthook.Result, error) {
	path := strings.TrimSpace(opts.ConfigPath)
	if path == "" {
		path = DefaultZcodeHooksPath()
	}
	if path == "" {
		return kitagenthook.Result{}, errors.New("could not resolve ZCode hook config path")
	}
	command, err := hookRunCommand(AgentZcode, opts)
	if err != nil {
		return kitagenthook.Result{}, err
	}
	root, err := readHooksConfig(path, "ZCode")
	if err != nil {
		return kitagenthook.Result{}, err
	}
	before, err := marshalHooksConfig(root)
	if err != nil {
		return kitagenthook.Result{}, fmt.Errorf("encode existing ZCode hook config %s: %w", path, err)
	}
	hooks, err := hooksObject(root, path, "ZCode")
	if err != nil {
		return kitagenthook.Result{}, err
	}
	// Configuration-file hooks never run while enabled is false, so the
	// install must opt in to the hook runner itself.
	hooks["enabled"] = true
	events, err := objectField(hooks, "events", path, "ZCode")
	if err != nil {
		return kitagenthook.Result{}, err
	}
	if err := removeOwnedHooks(events, path, "ZCode"); err != nil {
		return kitagenthook.Result{}, err
	}
	if err := appendHookEntries(events, path, "ZCode", command, opts.Timeout, []hookSpec{
		{event: "PreToolUse", matcher: ZcodeShellMatcher},
		{event: "PostToolUse", matcher: ZcodeShellMatcher},
		{event: "Stop"},
	}); err != nil {
		return kitagenthook.Result{}, err
	}
	after, err := marshalHooksConfig(root)
	if err != nil {
		return kitagenthook.Result{}, fmt.Errorf("encode ZCode hook config %s: %w", path, err)
	}
	return kitagenthook.Result{
		Agent: AgentZcode, ConfigPath: path, Changed: !bytes.Equal(before, after), Data: after,
	}, nil
}

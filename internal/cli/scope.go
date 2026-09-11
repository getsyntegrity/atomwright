package cli

import (
	"fmt"
	"github.com/gentleman-programming/gentle-ai/v2/internal/envcompat"
	"github.com/gentleman-programming/gentle-ai/v2/internal/identity"
	"strings"
)

// InstallScope controls where agent-scoped config files (system prompts, skills/, agents/, etc.) are written.
// ScopeGlobal writes to the user's global config root for each selected agent.
// ScopeWorkspace writes to the current workspace config root for each selected agent.
type InstallScope string

const (
	// ScopeGlobal writes to the global agent config dir (default, backward-compatible).
	ScopeGlobal InstallScope = "global"
	// ScopeWorkspace writes to the current workspace config root for each selected agent.
	ScopeWorkspace InstallScope = "workspace"

	// scopeEnvSuffix is the unprefixed variable that controls install scope;
	// envcompat decides which prefix answers.
	scopeEnvSuffix = "INSTALL_SCOPE"
)

// scopeEnvVar is the current, fully prefixed variable name, used where the name
// is shown to a user or set by a test.
var scopeEnvVar = identity.EnvPrefix() + scopeEnvSuffix

// ResolveInstallScope resolves the install scope from the flag value and env var.
// Priority: explicit flag > env var > default (global).
// An empty flagValue means the flag was not set.
func ResolveInstallScope(flagValue string) (InstallScope, error) {
	raw := strings.TrimSpace(flagValue)
	if raw == "" {
		value, _, _ := envcompat.Lookup(scopeEnvSuffix)
		raw = strings.TrimSpace(value)
	}
	if raw == "" {
		return ScopeGlobal, nil
	}
	return parseInstallScope(raw)
}

func parseInstallScope(raw string) (InstallScope, error) {
	switch InstallScope(raw) {
	case ScopeGlobal, ScopeWorkspace:
		return InstallScope(raw), nil
	default:
		return "", fmt.Errorf("unsupported scope %q (valid: global, workspace)", raw)
	}
}

// ResolveAgentConfigDir returns the directory to use as the agent config root.
// When scope is ScopeWorkspace, workspaceDir is returned; otherwise homeDir is used.
// Both homeDir and workspaceDir must be non-empty absolute paths.
func ResolveAgentConfigDir(scope InstallScope, homeDir, workspaceDir string) string {
	if scope == ScopeWorkspace && strings.TrimSpace(workspaceDir) != "" {
		return workspaceDir
	}
	return homeDir
}

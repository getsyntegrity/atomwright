package cli

import (
	"fmt"
	"github.com/gentleman-programming/gentle-ai/v2/internal/envcompat"
	"github.com/gentleman-programming/gentle-ai/v2/internal/identity"
	"strings"
)

type InstallChannel string

const (
	ChannelStable InstallChannel = "stable"
	ChannelBeta   InstallChannel = "beta"

	// channelEnvSuffix is the unprefixed variable name; envcompat decides which
	// prefix answers so the inherited GENTLE_AI_ one keeps working.
	channelEnvSuffix = "CHANNEL"
)

// channelEnvVar is the current, fully prefixed variable name, used where the
// name is shown to a user or set by a test.
var channelEnvVar = identity.EnvPrefix() + channelEnvSuffix

func ResolveInstallChannel(flagValue string) (InstallChannel, error) {
	raw := strings.TrimSpace(flagValue)
	if raw == "" {
		value, _, _ := envcompat.Lookup(channelEnvSuffix)
		raw = strings.TrimSpace(value)
	}
	if raw == "" {
		return ChannelStable, nil
	}

	switch InstallChannel(strings.ToLower(raw)) {
	case ChannelStable:
		return ChannelStable, nil
	case ChannelBeta, "nightly":
		return ChannelBeta, nil
	default:
		// refusal:by-design operator-knowledge: only the operator knows which channel they meant; the message already states the complete next action (use stable, beta, or nightly), and no runnable command can pick it for them
		return "", fmt.Errorf("unsupported Gentle AI channel %q (use stable, beta, or nightly)", raw)
	}
}

func (c InstallChannel) IsBeta() bool {
	return c == ChannelBeta
}

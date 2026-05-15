package cosmos_test

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
)

const nobleMainnetRPC = "rpc.noble.strange.love"

// TestUsedNonce hits Noble mainnet directly. Skip when DNS/network is
// unavailable (sandboxed dev containers, offline CI) instead of failing —
// the test is verifying our RPC client wiring, not the operator's network.
func TestUsedNonce(t *testing.T) {
	if _, err := net.LookupHost(nobleMainnetRPC); err != nil {
		t.Skipf("skipping: cannot resolve %s (no network access?): %v", nobleMainnetRPC, err)
	}

	cc, err := cosmos.NewProvider("https://" + nobleMainnetRPC + ":443")
	require.NoError(t, err)

	used, err := cc.QueryUsedNonce(context.TODO(), 0, 15365)
	if err != nil && (strings.Contains(err.Error(), "no such host") ||
		strings.Contains(err.Error(), "i/o timeout") ||
		strings.Contains(err.Error(), "connection refused")) {
		t.Skipf("skipping: %s unreachable: %v", nobleMainnetRPC, err)
	}
	require.NoError(t, err)
	require.True(t, used)

	used, err = cc.QueryUsedNonce(context.TODO(), 0, 100)
	require.NoError(t, err)
	require.False(t, used)
}

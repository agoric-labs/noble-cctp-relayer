package testutil

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var logger log.Logger
var EnvFile = os.ExpandEnv("$GOPATH/src/github.com/strangelove-ventures/noble-cctp-relayer/.env")

// EnvLoaded reports whether the .env file (containing live RPC endpoints +
// minter keys for integration-style unit tests) was found at package-init
// time. Tests that need those env vars should skip when this is false so
// the rest of the package can still run in environments without secrets.
var EnvLoaded bool

func init() {
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))

	if err := godotenv.Load(EnvFile); err != nil {
		logger.Error("env file not loaded — env-dependent tests will skip",
			"file", EnvFile, "err", err)
		return
	}
	EnvLoaded = true
}

func ConfigSetup(t *testing.T) (a *cmd.AppState, registeredDomains map[types.Domain]types.Chain) {
	t.Helper()

	if !EnvLoaded {
		t.Skipf("skipping: .env not present at %s (provide NOBLE_RPC, SEPOLIA_RPC, SEPOLIA_WS to run)", EnvFile)
	}

	var testConfig = types.Config{
		Chains: map[string]types.ChainConfig{
			"noble": &noble.ChainConfig{
				ChainID: "grand-1",
				RPC:     os.Getenv("NOBLE_RPC"),
			},
			"ethereum": &ethereum.ChainConfig{
				ChainID:          11155111,
				Domain:           types.Domain(0),
				MinterPrivateKey: "1111111111111111111111111111111111111111111111111111111111111111",
				RPC:              os.Getenv("SEPOLIA_RPC"),
				WS:               os.Getenv("SEPOLIA_WS"),
			},
		},
		Circle: types.CircleSettings{
			AttestationBaseURL: "https://iris-api-sandbox.circle.com/attestations/",
			FetchRetries:       0,
			FetchRetryInterval: 3,
		},

		EnabledRoutes: map[types.Domain][]types.Domain{
			0: {4},
			4: {0},
		},
	}

	a = cmd.NewAppState()
	a.LogLevel = "debug"
	a.InitLogger()
	a.Config = &testConfig

	registeredDomains = make(map[types.Domain]types.Chain)
	for name, cfgg := range a.Config.Chains {
		c, err := cfgg.Chain(name, false)
		require.NoError(t, err, "Error creating chain")

		registeredDomains[c.Domain()] = c
	}

	return a, registeredDomains
}

package ethereum

import (
	"fmt"
	"os"
	"strings"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.ChainConfig = (*ChainConfig)(nil)

type ChainConfig struct {
	BroadcastRetries       int   `yaml:"broadcast-retries"`
	BroadcastRetryInterval int   `yaml:"broadcast-retry-interval"`
	ChainID                int64 `yaml:"chain-id"`
	Domain                 types.Domain
	// Optional fixed gas limit for broadcast txs. 0 means "let go-ethereum
	// call eth_estimateGas". Setting this is recommended when fee caps are
	// pinned, because some RPC providers (Alchemy/Base) enforce a balance
	// check during eth_estimateGas using `fee-cap × block_gas_limit`, which
	// reserves orders of magnitude more than the call actually needs.
	GasLimit             uint64 `yaml:"gas-limit"`
	LookbackPeriod       uint64 `yaml:"lookback-period"`
	MaxFeePerGas         uint64 `yaml:"max-fee-per-gas"`
	MaxPriorityFeePerGas uint64 `yaml:"max-priority-fee-per-gas"`
	MessageTransmitter   string `yaml:"message-transmitter"`
	MetricsDenom         string `yaml:"metrics-denom"`
	MetricsExponent      int    `yaml:"metrics-exponent"`
	MinMintAmount        uint64 `yaml:"min-mint-amount"`
	MinterPrivateKey     string `yaml:"minter-private-key"`
	RPC                  string `yaml:"rpc"`
	StartBlock           uint64 `yaml:"start-block"`
	WS                   string `yaml:"ws"`
}

func (c *ChainConfig) Chain(name string, watchOnly bool) (types.Chain, error) {
	envKey := strings.ToUpper(name) + "_PRIV_KEY"
	privKey := os.Getenv(envKey)

	switch {
	case watchOnly:
		if len(privKey) != 0 {
			c.MinterPrivateKey = privKey
		}
	default:
		if len(c.MinterPrivateKey) == 0 || len(privKey) != 0 {
			if len(privKey) == 0 {
				return nil, fmt.Errorf("env variable %s is empty, priv key not found for chain %s", envKey, name)
			}
			c.MinterPrivateKey = privKey
		}
	}

	return NewChain(
		name,
		c.Domain,
		c.ChainID,
		c.RPC,
		c.WS,
		c.MessageTransmitter,
		c.StartBlock,
		c.LookbackPeriod,
		c.MinterPrivateKey,
		c.BroadcastRetries,
		c.BroadcastRetryInterval,
		c.MinMintAmount,
		c.MetricsDenom,
		c.MetricsExponent,
		c.MaxFeePerGas,
		c.MaxPriorityFeePerGas,
		c.GasLimit,
	)
}

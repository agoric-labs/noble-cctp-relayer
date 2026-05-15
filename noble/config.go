package noble

import (
	"fmt"
	"os"
	"strings"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.ChainConfig = (*ChainConfig)(nil)

const defaultBlockQueueChannelSize = 1000000

type ChainConfig struct {
	BlockQueueChannelSize  uint64 `yaml:"block-queue-channel-size"`
	BroadcastRetries       int    `yaml:"broadcast-retries"`
	BroadcastRetryInterval int    `yaml:"broadcast-retry-interval"`
	ChainID                string `yaml:"chain-id"`
	GasLimit               uint64 `yaml:"gas-limit"`
	LookbackPeriod         uint64 `yaml:"lookback-period"`
	MinMintAmount          uint64 `yaml:"min-mint-amount"`
	MinterPrivateKey       string `yaml:"minter-private-key"`
	RPC                    string `yaml:"rpc"`
	StartBlock             uint64 `yaml:"start-block"`
	TxMemo                 string `yaml:"tx-memo"`
	Workers                uint32 `yaml:"workers"`
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
		c.RPC,
		c.ChainID,
		c.MinterPrivateKey,
		c.StartBlock,
		c.LookbackPeriod,
		c.Workers,
		c.GasLimit,
		c.TxMemo,
		c.BroadcastRetries,
		c.BroadcastRetryInterval,
		c.BlockQueueChannelSize,
		c.MinMintAmount,
	)
}

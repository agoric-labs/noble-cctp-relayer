package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)

// Message defines ...
// https://github.com/circlefin/evm-cctp-contracts/blob/d53f0e1937a0a5c5158d356b6767b77dc32dcc90/src/messages/Message.sol#L29-L37
type Message struct {
	Version           uint32
	SourceDomain      uint32
	DestinationDomain uint32
	Nonce             uint64
	Sender            []byte
	Recipient         []byte
	DestinationCaller []byte
	MessageBody       []byte
}

// ErrNotBurnMessage signals that a parsed CCTP `MessageSent` payload is not a
// USDC burn (likely a non-burn application of the shared MessageTransmitter,
// e.g. Hyperlane). Callers should treat this as "skip silently" rather than a
// real parse failure.
var ErrNotBurnMessage = errors.New("not a CCTP burn message")

// BurnMessage defines ...
// https://github.com/circlefin/evm-cctp-contracts/blob/d53f0e1937a0a5c5158d356b6767b77dc32dcc90/src/messages/BurnMessage.sol#L24-L29
type BurnMessage struct {
	Version       uint32
	BurnToken     []byte
	MintRecipient []byte
	Amount        *big.Int
	MessageSender []byte
}

// MetadataMessage defines ...
type MetadataMessage struct {
	Nonce     uint64
	Sender    []byte
	Channel   uint64
	Prefix    string
	Recipient []byte
	Memo      string
}

// Parse decodes a CCTP MessageSent envelope. Auto-detects v1 vs v2 from the
// 4-byte leading version field: v1 has a 116-byte header, v2 has a 148-byte
// header (the nonce expands from uint64 to bytes32 and two finality fields
// are appended). The shared sender/recipient/destCaller layout is otherwise
// identical, just at different offsets.
func (msg *Message) Parse(bz []byte) (*Message, error) {
	const (
		VersionIndex           = 0
		SourceDomainIndex      = 4
		DestinationDomainIndex = 8
		NonceIndex             = 12
	)

	if len(bz) < SourceDomainIndex {
		return nil, fmt.Errorf("CCTP Message too short: %d bytes", len(bz))
	}
	msg.Version = binary.BigEndian.Uint32(bz[VersionIndex:SourceDomainIndex])

	switch msg.Version {
	case 0:
		const (
			SenderIndex            = 20
			RecipientIndex         = 52
			DestinationCallerIndex = 84
			MessageBodyIndex       = 116
		)

		if len(bz) < MessageBodyIndex {
			return nil, fmt.Errorf("v1 Message header too short: got %d bytes, need %d", len(bz), MessageBodyIndex)
		}

		msg.SourceDomain = binary.BigEndian.Uint32(bz[SourceDomainIndex:DestinationDomainIndex])
		msg.DestinationDomain = binary.BigEndian.Uint32(bz[DestinationDomainIndex:NonceIndex])
		msg.Nonce = binary.BigEndian.Uint64(bz[NonceIndex:SenderIndex])
		msg.Sender = bz[SenderIndex:RecipientIndex]
		msg.Recipient = bz[RecipientIndex:DestinationCallerIndex]
		msg.DestinationCaller = bz[DestinationCallerIndex:MessageBodyIndex]
		msg.MessageBody = bz[MessageBodyIndex:]
	case 1:
		// v2 layout:
		//   0-3   version                     uint32
		//   4-7   sourceDomain                uint32
		//   8-11  destinationDomain           uint32
		//   12-43 nonce                       bytes32  (typically zero; identity is by hash)
		//   44-75 sender                      bytes32
		//   76-107 recipient                  bytes32
		//   108-139 destinationCaller         bytes32
		//   140-143 minFinalityThreshold      uint32
		//   144-147 finalityThresholdExecuted uint32
		//   148+  messageBody                 bytes
		const (
			SenderIndex               = 44
			RecipientIndex            = 76
			DestinationCallerIndex    = 108
			MinFinalityThresholdIndex = 140
			MessageBodyIndex          = 148
		)

		if len(bz) < MessageBodyIndex {
			return nil, fmt.Errorf("v2 Message header too short: got %d bytes, need %d", len(bz), MessageBodyIndex)
		}

		msg.SourceDomain = binary.BigEndian.Uint32(bz[SourceDomainIndex:DestinationDomainIndex])
		msg.DestinationDomain = binary.BigEndian.Uint32(bz[DestinationDomainIndex:NonceIndex])
		// v2 nonce is bytes32; squeeze the low 8 bytes into our uint64 field.
		msg.Nonce = binary.BigEndian.Uint64(bz[SenderIndex-8 : SenderIndex])
		msg.Sender = bz[SenderIndex:RecipientIndex]
		msg.Recipient = bz[RecipientIndex:DestinationCallerIndex]
		msg.DestinationCaller = bz[DestinationCallerIndex:MinFinalityThresholdIndex]
		msg.MessageBody = bz[MessageBodyIndex:]
	default:
		return nil, fmt.Errorf("unsupported CCTP message version=%d", msg.Version)
	}

	return msg, nil
}

// Parse decodes a CCTP BurnMessage body. The first 132 bytes are identical in
// v1 and v2; v2 bodies are >= 228 bytes (trailing maxFee/feeExecuted/
// expirationBlock/hookData fields are ignored — we only need the first 132).
// We dispatch on body length, not on the embedded version byte, because v2
// TokenMessenger emits BurnMessage bodies with the version byte still set to 0.
func (c *BurnMessage) Parse(bz []byte) (*BurnMessage, error) {
	const (
		VersionIndex       = 0
		BurnTokenIndex     = 4
		MintRecipientIndex = 36
		AmountIndex        = 68
		MsgSenderIndex     = 100
		BurnContentLength  = 132
	)

	switch {
	case len(bz) == 132:
		// v1 burn body
	case len(bz) > 227:
		// v2 burn body
	default:
		return nil, fmt.Errorf("%w: BurnMessage length %d does not match v1 (=132) or v2 (>=228)", ErrNotBurnMessage, len(bz))
	}

	c.Version = binary.BigEndian.Uint32(bz[VersionIndex:BurnTokenIndex])
	c.BurnToken = bz[BurnTokenIndex:MintRecipientIndex]
	c.MintRecipient = bz[MintRecipientIndex:AmountIndex]
	c.Amount = new(big.Int).SetBytes(bz[AmountIndex:MsgSenderIndex])
	c.MessageSender = bz[MsgSenderIndex:BurnContentLength]

	return c, nil
}

func (c *MetadataMessage) Parse(bz []byte) (*MetadataMessage, error) {
	const (
		NonceIndex     = 0
		SenderIndex    = 8
		ChannelIndex   = 40
		PrefixIndex    = 48
		RecipientIndex = 80
		MemoIndex      = 112
	)

	if len(bz) < MemoIndex {
		return nil, errors.New("invalid MetadataMessage length")
	}

	c.Nonce = binary.BigEndian.Uint64(bz[NonceIndex:SenderIndex])
	c.Sender = bz[SenderIndex:ChannelIndex]
	c.Channel = binary.BigEndian.Uint64(bz[ChannelIndex:PrefixIndex])
	c.Prefix = string(bytes.TrimLeft(bz[PrefixIndex:RecipientIndex], string(byte(0))))
	c.Recipient = bz[RecipientIndex:MemoIndex]
	c.Memo = string(bz[MemoIndex:])

	return c, nil
}

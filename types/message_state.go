package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	Created  string = "created"
	Pending  string = "pending"
	Attested string = "attested"
	Complete string = "complete"
	Failed   string = "failed"
	Filtered string = "filtered"

	Mint    string = "mint"
	Forward string = "forward"
)

type Domain uint32

type TxState struct {
	TxHash       string
	Msgs         []*MessageState
	RetryAttempt int
}

type MessageState struct {
	IrisLookupID      string // hex encoded MessageSent bytes
	Status            string // created, pending, attested, complete, failed, filtered
	Attestation       string // hex encoded attestation
	SourceDomain      Domain // uint32 source domain id
	DestDomain        Domain // uint32 destination domain id
	SourceTxHash      string
	DestTxHash        string
	MsgSentBytes      []byte // bytes of the MessageSent message transmitter event
	MsgBody           []byte // bytes of the MessageBody
	DestinationCaller []byte // address authorized to call transaction
	Channel           string // "channel-%d" if a forward, empty if not a forward
	Created           time.Time
	Updated           time.Time
	Nonce             uint64
	MsgVersion        uint32 // CCTP envelope version (0 = v1, 1 = v2)

	// IrisMessage is the message bytes returned by Iris v2 for this tx —
	// differs from MsgSentBytes because Iris populates `nonce` and
	// `finalityThresholdExecuted` before signing. When non-nil it MUST be
	// passed to receiveMessage; using MsgSentBytes instead reverts with
	// "Invalid signature: not attester". Empty for v1.
	IrisMessage []byte
}

// EvmLogToMessageState transforms an evm log into a messageState given an ABI
func EvmLogToMessageState(abi abi.ABI, messageSent abi.Event, log *ethtypes.Log) (messageState *MessageState, err error) {
	event := make(map[string]interface{})
	if err = abi.UnpackIntoMap(event, messageSent.Name, log.Data); err != nil {
		return nil, fmt.Errorf("unable to unpack evm log. error: %w", err)
	}

	rawMessageSentBytes, ok := event["message"].([]byte)
	if !ok {
		return nil, fmt.Errorf("MessageSent.message field missing or not []byte")
	}
	message, err := new(Message).Parse(rawMessageSentBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse CCTP Message header: %w", err)
	}

	hashed := crypto.Keccak256(rawMessageSentBytes)
	hashedHexStr := hex.EncodeToString(hashed)

	messageState = &MessageState{
		IrisLookupID:      hashedHexStr,
		Status:            Created,
		SourceDomain:      Domain(message.SourceDomain),
		DestDomain:        Domain(message.DestinationDomain),
		SourceTxHash:      log.TxHash.Hex(),
		MsgSentBytes:      rawMessageSentBytes,
		MsgBody:           message.MessageBody,
		DestinationCaller: message.DestinationCaller,
		Nonce:             message.Nonce,
		MsgVersion:        message.Version,
		Created:           time.Now(),
		Updated:           time.Now(),
	}

	// BurnMessage.Parse is version-aware (v1 = 132 bytes, v2 >= 228 bytes).
	// We deliberately do NOT fall back to MetadataMessage: that body type
	// belongs to Noble's IBC-forwarding metadata, not burns, and the fallback
	// would mask version-mismatch parse failures by silently accepting any
	// body >= 112 bytes — including v2 burns parsed at v1 offsets.
	if _, err := new(BurnMessage).Parse(message.MessageBody); err != nil {
		return nil, fmt.Errorf("%w (version=%d, body=%d bytes): %w",
			ErrNotBurnMessage, message.Version, len(message.MessageBody), err)
	}
	return messageState, nil
}

// Equal checks if two MessageState instances are equal
func (m *MessageState) Equal(other *MessageState) bool {
	return (m.IrisLookupID == other.IrisLookupID &&
		m.Status == other.Status &&
		m.Attestation == other.Attestation &&
		m.SourceDomain == other.SourceDomain &&
		m.DestDomain == other.DestDomain &&
		m.SourceTxHash == other.SourceTxHash &&
		m.DestTxHash == other.DestTxHash &&
		bytes.Equal(m.MsgSentBytes, other.MsgSentBytes) &&
		bytes.Equal(m.DestinationCaller, other.DestinationCaller) &&
		m.Channel == other.Channel &&
		m.Created == other.Created &&
		m.Updated == other.Updated)
}

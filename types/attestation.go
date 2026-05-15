package types

import "encoding/json"

// AttestationResponse is the v1 attestation response.
// Example: https://iris-api.circle.com/attestations/0x85bbf7e6…
//
// Message is populated only by the v2 dispatch path: in CCTP v2 the Iris
// service back-fills the message's `nonce` (bytes 12-43) and
// `finalityThresholdExecuted` (bytes 144-147) fields before signing, so the
// raw on-chain MessageSent event bytes are NOT what the destination contract
// expects. Broadcasters must submit this message when it is non-empty.
// v1 responses leave it empty; the event bytes are the signed bytes there.
type AttestationResponse struct {
	Attestation string `json:"attestation"`
	Status      string `json:"status"`
	Message     string `json:"-"`
}

// AttestationResponseV2 is the v2 attestation response. The v2 API can return
// multiple messages for a single source transaction.
// Example: https://iris-api.circle.com/v2/messages/0?transactionHash=0x…
type AttestationResponseV2 struct {
	Messages []MessageResponseV2 `json:"messages"`
}

// MessageResponseV2 represents one message inside a v2 attestation response.
// Numeric-or-string fields (cctpVersion / sourceDomain / etc.) are typed as
// json.Number because Iris emits some of them as bare JSON numbers and others
// as strings — strict typing would fail decoding for the wrong shape.
type MessageResponseV2 struct {
	Message                   string      `json:"message"`
	Attestation               string      `json:"attestation"`
	Status                    string      `json:"status"`
	EventNonce                string      `json:"eventNonce"`
	SourceDomain              json.Number `json:"sourceDomain"`
	DestinationDomain         json.Number `json:"destinationDomain"`
	CctpVersion               json.Number `json:"cctpVersion"`
	FinalityThresholdExecuted json.Number `json:"finalityThresholdExecuted"`
	ExpirationBlock           json.Number `json:"expirationBlock"`
}

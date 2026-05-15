package circle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

const defaultHTTPTimeout = 10 * time.Second

// httpGet performs a GET against url and unmarshals JSON into out. Returns a
// "status N: body" error for any non-200, so callers can string-match 404.
func httpGet(url string, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}

func ensure0xPrefix(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s
	}
	return "0x" + s
}

// normalizeBaseURL strips a trailing slash and a trailing /attestations so the
// same configured URL works for both v1 and v2 endpoints.
func normalizeBaseURL(url string) string {
	url = strings.TrimSuffix(url, "/")
	return strings.TrimSuffix(url, "/attestations")
}

// CheckAttestation dispatches to the v1 or v2 Iris endpoint based on the
// configured api-version, returning a unified AttestationResponse so callers
// don't need to know which path was used.
func CheckAttestation(cfg types.CircleSettings, logger log.Logger, irisLookupID, txHash string, sourceDomain, destDomain types.Domain) *types.AttestationResponse {
	version, err := cfg.GetAPIVersion()
	if err != nil {
		logger.Error("invalid api-version", "error", err)
		return nil
	}

	switch version {
	case types.APIVersionV1:
		return checkAttestationV1(cfg.AttestationBaseURL, logger, irisLookupID, txHash, sourceDomain, destDomain)
	case types.APIVersionV2:
		return checkAttestationV2(cfg.AttestationBaseURL, logger, txHash, sourceDomain)
	default:
		logger.Error("unsupported api-version", "version", version)
		return nil
	}
}

func checkAttestationV1(baseURL string, logger log.Logger, irisLookupID, txHash string, sourceDomain, destDomain types.Domain) *types.AttestationResponse {
	base := normalizeBaseURL(baseURL)
	id := ensure0xPrefix(irisLookupID)
	url := fmt.Sprintf("%s/attestations/%s", base, id)
	logger.Debug(fmt.Sprintf("Checking v1 attestation for %s for source tx %s from %d to %d", url, txHash, sourceDomain, destDomain))

	var response types.AttestationResponse
	if err := httpGet(url, &response); err != nil {
		if strings.Contains(err.Error(), "status 404") {
			logger.Debug("v1 attestation not found (may not be ready yet)", "messageHash", id)
		} else {
			logger.Debug("v1 attestation request failed", "error", err.Error())
		}
		return nil
	}
	logger.Info(fmt.Sprintf("Attestation found for %s", url))
	return &response
}

func checkAttestationV2(baseURL string, logger log.Logger, txHash string, sourceDomain types.Domain) *types.AttestationResponse {
	base := normalizeBaseURL(baseURL)
	tx := ensure0xPrefix(txHash)
	url := fmt.Sprintf("%s/v2/messages/%d?transactionHash=%s", base, sourceDomain, tx)
	logger.Debug(fmt.Sprintf("Checking v2 attestation at %s", url))

	var v2 types.AttestationResponseV2
	if err := httpGet(url, &v2); err != nil {
		if strings.Contains(err.Error(), "status 404") {
			logger.Debug("v2 attestation not found (may not be ready yet)", "txHash", tx)
		} else {
			// Anything that isn't a 404 (decode error, 5xx, network glitch)
			// is operationally interesting — log at info so it surfaces
			// without --log-level debug.
			logger.Info("v2 attestation request failed", "txHash", tx, "url", url, "error", err.Error())
		}
		return nil
	}

	if len(v2.Messages) == 0 {
		return nil
	}
	if len(v2.Messages) > 1 {
		logger.Info(fmt.Sprintf("v2 attestation returned %d messages for tx %s, using the first", len(v2.Messages), tx))
	} else {
		logger.Info(fmt.Sprintf("Attestation found for tx %s", tx))
	}

	msg := v2.Messages[0]
	return &types.AttestationResponse{
		Attestation: msg.Attestation,
		// In v2 the message bytes Iris signs over differ from the raw
		// MessageSent event bytes (Iris back-fills nonce and
		// finalityThresholdExecuted). The broadcaster must use these
		// bytes, not the event bytes, otherwise the contract reverts
		// with "Invalid signature: not attester".
		Message: msg.Message,
		Status:  msg.Status,
	}
}

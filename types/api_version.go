package types

import (
	"fmt"
	"strings"
)

type APIVersion string

const (
	APIVersionV1 APIVersion = "v1"
	APIVersionV2 APIVersion = "v2"
)

func ParseAPIVersion(s string) (APIVersion, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "v1", "":
		return APIVersionV1, nil
	case "v2":
		return APIVersionV2, nil
	default:
		return "", fmt.Errorf("invalid api-version %q: must be %s or %s", s, APIVersionV1, APIVersionV2)
	}
}

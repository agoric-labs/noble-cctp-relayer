package types

type ChainConfig interface {
	Chain(name string, watchOnly bool) (Chain, error)
}

type CircleSettings struct {
	APIVersion         string `yaml:"api-version"`
	AttestationBaseURL string `yaml:"attestation-base-url"`
	FetchRetries       int    `yaml:"fetch-retries"`
	FetchRetryInterval int    `yaml:"fetch-retry-interval"`
}

type Config struct {
	AcceptPermissionlessCallers bool `yaml:"accept-permissionless-callers"`
	API                         struct {
		TrustedProxies []string `yaml:"trusted-proxies"`
	} `yaml:"api"`
	Chains        map[string]ChainConfig `yaml:"chains"`
	Circle        CircleSettings         `yaml:"circle"`
	EnabledRoutes map[Domain][]Domain    `yaml:"enabled-routes"`

	ProcessorWorkerCount uint32 `yaml:"processor-worker-count"`
}

type ConfigWrapper struct {
	AcceptPermissionlessCallers *bool `yaml:"accept-permissionless-callers"`
	API                         struct {
		TrustedProxies []string `yaml:"trusted-proxies"`
	} `yaml:"api"`
	Chains               map[string]map[string]any `yaml:"chains"`
	Circle               CircleSettings            `yaml:"circle"`
	EnabledRoutes        map[Domain][]Domain       `yaml:"enabled-routes"`
	ProcessorWorkerCount uint32                    `yaml:"processor-worker-count"`
}

func (c *CircleSettings) GetAPIVersion() (APIVersion, error) {
	return ParseAPIVersion(c.APIVersion)
}

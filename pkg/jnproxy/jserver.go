package jnproxy

// See https://jupyter-client.readthedocs.io/en/stable/messaging.html
type Ports struct {
	Shell     uint
	Stdin     uint
	IOPub     uint
	Control   uint
	Heartbeat uint
}

type NetworkConfig struct {
	IP    string
	Ports Ports
}

type JServerConfig struct {
	srcConfig NetworkConfig
	dstConfig NetworkConfig
}

func JServerConfigNew(srcConfig, dstConfig NetworkConfig) (*JServerConfig, error) {
	// TODO: Validate the inputs.
	return &JServerConfig{
		srcConfig: srcConfig,
		dstConfig: dstConfig,
	}, nil
}

func (s *JServerConfig) src() NetworkConfig {
	return s.srcConfig
}

func (s *JServerConfig) dst() NetworkConfig {
	return s.dstConfig
}

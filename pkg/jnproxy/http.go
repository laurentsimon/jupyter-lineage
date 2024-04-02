package jnproxy

type HttpConfig struct {
	addr []string
}

func HttpConfigNew(addrs []string) (*HttpConfig, error) {
	// TODO: Validate the inputs.
	return &HttpConfig{
		addr: addrs,
	}, nil
}

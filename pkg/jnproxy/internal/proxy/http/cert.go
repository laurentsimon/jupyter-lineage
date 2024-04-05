package http

import (
	"crypto/tls"
	"sync"
)

type certStorage struct {
	certs sync.Map
}

func (tcs *certStorage) Fetch(hostname string, gen func() (*tls.Certificate, error)) (*tls.Certificate, error) {
	var cert tls.Certificate
	icert, ok := tcs.certs.Load(hostname)
	if ok {
		cert = icert.(tls.Certificate)
	} else {
		certp, err := gen()
		if err != nil {
			return nil, err
		}
		cert = *certp
		tcs.certs.Store(hostname, cert)
	}
	return &cert, nil
}

func newCertStorage() *certStorage {
	tcs := &certStorage{}
	tcs.certs = sync.Map{}
	return tcs
}

// TODO: Copy https://github.com/elazarl/goproxy/blob/master/https.go#L467
// and support WithCAKey(), WithCASigner(), WithCertStorage() or EnableCertCaching()
// WithEndDate()
/*
func tlsConfigFromCA(ca *tls.Certificate) func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
	return func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
		var err error
		var cert *tls.Certificate

		hostname := stripPort(host)
		config := &tls.Config{}
		ctx.Logf("signing for %s", stripPort(host))

		genCert := func() (*tls.Certificate, error) {
			return signHost(*ca, []string{hostname})
		}
		if ctx.certStore != nil {
			cert, err = ctx.certStore.Fetch(hostname, genCert)
		} else {
			cert, err = genCert()
		}

		if err != nil {
			ctx.Warnf("Cannot sign host certificate with provided CA: %s", err)
			return nil, err
		}

		config.Certificates = append(config.Certificates, *cert)
		return config, nil
	}
}

func stripPort(s string) string {
	var ix int
	if strings.Contains(s, "[") && strings.Contains(s, "]") {
		//ipv6 : for example : [2606:4700:4700::1111]:443

		//strip '[' and ']'
		s = strings.ReplaceAll(s, "[", "")
		s = strings.ReplaceAll(s, "]", "")

		ix = strings.LastIndexAny(s, ":")
		if ix == -1 {
			return s
		}
	} else {
		//ipv4
		ix = strings.IndexRune(s, ':')
		if ix == -1 {
			return s
		}

	}
	return s[:ix]
}
*/

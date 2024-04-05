package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/elazarl/goproxy"
	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
)

type CA struct {
	Certificate io.Reader
	// TODO: Replace key ybyb signer interface.
	Key io.Reader
}

func (ca *CA) isValid() error {
	if ca.Certificate == nil {
		return fmt.Errorf("%w: empty certificate", errs.ErrorInvalid)
	}
	// TODO: signer
	return nil
}

func WithCA(ca CA) Option {
	return func(p *Proxy) error {
		return p.setCA(ca)
	}
}

func (p *Proxy) setCA(ca CA) error {
	if ca.Certificate == nil {
		return fmt.Errorf("%w: empty certificate", errs.ErrorInvalid)
	}
	// TODO: validate signer
	p.ca = &ca
	return nil
}

func setCA(pca *CA) error {
	cert, err := ioutil.ReadAll(pca.Certificate)
	if err != nil {
		return err
	}
	key, err := ioutil.ReadAll(pca.Key)
	if err != nil {
		return err
	}
	ca, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}
	if ca.Leaf, err = x509.ParseCertificate(ca.Certificate[0]); err != nil {
		return err
	}
	// NOTE: goproxy.GoproxyCa = ca should not be needed.
	goproxy.GoproxyCa = ca
	// NOTE: see https://github.com/elazarl/goproxy/blob/7cc037d33fb57d20c2fa7075adaf0e2d2862da78/https.go#L467,
	// the default cetificate verification is disabled, so we turn it back on.
	tlsConfigFn := func(ca *tls.Certificate) func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
		return func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
			// TODO: Use our own function to support custom signer and other options.
			config, err := goproxy.TLSConfigFromCA(ca)(host, ctx)
			if err != nil {
				return nil, err
			}
			// Disable insecure verification, ie., enable secure verification.
			config.InsecureSkipVerify = false
			return config, nil
		}

	}
	tlsConfig := tlsConfigFn(&ca)
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: tlsConfig}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: tlsConfig}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: tlsConfig}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: tlsConfig}
	return nil
}

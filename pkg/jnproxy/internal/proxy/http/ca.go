package http

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	mrand "math/rand"
	"net"
	"strings"
	"time"

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

func (p *Proxy) setCA(_ca CA) error {
	cert, err := ioutil.ReadAll(_ca.Certificate)
	if err != nil {
		return err
	}
	key, err := ioutil.ReadAll(_ca.Key)
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
	// goproxy.defaultTLSConfig = &tls.Config{}
	// goproxy.tlsClientSkipVerify = &tls.Config{}
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: tlsConfigFromCA(&ca)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: tlsConfigFromCA(&ca)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: tlsConfigFromCA(&ca)}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: tlsConfigFromCA(&ca)}

	return nil
}

// NOTE: Copy from https://github.com/elazarl/goproxy/blob/master/https.go#L467
// TODO: Add support WithCertStorage() or EnableCertCaching()
// See https://go.dev/src/crypto/tls/generate_cert.go for generation.
func tlsConfigFromCA(ca *tls.Certificate) func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
	return func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
		var err error
		var cert *tls.Certificate

		hostname := stripPort(host)
		config := &tls.Config{}

		genCert := func() (*tls.Certificate, error) {
			return generateCert(*ca, []string{hostname})
		}
		// Cert storage is private so we can't use it.
		cert, err = genCert()
		if err != nil {
			return nil, err
		}

		config.Certificates = append(config.Certificates, *cert)
		return config, nil
	}
}

func generateCert(ca tls.Certificate, hosts []string) (*tls.Certificate, error) {
	var x509ca *x509.Certificate

	// Use the provided ca and not the global GoproxyCa for certificate generation.
	x509ca, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, err
	}

	start := time.Unix(time.Now().Unix()-2592000, 0) // 2592000  = 30 day
	end := time.Unix(time.Now().Unix()+31536000, 0)  // 31536000 = 365 day

	serial := big.NewInt(mrand.Int63())
	template := x509.Certificate{
		// TODO(elazar): instead of this ugly hack, just encode the certificate and hash the binary form.
		SerialNumber: serial,
		Issuer:       x509ca.Subject,
		Subject: pkix.Name{
			Organization: []string{"GoProxy untrusted MITM proxy Inc"},
		},
		NotBefore: start,
		NotAfter:  end,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
			template.Subject.CommonName = h
		}
	}

	var certpriv crypto.Signer
	switch ca.PrivateKey.(type) {
	// case *rsa.PrivateKey:
	// 	if certpriv, err = rsa.GenerateKey(&csprng, 2048); err != nil {
	// 		return
	// 	}
	case *ecdsa.PrivateKey:
		// TODO: select key type based on its size.
		//len := key.Curve.Params().BitSize
		certpriv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported key type %T", ca.PrivateKey)
	}

	var derBytes []byte
	derBytes, err = x509.CreateCertificate(rand.Reader, &template, x509ca, certpriv.Public(), ca.PrivateKey)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{
		Certificate: [][]byte{derBytes, ca.Certificate[0]},
		PrivateKey:  certpriv,
	}, nil
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

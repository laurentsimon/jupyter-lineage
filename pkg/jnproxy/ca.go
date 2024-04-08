package jnproxy

import (
	"fmt"
	"io"

	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
)

type CA struct {
	Certificate io.Reader
	// TODO: Replace key ybyb signer interface.
	Key io.Reader
}

func WithCA(ca CA) Option {
	return func(p *JNProxy) error {
		return p.setCA(ca)
	}
}

func (p *JNProxy) setCA(ca CA) error {
	if err := ca.isValid(); err != nil {
		return err
	}
	// TODO: validate signer
	p.ca = &ca
	return nil
}

func (ca *CA) isValid() error {
	if ca.Certificate == nil {
		return fmt.Errorf("%w: empty certificate", errs.ErrorInvalid)
	}
	// TODO: signer
	return nil
}

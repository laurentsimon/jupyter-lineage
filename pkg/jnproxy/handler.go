package jnproxy

import (
	"fmt"

	"github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"
	"github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http/deny"
	hfmodel "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http/huggingface/model"
)

func InstallHandler(handler http.Handler) Option {
	return func(p *JNProxy) error {
		return p.installHandler(handler)
	}
}

func (p *JNProxy) installHandler(handler http.Handler) error {
	p.httpHandlers = append(p.httpHandlers, handler)
	return nil
}

func InstallBuiltinHandlers() Option {
	return func(p *JNProxy) error {
		return p.installBuiltinHandlers()
	}
}

func (p *JNProxy) installBuiltinHandlers() error {
	p.httpHandlers = nil
	// Huggingface model handler.
	if err := p.installHuggingfaceModel(); err != nil {
		return err
	}
	// Add handlers here.
	return nil
}

func RemoveBuiltinHandlers() Option {
	return func(p *JNProxy) error {
		p.httpHandlers = nil
		return nil
	}
}

func InstallHuggingfaceModel() Option {
	return func(p *JNProxy) error {
		return p.installHuggingfaceModel()
	}
}

func (p *JNProxy) installHuggingfaceModel() error {
	hf, err := hfmodel.New()
	if err != nil {
		return fmt.Errorf("huggingface model new: %w", err)
	}
	p.httpHandlers = append(p.httpHandlers, hf)
	return nil
}

func InstallDenyHandler() Option {
	return func(p *JNProxy) error {
		return p.installDenyHandler()
	}
}

func (p *JNProxy) installDenyHandler() error {
	denyHandler, err := deny.New()
	if err != nil {
		return fmt.Errorf("deny new: %w", err)
	}
	p.httpHandlers = append(p.httpHandlers, denyHandler)
	return nil
}

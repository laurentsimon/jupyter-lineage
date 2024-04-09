package proxy

import "github.com/laurentsimon/jupyter-lineage/pkg/slsa"

type Type int

const (
	TypeUserSource Type = iota
	TypeRuntime
)

type Proxy interface {
	Start() error
	Stop() error
	Type() Type
	Dependencies() ([]slsa.ResourceDescriptor, error)
}

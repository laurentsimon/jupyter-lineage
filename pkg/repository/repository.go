package repository

import "github.com/laurentsimon/jupyter-lineage/pkg/slsa"

type Client interface {
	Init() error
	CreateFile(path string, content []byte) error
	Digest() (slsa.DigestSet, error)
	Close() error
}

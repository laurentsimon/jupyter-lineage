package git

import (
	"fmt"
	"os"

	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
)

type Git struct {
	dir string
}

func New() (*Git, error) {
	return &Git{}, nil
}

func (g *Git) Open() error {
	dir, err := os.MkdirTemp("", "jupyter_repo")
	if err != nil {
		return fmt.Errorf("create repo dir: %w", err)
	}
	g.dir = dir
	return nil
}

func (g *Git) ID() string {
	return g.dir
}

func (g *Git) CreateFile(path string, content []byte) error {
	// TODO: verify ID
	return nil
}

func (g *Git) AppendFile(path string, content []byte) error {
	// TODO: verify ID
	return nil
}

func (g *Git) Digest() (repository.Digest, error) {
	return repository.Digest{
			"sha1": "sha1-value"},
		nil
}

func (g *Git) Close() error {
	g.dir = ""
	return nil
}

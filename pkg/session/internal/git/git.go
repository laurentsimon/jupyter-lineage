package git

import (
	"fmt"
	"os"

	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
)

type Client struct {
	dir string
}

func New() (*Client, error) {
	return &Client{}, nil
}

func (c *Client) Init(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("%w: %w", errs.ErrorInvalid, err)
	}
	c.dir = dir
	return nil
}

func (c *Client) Commit(paths []string, message string) (repository.Digest, error) {
	return repository.Digest{
			"sha1": "sha1-value"},
		nil
}

func (c *Client) Close() error {
	c.dir = ""
	return nil
}

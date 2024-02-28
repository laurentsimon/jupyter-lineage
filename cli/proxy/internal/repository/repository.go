package repository

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	repo "github.com/laurentsimon/jupyter-lineage/pkg/repository"
)

type Client struct {
	dir    string
	logger logger.Logger
}

func New(l logger.Logger) (*Client, error) {
	// TODO: init folders.
	return &Client{
		logger: l,
	}, nil
}

func (c *Client) Open() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	dir := filepath.Join(workingDir, "jupyter_repo")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	isEmpty, err := isEmptyDir(dir)
	if err != nil {
		return fmt.Errorf("empty dir: %w", err)
	}
	if !isEmpty {
		return fmt.Errorf("directory %q not clean", dir)
	}
	c.dir = dir
	return nil
}

func isEmptyDir(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func (c *Client) ID() string {
	return c.dir
}

func (c *Client) CreateFile(path string, content []byte) error {
	c.logger.Infof("create file %q with content %q", path, string(content))
	return nil
}

func (c *Client) AppendFile(path string, content []byte) error {
	c.logger.Infof("append file %q with content %q", path, string(content))
	return nil
}

func (c *Client) Digest() (repo.Digest, error) {
	return repo.Digest{
			"sha1": "sha1-value"},
		nil
}

func (c *Client) Close() error {
	c.logger.Infof("close repo %q", c.dir)
	c.dir = ""
	return nil
}

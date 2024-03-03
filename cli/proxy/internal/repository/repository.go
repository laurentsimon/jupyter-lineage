package repository

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	repo "github.com/laurentsimon/jupyter-lineage/pkg/repository"
)

type Client struct {
	dir    string
	dirty  bool
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
	// Init the repo.
	_, stderr, err := c.run("git", "init")
	if err != nil {
		return fmt.Errorf("git init: (stderr=%q): %w", stderr, err)
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
	fn := filepath.Join(c.dir, path)
	os.MkdirAll(filepath.Dir(fn), os.ModePerm)
	c.dirty = true
	return os.WriteFile(fn, content, 0644)
}

func (c *Client) Digest() (repo.Digest, error) {
	if c.dirty {
		// Commit files.
		_, stderr, err := c.run("git", "add", "--all")
		if err != nil {
			return repo.Digest{}, fmt.Errorf("git add -all: (stderr=%q): %w", stderr, err)
		}
		c.dirty = false
	}
	stdout, stderr, err := c.run("git", "rev-parse", "HEAD")
	if err != nil {
		return repo.Digest{}, fmt.Errorf("git rev-parse HEAD: (stderr=%q): %w", stderr, err)
	}

	return repo.Digest{
			"sha1": stdout},
		nil
}

func (c *Client) Close() error {
	c.logger.Infof("close repo %q", c.dir)
	c.dir = ""
	return nil
}

func (c *Client) run(bin string, args ...string) (string, string, error) {
	command := exec.Command(bin, args...)
	command.Dir = c.dir
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("git init: %w", err)
	}
	c.logger.Debugf("repo initialized in %s", c.dir)
	return stdout.String(), "", nil
}

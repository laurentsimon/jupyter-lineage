package repository

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

type Client struct {
	dir    string
	dirty  bool
	logger logger.Logger
}

func New(l logger.Logger, dir string) (*Client, error) {
	return &Client{
		logger: l,
		dir:    dir,
	}, nil
}

func (c *Client) Init() error {
	if _, err := os.Stat(c.dir); os.IsNotExist(err) {
		return fmt.Errorf("dir %q does not exist", c.dir)
	}
	isEmpty, err := isEmptyDir(c.dir)
	if err != nil {
		return fmt.Errorf("empty dir: %w", err)
	}
	if !isEmpty {
		return fmt.Errorf("directory %q not clean", c.dir)
	}
	// Init the repo.
	_, stderr, err := c.run("git", "init")
	if err != nil {
		return fmt.Errorf("git init: (stderr=%q): %w", stderr, err)
	}
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

func (c *Client) CreateFile(path string, content []byte) error {
	c.logger.Infof("create file %q with content %q", path, string(content))
	fn := filepath.Join(c.dir, path)
	os.MkdirAll(filepath.Dir(fn), os.ModePerm)
	c.dirty = true
	return os.WriteFile(fn, content, 0644)
}

func (c *Client) Digest() (slsa.DigestSet, error) {
	if c.dirty {
		// Add files.
		_, stderr, err := c.run("git", "add", "--all")
		if err != nil {
			return slsa.DigestSet{}, fmt.Errorf("git add -all: (stderr=%q): %w", stderr, err)
		}
		// Commit files.
		_, stderr, err = c.run("git", "commit", "-m", "commit_msg")
		if err != nil {
			return slsa.DigestSet{}, fmt.Errorf("git commit -m \"commit_msg\": (stderr=%q): %w", stderr, err)
		}
		c.dirty = false
	}
	stdout, stderr, err := c.run("git", "rev-parse", "HEAD")
	if err != nil {
		return slsa.DigestSet{}, fmt.Errorf("git rev-parse HEAD: (stderr=%q): %w", stderr, err)
	}

	return slsa.DigestSet{
			"sha1": stdout[:len(stdout)-1]}, // Remove last characters which is '\n'
		nil
}

func (c *Client) Close() error {
	c.logger.Infof("close repo %q", c.dir)
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
	c.logger.Debugf("command %v: %s", append([]string{bin}, args...), stdout.String())
	return stdout.String(), "", nil
}

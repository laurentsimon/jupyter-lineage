package session

import (
	"fmt"
	"io"
	"os"

	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
	"github.com/laurentsimon/jupyter-lineage/pkg/repository"
	"github.com/laurentsimon/jupyter-lineage/pkg/session/internal/git"
)

// type Direction uint

// const (
// 	Ingress Direction = 1 << iota
// 	Egress
// )

type state uint

const (
	stateNew state = iota + 1
	stateStarted
	stateFinished
)

// See https://jupyter-client.readthedocs.io/en/stable/messaging.html
type Ports struct {
	Shell     uint
	Stdin     uint
	IOPub     uint
	Control   uint
	Heartbeat uint
}

type NetworkMetadata struct {
	IP    string
	Ports Ports
}

type Session struct {
	srcMetadata NetworkMetadata
	dstMetadata NetworkMetadata
	state       state
	repoClient  repository.Client
	repoDir     string
	proxy       *proxy
	logger      Logger
}

type Option func(*Session) error

type Logger interface {
	Fatalf(string, ...any)
	Errorf(string, ...any)
	Warnf(string, ...any)
	Infof(string, ...any)
	Debugf(string, ...any)
}

// TODO: add a logger interface
func New(srcMeta, dstMeta NetworkMetadata, options ...Option) (*Session, error) {
	// If https://go.googlesource.com/proposal/+/master/design/draft-iofs.md is ever implemented and merged,
	// we'll update the API to take an fs interface.
	addressBinding := []addressBinding{
		{
			name: "shell",
			src:  address(srcMeta.IP, srcMeta.Ports.Shell),
		},
	}
	// TODO: Update this to be in our own repository with better ACLs / permissions.
	session := Session{
		srcMetadata: srcMeta,
		dstMetadata: dstMeta,
		state:       stateNew,
	}

	// Set optional parameters.
	for _, option := range options {
		err := option(&session)
		if err != nil {
			return nil, err
		}
	}
	// Set the default logger
	if err := session.setDefaultLogger(); err != nil {
		return nil, err
	}
	// Create the repo directory.
	if err := session.setDefaultRepoDir(); err != nil {
		return nil, err
	}
	// Set repo client to our default git implementation is not set by the caller.
	if err := session.setDefaultRepoClient(); err != nil {
		return nil, err
	}
	// Set the proxy last, since we need to have the logger setup.
	proxy, err := proxyNew(addressBinding, session.logger)
	if err != nil {
		return nil, err
	}
	session.proxy = proxy
	return &session, nil
}

func address(ip string, port uint) string {
	return fmt.Sprintf("%s:%d", ip, port)
}

func (s *Session) Start() error {
	if s.state != stateNew {
		return fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}
	if err := s.proxy.Start(); err != nil {
		return err
	}
	// Update the session state.
	s.state = stateStarted
	return nil
}

func (s *Session) Finish() error {
	if s.state == stateFinished {
		return fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}
	if err := s.proxy.Finish(); err != nil {
		return err
	}
	// TODO: Use repo to save the information
	// TODO: generate provenance
	// err := os.RemoveAll(s.repoDir)
	return nil
}

func (s *Session) setDefaultLogger() error {
	if s.logger != nil {
		return nil
	}
	s.logger = log{}
	return nil
}

func (s *Session) setDefaultRepoClient() error {
	if s.repoClient != nil {
		return nil
	}
	client, err := git.New()
	if err != nil {
		return fmt.Errorf("create git: %w", err)
	}
	s.repoClient = client

	return nil
}

func (s *Session) setDefaultRepoDir() error {
	if s.repoDir != "" {
		return nil
	}
	repoDir, err := os.MkdirTemp("", "jupyter_repo")
	if err != nil {
		return fmt.Errorf("create repo dir: %w", err)
	}
	s.repoDir = repoDir

	return nil
}

func WithLogger(l Logger) Option {
	return func(s *Session) error {
		return s.setLogger(l)
	}
}

func (s *Session) setLogger(l Logger) error {
	s.logger = l
	return nil
}

func WithRepositoryDir(dir string) Option {
	return func(s *Session) error {
		return s.setRepositoryDir(dir)
	}
}

func (s *Session) setRepositoryDir(dir string) error {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("create user repo dir: %w", err)
	}
	isEmpty, err := isEmptyDir(dir)
	if err != nil {
		return fmt.Errorf("is empty dir: %w", err)
	}
	if !isEmpty {
		return fmt.Errorf("directory %q not clean", dir)
	}
	s.repoDir = dir
	return nil
}

func WithRepositoryClient(repoClient repository.Client) Option {
	return func(s *Session) error {
		return s.setRepositoryClient(repoClient)
	}
}

func (s *Session) setRepositoryClient(repoClient repository.Client) error {
	s.repoClient = repoClient
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

// TODO: HMAC keys

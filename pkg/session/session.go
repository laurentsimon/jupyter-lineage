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
	stateRecording
	stateEnded
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
	listeningMetadata   NetworkMetadata
	destinationMetadata NetworkMetadata
	state               state
	repoClient          repository.Client
	repoDir             string
}

type Option func(*Session) error

func New(listeningMeta, destinationMeta NetworkMetadata, options ...Option) (*Session, error) {
	// If https://go.googlesource.com/proposal/+/master/design/draft-iofs.md is ever implemented and merged,
	// we'll update the API to take an fs interface.

	// Create a directory to store the repo.
	// TODO: Update this to be in our own repository with better ACLs / permissions.
	session := Session{
		listeningMetadata:   listeningMeta,
		destinationMetadata: destinationMeta,
		state:               stateNew,
	}

	// Set optional parameters.
	for _, option := range options {
		err := option(&session)
		if err != nil {
			return nil, err
		}
	}

	// Create the repo directory.
	if err := session.setDefaultRepoDir(); err != nil {
		return nil, err
	}
	// Set repo client to our default git implementation is not set by the caller.
	if err := session.setDefaultRepoClient(); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Session) Start() error {
	if s.state != stateNew {
		return fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}
	// create the repoDir
	// TODO: Start listening
	// Update the session state.
	s.state = stateRecording
	return nil
}

func (s *Session) End() error {
	if s.state == stateEnded {
		return fmt.Errorf("%w: state %q", errs.ErrorInvalid, s.state)
	}
	// TODO: Use repo to save the information
	// TODO: generate provenance
	// err := os.RemoveAll(s.repoDir)
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

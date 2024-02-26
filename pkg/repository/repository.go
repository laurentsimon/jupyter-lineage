package repository

type Digest map[string]string

type Client interface {
	Init(dir string) error
	Commit(paths []string, message string) (Digest, error)
	Close() error
}

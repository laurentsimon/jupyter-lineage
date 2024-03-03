package repository

type Digest map[string]string

type Client interface {
	Open() error
	ID() string
	CreateFile(path string, content []byte) error
	Digest() (Digest, error)
	Close() error
}

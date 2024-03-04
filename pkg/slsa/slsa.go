package slsa

type DigestSet map[string]string

type Subject struct {
	Name      string    `json:"name,omitempty"`
	DigestSet DigestSet `json:"digest,omitempty"`
}

type Builder struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

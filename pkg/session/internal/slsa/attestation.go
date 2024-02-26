package slsa

// See https://github.com/in-toto/in-toto-golang/tree/master/in_toto/slsa_provenance/v1

type DigestSet map[string]string

type Dependency struct {
	URI       string    `json:"uri,omitempty"`
	DigestSet DigestSet `json:"digest,omitempty"`
}

type Subject struct {
	Name    string    `json:"name,omitempty"`
	Digests DigestSet `json:"digest,omitempty"`
}

type Builder struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type Header struct {
	Type          string    `json:"_type"`
	PredicateType string    `json:"predicateType"`
	Subjects      []Subject `json:"subject"`
}

type BuildDefinition struct {
	BuildType            string       `json:"buildType"`
	InternalParameters   interface{}  `json:"internalParameters,omitempty"`
	ResolvedDependencies []Dependency `json:"resolvedDependencies,omitempty"`
}

type RunDetails struct {
	Builder       Builder       `json:"builder"`
	BuildMetadata BuildMetadata `json:"metadata,omitempty"`
}

type BuildMetadata struct {
	InvocationID string `json:"invocationID,omitempty"`
	StartedOn    string `json:"startedOn,omitempty"`
	FinishedOn   string `json:"finishedOn,omitempty"`
}

type predicate struct {
	BuildDefinition BuildDefinition `json:"buildDefinition"`
	RunDetails      RunDetails      `json:"runDetails"`
}

type attestation struct {
	Header
	Predicate predicate `json:"predicate"`
}

const (
	statementType = "https://in-toto.io/Statement/v1"
	predicateType = "https://slsa.dev/provenance/v1"
	buildType     = "https://slsa-framework/jupyter-lineage/v1"
)

package slsa

import "github.com/laurentsimon/jupyter-lineage/pkg/slsa"

// See https://github.com/in-toto/in-toto-golang/tree/master/in_toto/slsa_provenance/v1

type Dependency struct {
	URI       string         `json:"uri,omitempty"`
	DigestSet slsa.DigestSet `json:"digest,omitempty"`
}

type Header struct {
	Type          string         `json:"_type"`
	PredicateType string         `json:"predicateType"`
	Subjects      []slsa.Subject `json:"subject"`
}

type BuildDefinition struct {
	BuildType            string       `json:"buildType"`
	InternalParameters   interface{}  `json:"internalParameters,omitempty"`
	ResolvedDependencies []Dependency `json:"resolvedDependencies,omitempty"`
}

type RunDetails struct {
	Builder       slsa.Builder  `json:"builder"`
	BuildMetadata BuildMetadata `json:"metadata,omitempty"`
}

type BuildMetadata struct {
	InvocationID string `json:"invocationID,omitempty"`
	StartedOn    string `json:"startedOn,omitempty"`
	FinishedOn   string `json:"finishedOn,omitempty"`
}

type Predicate struct {
	BuildDefinition BuildDefinition `json:"buildDefinition"`
	RunDetails      RunDetails      `json:"runDetails"`
}

type attestation struct {
	Header
	Predicate Predicate `json:"predicate"`
}

const (
	statementType = "https://in-toto.io/Statement/v1"
	predicateType = "https://slsa.dev/provenance/v1"
	buildType     = "https://slsa-framework/jupyter-lineage/back-position/0.1"
)

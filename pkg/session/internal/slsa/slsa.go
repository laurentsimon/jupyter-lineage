package slsa

import (
	"encoding/json"
	"fmt"
	"time"
)

type Provenance struct {
	attestation attestation
}

type Option func(*Provenance) error

func New(builder Builder, subjects []Subject, repo Dependency) (*Provenance, error) {
	return &Provenance{
		attestation: attestation{
			Header: Header{
				Type:          statementType,
				PredicateType: predicateType,
				Subjects:      append([]Subject{}, subjects...), // NOTE: Make a copy.
			},
			Predicate: Predicate{
				BuildDefinition: BuildDefinition{
					BuildType:            buildType,
					ResolvedDependencies: append([]Dependency{}, repo), // NOTE: Make a copy.
				},
				RunDetails: RunDetails{
					Builder: builder, // TODO: Should make a copy?
				},
			},
		},
	}, nil
}

func (p *Provenance) AddDependencies(deps []Dependency) Option {
	return func(p *Provenance) error {
		return p.addDependencies(deps)
	}
}

func (p *Provenance) addDependencies(deps []Dependency) error {
	p.attestation.Predicate.BuildDefinition.ResolvedDependencies = append(p.attestation.Predicate.BuildDefinition.ResolvedDependencies, deps...)
	return nil
}

func (p *Provenance) WithStartTime(t time.Time) Option {
	return func(p *Provenance) error {
		return p.withStartTime(t)
	}
}

func (p *Provenance) withStartTime(t time.Time) error {
	p.attestation.Predicate.RunDetails.BuildMetadata.StartedOn = t.UTC().Format(time.RFC3339)
	return nil
}

func (p *Provenance) WithfinishTime(t time.Time) Option {
	return func(p *Provenance) error {
		return p.withFinishTime(t)
	}
}

func (p *Provenance) withFinishTime(t time.Time) error {
	p.attestation.Predicate.RunDetails.BuildMetadata.FinishedOn = t.UTC().Format(time.RFC3339)
	return nil
}

func (p *Provenance) ToBytes() ([]byte, error) {
	content, err := json.Marshal(p.attestation)
	if err != nil {
		return nil, fmt.Errorf("marshal: %v", err)
	}
	return content, nil
}

func Format(content []byte) ([]byte, error) {
	type Content struct {
		Time  string `json:"time"`
		Value []byte `json:"content"`
	}
	c := Content{
		Time:  time.Now().UTC().Format(time.RFC3339),
		Value: content,
	}
	// TODO: decide the format we want. Is jupyter text-based only?
	// NOTE: https://golang.org/pkg/encoding/json/#Marshal
	// Array and slice values encode as JSON arrays, except that []byte encodes as a base64-encoded string, and a nil slice encodes as the null JSON object.
	// use base64.StdEncoding.DecodeString() for decoding.
	ret, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return ret, nil
}

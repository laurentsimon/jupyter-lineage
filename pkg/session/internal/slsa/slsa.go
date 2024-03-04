package slsa

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

type Provenance struct {
	attestation attestation
}

type Option func(*Provenance) error

func New(builder slsa.Builder, subjects []slsa.Subject, repo Dependency, opts ...Option) (*Provenance, error) {
	p := Provenance{
		attestation: attestation{
			Header: Header{
				Type:          statementType,
				PredicateType: predicateType,
				Subjects:      append([]slsa.Subject{}, subjects...),
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
	}

	// Set optional parameters.
	for _, option := range opts {
		err := option(&p)
		if err != nil {
			return nil, err
		}
	}
	return &p, nil
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

func WithStartTime(t time.Time) Option {
	return func(p *Provenance) error {
		return p.withStartTime(t)
	}
}

func (p *Provenance) withStartTime(t time.Time) error {
	p.attestation.Predicate.RunDetails.BuildMetadata.StartedOn = t.UTC().Format(time.RFC3339)
	return nil
}

func WithFinishTime(t time.Time) Option {
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

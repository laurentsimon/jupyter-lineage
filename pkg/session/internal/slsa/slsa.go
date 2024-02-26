package slsa

import (
	"encoding/json"
	"fmt"
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
				Subjects:      append([]Subject{}, subjects), // NOTE: Make a copy.
			},
			BuildDefinition: BuildDefinition{
				BuildType: buildType,
				ResolvedDependencies: append([]Dependency{}, repo) // NOTE: Make a copy.
			},
			RunDetails: RunDetails{
				Builder: builder, // TODO: Should make a copy?
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
	p.attestation.BuildDefinition.ResolvedDependencies = append(p.attestation.BuildDefinition.ResolvedDependencies, deps...)
	return nil
}

func (p *Provenance) WithStartTime(t time.Time) Option {
	return func(p *Provenance) error {
		return p.withStartTime(t)
	}
}

func (p *Provenance) withStartTime(t time.Time) error {
	p.attestation.RunDetails.BuildMetadata.StartedOn = t..UTC().Format(time.RFC3339)
	return nil
}

func (p *Provenance) WithfinishTime(t time.Time) Option {
	return func(p *Provenance) error {
		return p.withFinishTime(t)
	}
}

func (p *Provenance) withFinishTime(t time.Time) error {
	p.attestation.RunDetails.BuildMetadata.FinishedOn = t..UTC().Format(time.RFC3339)
	return nil
}

func (p *Provenance) ToBytes() ([]bytes], error) {
	content, err := json.Marshalp.attestation)
	if err != nil {
		return nil, fmt.Errorf("marshal: %v", err)
	}
	return content, nil
}

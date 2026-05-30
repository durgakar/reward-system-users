package segment

import (
	"context"
	"fmt"

	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/internal/rules"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// StaticProvider evaluates YAML-defined segments. Register additional
// SegmentProvider implementations for CRM-specific logic.
type StaticProvider struct {
	definitions []domain.SegmentDefinition
}

func NewStaticProvider(definitions []domain.SegmentDefinition) *StaticProvider {
	return &StaticProvider{definitions: definitions}
}

func (p *StaticProvider) Name() string { return "static" }

func (p *StaticProvider) Evaluate(_ context.Context, _ plugin.Client, profile plugin.ClientProfile) ([]string, error) {
	var matched []string
	for _, seg := range p.definitions {
		ok, err := matchAll(seg.Match.All, profile)
		if err != nil {
			return nil, fmt.Errorf("segment %q: %w", seg.ID, err)
		}
		if ok {
			matched = append(matched, seg.ID)
		}
	}
	return matched, nil
}

func matchAll(conditions []domain.Condition, profile plugin.ClientProfile) (bool, error) {
	for _, c := range conditions {
		ok, err := rules.MatchCondition(c, profile)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

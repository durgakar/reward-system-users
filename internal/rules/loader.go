package rules

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/durgakar/reward-system-users/internal/domain"
)

func LoadSegments(path string) ([]domain.SegmentDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read segments: %w", err)
	}
	var file domain.SegmentsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse segments: %w", err)
	}
	return file.Segments, nil
}

func LoadRules(path string) ([]domain.RuleDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules: %w", err)
	}
	var file domain.RulesFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	return file.Rules, nil
}

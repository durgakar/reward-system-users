package domain

// Operator compares a profile field to a configured value.
type Operator string

const (
	OpEQ  Operator = "eq"
	OpNEQ Operator = "neq"
	OpGT  Operator = "gt"
	OpGTE Operator = "gte"
	OpLT  Operator = "lt"
	OpLTE Operator = "lte"
	OpIn  Operator = "in"
)

// Condition is a single predicate on a client profile field.
type Condition struct {
	Field    string   `json:"field" yaml:"field"`
	Operator Operator `json:"operator" yaml:"operator"`
	Value    any      `json:"value" yaml:"value"`
}

// MatchAll requires every nested condition to pass.
type MatchAll struct {
	All []Condition `json:"all" yaml:"all"`
}

// SegmentDefinition describes how clients enter a segment.
type SegmentDefinition struct {
	ID          string   `json:"id" yaml:"id"`
	Description string   `json:"description" yaml:"description"`
	Match       MatchAll `json:"match" yaml:"match"`
}

// ActionType is what a rule triggers when it matches.
type ActionType string

const (
	ActionAwardPoints ActionType = "award_points"
	ActionSendEmail   ActionType = "send_email"
)

// Action is one side-effect of a matched rule.
type Action struct {
	Type     ActionType `json:"type" yaml:"type"`
	Points   int        `json:"points,omitempty" yaml:"points,omitempty"`
	Template string     `json:"template,omitempty" yaml:"template,omitempty"`
	Subject  string     `json:"subject,omitempty" yaml:"subject,omitempty"`
}

// RuleDefinition is a declarative campaign rule loaded from YAML.
type RuleDefinition struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
	Segment     string    `json:"segment,omitempty" yaml:"segment,omitempty"`
	Condition   Condition `json:"condition" yaml:"condition"`
	Actions     []Action  `json:"actions" yaml:"actions"`
	Enabled     bool      `json:"enabled" yaml:"enabled"`
}

// SegmentsFile is the on-disk shape of config/segments.yaml.
type SegmentsFile struct {
	Segments []SegmentDefinition `yaml:"segments"`
}

// RulesFile is the on-disk shape of config/rules.yaml.
type RulesFile struct {
	Rules []RuleDefinition `yaml:"rules"`
}

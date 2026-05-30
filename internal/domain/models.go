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
	Field    string   `yaml:"field"`
	Operator Operator `yaml:"operator"`
	Value    any      `yaml:"value"`
}

// MatchAll requires every nested condition to pass.
type MatchAll struct {
	All []Condition `yaml:"all"`
}

// SegmentDefinition describes how clients enter a segment.
type SegmentDefinition struct {
	ID          string     `yaml:"id"`
	Description string     `yaml:"description"`
	Match       MatchAll   `yaml:"match"`
}

// ActionType is what a rule triggers when it matches.
type ActionType string

const (
	ActionAwardPoints ActionType = "award_points"
	ActionSendEmail   ActionType = "send_email"
)

// Action is one side-effect of a matched rule.
type Action struct {
	Type     ActionType `yaml:"type"`
	Points   int        `yaml:"points,omitempty"`
	Template string     `yaml:"template,omitempty"`
	Subject  string     `yaml:"subject,omitempty"`
}

// RuleDefinition is a declarative campaign rule loaded from YAML.
type RuleDefinition struct {
	ID          string     `yaml:"id"`
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Segment     string     `yaml:"segment,omitempty"` // empty = all clients
	Condition   Condition  `yaml:"condition"`
	Actions     []Action   `yaml:"actions"`
	Enabled     bool       `yaml:"enabled"`
}

// SegmentsFile is the on-disk shape of config/segments.yaml.
type SegmentsFile struct {
	Segments []SegmentDefinition `yaml:"segments"`
}

// RulesFile is the on-disk shape of config/rules.yaml.
type RulesFile struct {
	Rules []RuleDefinition `yaml:"rules"`
}

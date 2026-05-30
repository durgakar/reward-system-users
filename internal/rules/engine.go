package rules

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// Engine evaluates declarative rules against a client profile.
type Engine struct {
	rules []domain.RuleDefinition
}

func NewEngine(rules []domain.RuleDefinition) *Engine {
	enabled := make([]domain.RuleDefinition, 0, len(rules))
	for _, r := range rules {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	return &Engine{rules: enabled}
}

func (e *Engine) Evaluate(client plugin.Client, profile plugin.ClientProfile, segments []string) ([]plugin.RuleOutcome, error) {
	segmentSet := make(map[string]struct{}, len(segments))
	for _, s := range segments {
		segmentSet[s] = struct{}{}
	}

	var outcomes []plugin.RuleOutcome
	for _, rule := range e.rules {
		if rule.Segment != "" {
			if _, ok := segmentSet[rule.Segment]; !ok {
				continue
			}
		}
		ok, err := MatchCondition(rule.Condition, profile)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", rule.ID, err)
		}
		if !ok {
			continue
		}
		outcome := plugin.RuleOutcome{
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Metadata: map[string]string{"description": rule.Description},
		}
		for _, action := range rule.Actions {
			switch action.Type {
			case domain.ActionAwardPoints:
				outcome.AwardPoints += action.Points
			case domain.ActionSendEmail:
				outcome.EmailTemplate = action.Template
				outcome.EmailSubject = action.Subject
			default:
				return nil, fmt.Errorf("rule %q: unknown action type %q", rule.ID, action.Type)
			}
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}

// MatchCondition evaluates a single condition against a profile.
func MatchCondition(c domain.Condition, profile plugin.ClientProfile) (bool, error) {
	actual, err := fieldValue(c.Field, profile)
	if err != nil {
		return false, err
	}
	return compare(actual, c.Operator, c.Value)
}

func fieldValue(field string, profile plugin.ClientProfile) (any, error) {
	switch field {
	case "lifetime_spend_usd":
		return profile.LifetimeSpendUSD, nil
	case "last_order_total_usd":
		return profile.LastOrderTotalUSD, nil
	case "orders_last_90_days":
		return profile.OrdersLast90Days, nil
	case "average_order_usd":
		return profile.AverageOrderUSD, nil
	case "days_since_last_order":
		return profile.DaysSinceLastOrder, nil
	case "preferred_category":
		return profile.PreferredCategory, nil
	case "last_order_at":
		return profile.LastOrderAt, nil
	default:
		if profile.Custom != nil {
			if v, ok := profile.Custom[field]; ok {
				return v, nil
			}
		}
		return nil, fmt.Errorf("unknown profile field %q", field)
	}
}

func compare(actual any, op domain.Operator, expected any) (bool, error) {
	switch op {
	case domain.OpIn:
		list, ok := expected.([]any)
		if !ok {
			return false, fmt.Errorf("in operator expects array value")
		}
		actualStr := fmt.Sprint(actual)
		for _, item := range list {
			if fmt.Sprint(item) == actualStr {
				return true, nil
			}
		}
		return false, nil
	}

	if tActual, ok := actual.(time.Time); ok {
		tExpected, err := parseTime(expected)
		if err != nil {
			return false, err
		}
		switch op {
		case domain.OpEQ:
			return tActual.Equal(tExpected), nil
		case domain.OpGT:
			return tActual.After(tExpected), nil
		case domain.OpGTE:
			return !tActual.Before(tExpected), nil
		case domain.OpLT:
			return tActual.Before(tExpected), nil
		case domain.OpLTE:
			return !tActual.After(tExpected), nil
		default:
			return false, fmt.Errorf("unsupported operator %q for time", op)
		}
	}

	a, err := toFloat64(actual)
	if err != nil {
		if op == domain.OpEQ || op == domain.OpNEQ {
			as := fmt.Sprint(actual)
			es := fmt.Sprint(expected)
			if op == domain.OpEQ {
				return as == es, nil
			}
			return as != es, nil
		}
		return false, err
	}
	e, err := toFloat64(expected)
	if err != nil {
		return false, err
	}
	switch op {
	case domain.OpEQ:
		return a == e, nil
	case domain.OpNEQ:
		return a != e, nil
	case domain.OpGT:
		return a > e, nil
	case domain.OpGTE:
		return a >= e, nil
	case domain.OpLT:
		return a < e, nil
	case domain.OpLTE:
		return a <= e, nil
	default:
		return false, fmt.Errorf("unknown operator %q", op)
	}
}

func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return float64(rv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return float64(rv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return rv.Float(), nil
		default:
			return 0, fmt.Errorf("cannot compare type %T", v)
		}
	}
}

func parseTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case string:
		layouts := []string{time.RFC3339, "2006-01-02", time.RFC3339Nano}
		for _, layout := range layouts {
			if parsed, err := time.Parse(layout, strings.TrimSpace(t)); err == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("invalid time %q", t)
	default:
		return time.Time{}, fmt.Errorf("invalid time type %T", v)
	}
}

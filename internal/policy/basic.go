package policy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Operator string

type Policy interface {
	// ToRego converts the policy to a Rego expression.
	ToRego() string
}

const (
	OperatorAnd Operator = "AND"
	OperatorOr  Operator = "OR"
)

type PolicyDetail struct {
	Value    []string `yaml:"value"`
	Operator Operator `yaml:"operator"`
}

// UserPolicy represents a policy that checks if a user is in a list of allowed users (OR)
// or if the user is equal to a specific list of values (AND).
type UserPolicy struct {
	PolicyDetail `yaml:",inline"`
}

// ToRego converts the UserPolicy to a Rego expression.
// It generates a Rego rule that checks if the user is in the allowed list or matches the specified values.
// If the operator is AND, it checks for equality with each value.
// If the operator is OR, it checks if the user is in the list of values.
// The generated Rego rule is returned as a string.
// (user read from JWT)
func (p *UserPolicy) ToRego() string {
	if len(p.Value) == 0 {
		return ""
	}
	var result string
	if p.Operator == OperatorAnd {
		for _, v := range p.Value {
			vJSON, err := json.Marshal(v)
			if err != nil {
				panic(err)
			}
			result += "user == " + string(vJSON) + "\n"
		}
	} else {
		values, err := json.Marshal(p.Value)
		if err != nil {
			panic(err)
		}
		result = "user in " + string(values) + "\n"
	}
	return result
}

// RolePolicy represents a policy that checks if a user has a specific role (AND)
// or if the user has any of the roles in a list (OR).
type RolePolicy struct {
	PolicyDetail `yaml:",inline"`
}

// ToRego converts the RolePolicy to a Rego expression.
// It generates a Rego rule that checks if the user has the specified role or matches any of the roles in the list.
// If the operator is AND, it checks that all roles required are present.
// If the operator is OR, it checks if the user has any of the roles in the list.
// (roles read from JWT)
func (p *RolePolicy) ToRego() string {
	if len(p.Value) == 0 {
		return ""
	}
	var result string

	roleJson, err := json.Marshal(p.Value)
	if err != nil {
		panic(err)
	}
	roleJsonSet := string(roleJson)
	roleJsonSet = strings.Replace(roleJsonSet, "[", "{", 1)
	roleJsonSet = strings.Replace(roleJsonSet, "]", "}", 1)

	if p.Operator == OperatorAnd {
		result = "count(" + roleJsonSet + "-roles) == 0\n"
	} else {
		result = "count(" + roleJsonSet + "&roles) != 0\n"
	}
	return result
}

// LocationPolicy represents a policy that checks if the user location is in a list of allowed locations (OR)
// or if the location is equal to a specific list of values (AND).
// (location read from JWT)
type LocationPolicy struct {
	PolicyDetail `yaml:",inline"`
}

func (p *LocationPolicy) ToRego() string {
	if len(p.Value) == 0 {
		return ""
	}

	const locationClaim = "token.payload.location"

	if p.Operator == OperatorAnd {
		var result string
		for _, v := range p.Value {
			vJSON, err := json.Marshal(v)
			if err != nil {
				panic(err)
			}
			result += locationClaim + " == " + string(vJSON) + "\n"
		}
		return result
	}

	values, err := json.Marshal(p.Value)
	if err != nil {
		panic(err)
	}
	return locationClaim + " in " + string(values) + "\n"
}

// RateLimitEntry is the shared shape of one "value" array entry for both CallPolicy and CallTimePolicy:
type RateLimitEntry struct {
	Min  *int   `yaml:"min"`
	Max  *int   `yaml:"max"`
	Unit string `yaml:"unit"`
}

func (r *RateLimitEntry) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Min           yaml.Node `yaml:"min"`
		Max           yaml.Node `yaml:"max"`
		Unit          string    `yaml:"unit"`
		UnitOfMeasure string    `yaml:"unit_of_measure"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if raw.Unit != "" {
		r.Unit = raw.Unit
	} else {
		r.Unit = raw.UnitOfMeasure
	}
	parseIntNode := func(node *yaml.Node) (*int, error) {
		if node == nil || node.Kind == 0 || node.Tag == "!!null" {
			return nil, nil
		}
		var val int
		if err := node.Decode(&val); err == nil {
			return &val, nil
		}
		var str string
		if err := node.Decode(&str); err != nil {
			return nil, err
		}
		if str == "" {
			return nil, nil
		}
		i, err := strconv.Atoi(str)
		if err != nil {
			return nil, err
		}
		return &i, nil
	}
	min, err := parseIntNode(&raw.Min)
	if err != nil {
		return err
	}
	r.Min = min
	max, err := parseIntNode(&raw.Max)
	if err != nil {
		return err
	}
	r.Max = max
	return nil
}

// isValidTimeUnit returns true if the unit of measure is a valid one
func isValidTimeUnit(unit string) bool {
	switch strings.ToLower(unit) {
	case "h", "hr", "hour", "hours", "m", "min", "minute", "minutes":
		return true
	}
	return false
}

// scaleToMinutes converts hours to minutes
func scaleToMinutes(unit string) int {
	switch strings.ToLower(unit) {
	case "m", "min", "minute", "minutes":
		return 1
	default: // "h", "hr", "hour", "hours"
		return 60
	}
}

// minuteOfDayExpr is the Rego expression for "minutes since midnight" of the current evaluation time
const minuteOfDayExpr = "((time.clock(time.now_ns())[0] * 60) + time.clock(time.now_ns())[1])"

// CallTimePolicy matches the "call_time" clause
type CallTimePolicy struct {
	Operator Operator         `yaml:"operator"`
	Value    []RateLimitEntry `yaml:"value"`
}

type timeWindowMinutes struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func (t *CallTimePolicy) ToRego() string {
	if len(t.Value) == 0 {
		slog.Warn("Empty 'call_time' value, the policy clause is dropped", "operator", t.Operator)
		return ""
	}

	var windowEntries []RateLimitEntry
	invalidUnit := false
	for _, e := range t.Value {
		if isValidTimeUnit(e.Unit) {
			windowEntries = append(windowEntries, e)
		} else {
			invalidUnit = true
		}
	}
	slog.Debug("Translating 'call_time' policy",
		"operator", t.Operator, "entries", len(t.Value))

	var result string
	if invalidUnit {
		slog.Warn("'call_time' doesn't support the selected unit of time, the rule will be skipped (so access granted)",
			"operator", t.Operator, "values", formatRateLimitEntries(t.Value))
	}
	if len(windowEntries) == 0 {
		return result
	}

	if t.Operator == OperatorAnd {
		// AND: every window must hold simultaneously, one line per entry
		for _, e := range windowEntries {
			scale := scaleToMinutes(e.Unit)
			if e.Min != nil {
				result += fmt.Sprintf("%s >= %d\n", minuteOfDayExpr, *e.Min*scale)
			}
			if e.Max != nil {
				result += fmt.Sprintf("%s <= %d\n", minuteOfDayExpr, *e.Max*scale)
			}
		}
		return result
	}

	// OR: the request time must fall inside at least one window
	windows := make([]timeWindowMinutes, 0, len(windowEntries))
	for _, e := range windowEntries {
		scale := scaleToMinutes(e.Unit)
		w := timeWindowMinutes{Max: 24 * 60} // open-ended upper bound by default
		if e.Min != nil {
			w.Min = *e.Min * scale
		}
		if e.Max != nil {
			w.Max = *e.Max * scale
		}
		windows = append(windows, w)
	}
	windowsJSON, err := json.Marshal(windows)
	if err != nil {
		panic(err)
	}
	result += fmt.Sprintf("count([w | w := %s[_]; %s >= w.min; %s <= w.max]) > 0\n",
		string(windowsJSON), minuteOfDayExpr, minuteOfDayExpr)
	return result
}

// TODO: create the setup for this to work ------------------------------------------------------------

// CallPolicy matches the "calls" clause
type CallPolicy struct {
	Operator Operator         `yaml:"operator"`
	Value    []RateLimitEntry `yaml:"value"`
}

// formatRateLimitEntries renders rate-limit entries for logs
func formatRateLimitEntries(entries []RateLimitEntry) string {
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		minValue, maxValue := "none", "none"
		if e.Min != nil {
			minValue = strconv.Itoa(*e.Min)
		}
		if e.Max != nil {
			maxValue = strconv.Itoa(*e.Max)
		}
		parts = append(parts, fmt.Sprintf("{min:%s max:%s unit:%s}", minValue, maxValue, e.Unit))
	}
	return strings.Join(parts, " ")
}

func (c *CallPolicy) ToRego() string {
	if len(c.Value) == 0 {
		slog.Warn("Empty 'calls' value, the policy clause is dropped", "operator", c.Operator)
		return ""
	}
	return "# UNENFORCED: 'calls' rate-limit policy requires external setup; not evaluable in Rego alone\n"
}

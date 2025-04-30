package policy

import (
	"encoding/json"
	"strings"
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

// UserPolicy represents a policy that checks if a user is in a list of allowed users (OR) or if the user is equal to a specific list of values (AND).
type UserPolicy struct {
	PolicyDetail `yaml:",inline"`
}

// ToRego converts the UserPolicy to a Rego expression.
// It generates a Rego rule that checks if the user is in the allowed list or matches the specified values.
// If the operator is AND, it checks for equality with each value.
// If the operator is OR, it checks if the user is in the list of values.
// The generated Rego rule is returned as a string.
func (p *UserPolicy) ToRego() string {
	var result string
	if p.Operator == OperatorAnd {
		for _, v := range p.Value {
			result += "user == " + v + "\n"
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

// RolePolicy represents a policy that checks if a user has a specific role (AND) or if the user has any of the roles in a list (OR).
type RolePolicy struct {
	PolicyDetail `yaml:",inline"`
}

// ToRego converts the RolePolicy to a Rego expression.
// It generates a Rego rule that checks if the user has the specified role or matches any of the roles in the list.
// If the operator is AND, it checks that all roles required are present.
// If the operator is OR, it checks if the user has any of the roles in the list.
func (p *RolePolicy) ToRego() string {
	var result string
	if len(p.Value) == 0 {
		return result
	}

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

// StorageLocationPolicy represents a policy that checks if a storage location is in a list of allowed locations (OR) or if the location is equal to a specific list of values (AND).
type StorageLocationPolicy struct {
	PolicyDetail `yaml:",inline"`
}

func (p *StorageLocationPolicy) ToRego() string {
	return ""
}

type CallFrequency string

const (
	CallFrequencyDaily   CallFrequency = "call_per_day"
	CallFrequencyWeekly  CallFrequency = "call_per_week"
	CallFrequencyMonthly CallFrequency = "call_per_month"
)

// CallPolicy represents a policy that checks the maximum number of calls allowed in a given time period.
type CallPolicy struct {
	Value []struct {
		Max           string
		UnitOfMeasure CallFrequency `yaml:"unit_of_measure"`
	}
}

func (call *CallPolicy) ToRego() string {
	// TODO(anyone): Understand how to implement this in Rego
	return ""
}

type StorageDuration string

const (
	StorageDurationDay   StorageDuration = "days"
	StorageDurationWeek  StorageDuration = "weeks"
	StorageDurationMonth StorageDuration = "months"
)

// TimelinessPolicy represents a policy that checks the maximum time allowed for data persistence.
type TimelinessPolicy struct {
	Value []struct {
		Max           string
		UnitOfMeasure StorageDuration `yaml:"unit_of_measure"`
	}
}

func (timeliness *TimelinessPolicy) ToRego() string {
	// TODO(anyone): Understand how to implement this in Rego
	return ""
}

package policy

import "encoding/json"

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

func (p *RolePolicy) ToRego() string {
	var result string
	if len(p.Value) == 0 {
		return result
	}
	if p.Operator == OperatorAnd {
		for _, v := range p.Value {
			result += "\"" + v + "\" in roles\n"
		}
	} else {
		values, err := json.Marshal(p.Value)
		if err != nil {
			panic(err)
		}
		result = "some role in roles\nrole in " + string(values) + "\n"
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

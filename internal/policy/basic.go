package policy

type Operator string

type Interval struct {
	Min           int    `yaml:"min" json:"min,omitempty"`
	Max           int    `yaml:"max" json:"max,omitempty"`
	UnitOfMeasure string `yaml:"unit_of_measure" json:"unit_of_measure,omitempty"`
}

type IntervalValue struct {
	Value []Interval `yaml:"value" json:"value"`
}

type EnumeratedValue struct {
	Value []string `yaml:"value" json:"value"`
}

const (
	OperatorAnd Operator = "AND"
	OperatorOr  Operator = "OR"
)

type UserRolePolicy struct {
	Operator        Operator `yaml:"operator" json:"operator,omitempty"`
	EnumeratedValue `yaml:",inline" json:",omitempty"`
}

// UserPolicy represents a policy that checks if a user is in a list of allowed users (OR) or if the user is equal to a specific list of values (AND).
type UserPolicy UserRolePolicy

type RolePolicy UserRolePolicy

type CallPolicy struct {
	Operator      Operator `yaml:"operator" json:"operator,omitempty"`
	IntervalValue `yaml:",inline" json:",omitempty"`
}

type StoragePolicy struct {
	Operator        Operator `yaml:"operator" json:"operator,omitempty"`
	EnumeratedValue `yaml:",inline" json:",omitempty"`
}

type TimelinessPolicy struct {
	Operator      Operator `yaml:"operator" json:"operator,omitempty"`
	IntervalValue `yaml:",inline" json:",omitempty"`
}

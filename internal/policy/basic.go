package policy

type Operator string

type Interval struct {
	Min           int    `yaml:"min"`
	Max           int    `yaml:"max"`
	UnitOfMeasure string `yaml:"unit_of_measure,omitempty"`
}

type IntervalValue struct {
	Value []Interval `yaml:"value"`
}

type EnumeratedValue struct {
	Value []string `yaml:"value"`
}

const (
	OperatorAnd Operator = "AND"
	OperatorOr  Operator = "OR"
)

type UserRolePolicy struct {
	Operator        Operator `yaml:"operator"`
	EnumeratedValue `yaml:"inline"`
}

// UserPolicy represents a policy that checks if a user is in a list of allowed users (OR) or if the user is equal to a specific list of values (AND).
type UserPolicy UserRolePolicy

type RolePolicy UserRolePolicy

type CallPolicy struct {
	Operator      Operator `yaml:"operator"`
	IntervalValue `yaml:"inline"`
}

type StoragePolicy struct {
	Operator        Operator `yaml:"operator"`
	EnumeratedValue `yaml:"inline"`
}

type TimelinessPolicy struct {
	Operator      Operator `yaml:"operator"`
	IntervalValue `yaml:"inline"`
}

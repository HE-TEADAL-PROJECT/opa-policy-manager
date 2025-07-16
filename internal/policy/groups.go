package policy

// Represent a policy clauses, which can contains at most one of each type of policy
type PolicyClause struct {
	UserPolicy            *UserPolicy       `yaml:"user" json:"user"`
	RolePolicy            *RolePolicy       `yaml:"roles" json:"roles"`
	StorageLocationPolicy *StoragePolicy    `yaml:"storage_location" json:"storage_location"`
	CallPolicy            *CallPolicy       `yaml:"call" json:"call"`
	TimelinessPolicy      *TimelinessPolicy `yaml:"timeliness" json:"timeliness"`
}

// GeneralPolicies represents a collection of policy clauses that should applied to all paths and endpoints
// It applies the policies to all endpoints, but it can be extended (AND) with specialized policies for specific paths
type GeneralPolicies struct {
	Policies         []PolicyClause
	SpecializedPaths map[string]PathPolicies
}

func NewGeneralPolicies() *GeneralPolicies {
	return &GeneralPolicies{
		Policies:         make([]PolicyClause, 0),
		SpecializedPaths: make(map[string]PathPolicies),
	}
}

// PathPolicies represents a collection of policy clauses that should applied to a specific path.
type PathPolicies struct {
	Policies           []PolicyClause
	Path               string
	SpecializedMethods map[string]PathMethodPolicies
}

type PathMethodPolicies struct {
	Policies []PolicyClause
	Path     string
	Method   string
}

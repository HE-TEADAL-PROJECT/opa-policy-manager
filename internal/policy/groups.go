package policy

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
)

// Represent a policy clauses, which can contains at most one of each type of policy
type PolicyClause struct {
	UserPolicy            *UserPolicy            `yaml:"user"`
	RolePolicy            *RolePolicy            `yaml:"roles"`
	StorageLocationPolicy *StorageLocationPolicy `yaml:"storage_location"`
	CallPolicy            *CallPolicy            `yaml:"call"`
	TimelinessPolicy      *TimelinessPolicy      `yaml:"timeliness"`
}

func (p *PolicyClause) ToRego() string {
	var result string
	if p.UserPolicy != nil {
		result += p.UserPolicy.ToRego()
	}
	if p.RolePolicy != nil {
		result += p.RolePolicy.ToRego()
	}
	if p.StorageLocationPolicy != nil {
		result += p.StorageLocationPolicy.ToRego()
	}
	if p.CallPolicy != nil {
		result += p.CallPolicy.ToRego()
	}
	if p.TimelinessPolicy != nil {
		result += p.TimelinessPolicy.ToRego()
	}
	return result
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

func (p *GeneralPolicies) ToRego() string {
	// Handle excludedPaths
	rules := make([]string, 0, len(p.Policies)+len(p.SpecializedPaths))
	if len(p.Policies) > 0 {
		rules = append(rules, p.buildGeneralRules()...)
	}
	if len(p.SpecializedPaths) > 0 {
		rules = append(rules, p.buildPathsRules()...)
	}
	for i, rule := range rules {
		rules[i] = "allow_request if {\n" + rule + "}\n"
	}
	// Return
	return strings.Join(rules, "\n")
}

func (p *GeneralPolicies) buildGeneralRules() []string {
	rules := make([]string, 0, len(p.Policies))
	excludedPaths := slices.Collect(maps.Keys(p.SpecializedPaths))
	excludedPathsJson, err := json.Marshal(excludedPaths)
	if err != nil {
		panic(err)
	}
	for _, policy := range p.Policies {
		rule := policy.ToRego()
		if len(excludedPaths) > 0 {
			rule += "not path in " + string(excludedPathsJson) + "\n"
		}
		rules = append(rules, rule)
	}
	return rules
}

func (p *GeneralPolicies) buildPathsRules() []string {
	pathRules := make([]string, 0, len(p.SpecializedPaths))
	for _, path := range p.SpecializedPaths {
		pathRules = append(pathRules, path.ToRego()...)
	}
	// No general policies, return only specialized ones
	if len(p.Policies) == 0 {
		return pathRules
	}
	// Add general policies to specialized ones
	rules := make([]string, 0, len(p.Policies)*len(pathRules))
	for _, policy := range p.Policies {
		rule := policy.ToRego()
		for _, pathRule := range pathRules {
			rules = append(rules, rule+pathRule)
		}
	}
	return rules
}

// PathPolicies represents a collection of policy clauses that should applied to a specific path.
type PathPolicies struct {
	Policies           []PolicyClause
	Path               string
	SpecializedMethods map[string]PathMethodPolicies
}

func (p *PathPolicies) ToRego() []string {
	if len(p.Policies) == 0 && len(p.SpecializedMethods) == 0 {
		return []string{}
	}
	pathJson, err := json.Marshal(p.Path)
	if err != nil {
		panic(err)
	}
	specializedMethods := slices.Collect(maps.Keys(p.SpecializedMethods))
	specializedMethodsJson, err := json.Marshal(specializedMethods)
	if err != nil {
		panic(err)
	}
	blocks := make([]string, 0, len(p.Policies)+len(p.SpecializedMethods))
	for _, policy := range p.Policies {
		// Add general path rules
		policyCode := "path == " + string(pathJson) + "\n"
		policyCode += policy.ToRego()
		if len(specializedMethods) > 0 {
			policyCode += "not method in " + string(specializedMethodsJson) + "\n"
		}
		blocks = append(blocks, policyCode)

		// Add specialized methods rules
		for _, method := range p.SpecializedMethods {
			policyCode := "path == " + string(pathJson) + "\n"
			policyCode += policy.ToRego()
			for _, methodPolicy := range method.ToRego() {
				blocks = append(blocks, policyCode+methodPolicy)
			}
		}

	}
	return blocks
}

type PathMethodPolicies struct {
	Policies []PolicyClause
	Path     string
	Method   string
}

func (p *PathMethodPolicies) ToRego() []string {
	if len(p.Policies) == 0 {
		return []string{}
	}
	methodJson, err := json.Marshal(p.Method)
	if err != nil {
		panic(err)
	}
	blocks := make([]string, 0, len(p.Policies))
	for _, policy := range p.Policies {
		policyCode := "method == " + string(methodJson) + "\n"
		policyCode += policy.ToRego()
		blocks = append(blocks, policyCode)
	}
	return blocks
}

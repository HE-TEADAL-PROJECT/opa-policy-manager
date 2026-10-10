package policy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Represent a policy clauses, which can contains at most one of each type of policy
type PolicyClause struct {
	UserPolicy       *UserPolicy       `yaml:"users"`
	RolePolicy       *RolePolicy       `yaml:"roles"`
	LocationPolicy   *LocationPolicy   `yaml:"location"`
	CallPolicy       *CallPolicy       `yaml:"calls"`
	TimelinessPolicy *TimelinessPolicy `yaml:"timeliness"`
}

func (p *PolicyClause) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node for PolicyClause, got kind %d", value.Kind)
	}
	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]
		switch keyNode.Value {
		case "user", "users":
			var up UserPolicy
			if err := valNode.Decode(&up); err != nil {
				return err
			}
			p.UserPolicy = &up
		case "role", "roles":
			var rp RolePolicy
			if err := valNode.Decode(&rp); err != nil {
				return err
			}
			p.RolePolicy = &rp
		case "location", "locations", "storage_location", "storage-location":
			var lp LocationPolicy
			if err := valNode.Decode(&lp); err != nil {
				return err
			}
			p.LocationPolicy = &lp
		case "call", "calls":
			var cp CallPolicy
			if err := valNode.Decode(&cp); err != nil {
				return err
			}
			p.CallPolicy = &cp
		case "timeliness":
			var tp TimelinessPolicy
			if err := valNode.Decode(&tp); err != nil {
				return err
			}
			p.TimelinessPolicy = &tp
		}
	}
	return nil
}

func (p *PolicyClause) ToRego() string {
	var result string
	// addClause tracks a sub-policy that was present in the clause but translated to no rego at all
	addClause := func(kind string, rego string) {
		if rego == "" {
			slog.Warn("Policy clause translated to empty rego, the constraint is dropped from the generated policy",
				"clause", kind)
			return
		}
		result += rego
	}
	recognized := 0
	if p.UserPolicy != nil {
		recognized++
		addClause("users", p.UserPolicy.ToRego())
	}
	if p.RolePolicy != nil {
		recognized++
		addClause("roles", p.RolePolicy.ToRego())
	}
	if p.LocationPolicy != nil {
		recognized++
		addClause("location", p.LocationPolicy.ToRego())
	}
	if p.CallPolicy != nil {
		recognized++
		addClause("calls", p.CallPolicy.ToRego())
	}
	if p.TimelinessPolicy != nil {
		recognized++
		addClause("timeliness", p.TimelinessPolicy.ToRego())
	}
	if recognized == 0 {
		slog.Warn("Policy clause declares no recognized policy type, the clause is dropped from the generated policy",
			"recognizedClauses", 0, "expectedClauseKeys", []string{"users", "roles", "location", "calls", "timeliness"})
	}
	return result
}

// paramSegment matches one {param} path-template segment.
var paramSegment = regexp.MustCompile(`\{[^/{}]+\}`)

// placeholder substitued for {param}
const paramPlaceholder = "ZZZPATHPARAMZZZ"

// PathTemplateToRegex converts an OpenAPI path template such as "/anything/{anything}" into an anchored regex
// ("^/anything/[^/]+$") that matches the CONCRETE request path Envoy hands to OPA.
func PathTemplateToRegex(tmpl string) string {
	withPlaceholders := paramSegment.ReplaceAllString(tmpl, paramPlaceholder)
	escaped := regexp.QuoteMeta(withPlaceholders)
	escaped = strings.ReplaceAll(escaped, paramPlaceholder, `[^/]+`)
	return "^" + escaped + "$"
}

// translateClauses translates each clause into a rule body.
// A clause that translates to no rego is dropped: it would produce a rule body that allows every request.
func translateClauses(clauses []PolicyClause, scope string) []string {
	result := make([]string, 0, len(clauses))
	for i, clause := range clauses {
		body := clause.ToRego()
		if body == "" {
			slog.Warn("Clause translated to empty rego, no rule generated for it", "scope", scope, "index", i)
			continue
		}
		result = append(result, body)
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
	rules := make([]string, 0, len(p.Policies)+len(p.SpecializedPaths))
	generalRules := translateClauses(p.Policies, "paths")
	if len(generalRules) > 0 {
		rules = append(rules, p.buildGeneralRules(generalRules)...)
	}
	if len(p.SpecializedPaths) > 0 {
		rules = append(rules, p.buildPathsRules(generalRules)...)
	}
	slog.Debug("Translating parsed policies to rego rules",
		"generalClauses", len(p.Policies), "specializedPaths", len(p.SpecializedPaths),
		"generatedRules", len(rules))
	if len(rules) == 0 {
		slog.Warn("No rego rule generated, the service will deny every request",
			"generalClauses", len(p.Policies), "specializedPaths", len(p.SpecializedPaths))
		return ""
	}
	for i, rule := range rules {
		rules[i] = "allow_request if {\n" + rule + "}\n"
	}
	// Return
	return strings.Join(rules, "\n")
}

func (p *GeneralPolicies) buildGeneralRules(generalRules []string) []string {
	rules := make([]string, 0, len(generalRules))
	// Sorted, so the generated rego does not depend on map iteration order.
	excludedPaths := slices.Sorted(maps.Keys(p.SpecializedPaths))

	// excluded paths may contain {params}
	excludedPatterns := make([]string, len(excludedPaths))
	for i, ep := range excludedPaths {
		excludedPatterns[i] = PathTemplateToRegex(ep)
	}

	excludedPatternsJson, err := json.Marshal(excludedPatterns)
	if err != nil {
		panic(err)
	}

	for _, r := range generalRules {
		if len(excludedPaths) > 0 {
			r += "count([p | p := " + string(excludedPatternsJson) + "[_]; regex.match(p, path)]) == 0\n"
		}
		rules = append(rules, r)
	}
	return rules
}

func (p *GeneralPolicies) buildPathsRules(generalRules []string) []string {
	pathRules := make([]string, 0, len(p.SpecializedPaths))
	// Sorted, so the generated rego does not depend on map iteration order.
	for _, pathKey := range slices.Sorted(maps.Keys(p.SpecializedPaths)) {
		path := p.SpecializedPaths[pathKey]
		pathRules = append(pathRules, path.ToRego()...)
	}
	// No general policies, return only specialized ones
	if len(generalRules) == 0 {
		return pathRules
	}
	// Add general policies to specialized ones
	rules := make([]string, 0, len(generalRules)*len(pathRules))
	for _, generalRule := range generalRules {
		for _, pathRule := range pathRules {
			rules = append(rules, generalRule+pathRule)
		}
	}
	return rules
}

// PathPolicies represents a collection of policy clauses that should applied to a specific path
// which may contain {params}
type PathPolicies struct {
	Policies           []PolicyClause
	Path               string
	SpecializedMethods map[string]PathMethodPolicies
}

func (p *PathPolicies) ToRego() []string {
	if len(p.Policies) == 0 && len(p.SpecializedMethods) == 0 {
		return []string{}
	}
	pathPattern, err := json.Marshal(PathTemplateToRegex(p.Path))
	if err != nil {
		panic(err)
	}
	specializedMethods := slices.Sorted(maps.Keys(p.SpecializedMethods))
	specializedMethodsJson, err := json.Marshal(specializedMethods)
	if err != nil {
		panic(err)
	}
	slog.Debug("Translating path policies to rego rules",
		"path", p.Path, "pathRegex", PathTemplateToRegex(p.Path),
		"clauses", len(p.Policies), "specializedMethods", specializedMethods)

	pathMatch := "regex.match(" + string(pathPattern) + ", path)\n"

	// The path-level clauses are translated once and reused: they appear in the
	// rule covering the non-specialized methods and, when present, in every
	// method rule as an additional restriction.
	pathClauses := translateClauses(p.Policies, "path:"+p.Path)

	blocks := make([]string, 0, len(p.Policies)+len(p.SpecializedMethods))
	// Rules for the methods that do NOT declare their own policy. A path with
	// several clauses yields one alternative rule per clause.
	for _, clauseRego := range pathClauses {
		policyCode := pathMatch + clauseRego
		if len(specializedMethods) > 0 {
			policyCode += "not method in " + string(specializedMethodsJson) + "\n"
		}
		blocks = append(blocks, policyCode)
	}

	// Rules for the methods that DO declare their own policy. They are
	// self-contained when the path declares no path-level clause, otherwise the
	// path clause narrows them down further, one alternative per path clause.
	methodPrefixes := make([]string, 0, len(pathClauses))
	if len(pathClauses) == 0 {
		slog.Debug("Path has no path-level policy, method policies apply on their own",
			"path", p.Path, "specializedMethods", specializedMethods)
		methodPrefixes = append(methodPrefixes, pathMatch)
	} else {
		for _, clauseRego := range pathClauses {
			methodPrefixes = append(methodPrefixes, pathMatch+clauseRego)
		}
	}
	for _, methodKey := range specializedMethods {
		method := p.SpecializedMethods[methodKey]
		for _, methodPolicy := range method.ToRego() {
			for _, methodPrefix := range methodPrefixes {
				blocks = append(blocks, methodPrefix+methodPolicy)
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
	methodMatch := "method == " + string(methodJson) + "\n"
	methodClauses := translateClauses(p.Policies, p.Method+":"+p.Path)
	blocks := make([]string, 0, len(methodClauses))
	for _, clause := range methodClauses {
		blocks = append(blocks, methodMatch+clause)
	}
	return blocks
}

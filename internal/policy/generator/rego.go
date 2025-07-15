package generator

import (
	"encoding/json"
	"fmt"
	"strings"

	. "dspn-regogenerator/internal/policy"
)

type ServiceData struct{}

// Accept any of the policy types defined in the policy package and return the Rego representation.
func GenerateRego(serviceData ServiceData, policies []any) (string, error) {
	var regoPolicies []string
	var rego string
	var err error

	for _, policy := range policies {
		switch p := policy.(type) {
		case UserPolicy, RolePolicy, CallPolicy, StoragePolicy, TimelinessPolicy:
			rego, err = generateBasicPolicyRego(serviceData, p)
		default:
			return "", fmt.Errorf("unsupported policy type: %T", p)
		}
		if err != nil {
			return "", fmt.Errorf("error generating Rego for policy %T: %w", policy, err)
		}
		regoPolicies = append(regoPolicies, rego)
	}

	return strings.Join(regoPolicies, "\n"), nil
}

func generateBasicPolicyRego(serviceData ServiceData, policy any) (string, error) {
	var rego string
	var err error

	switch p := policy.(type) {
	case UserPolicy:
		if len(p.Value) == 0 {
			return "", fmt.Errorf("policy value cannot be empty for UserPolicy")
		}
		userSet := arrayToRegoSet(p.Value)
		switch p.Operator {
		case OperatorAnd:
			rego = fmt.Sprintf("{user} == %s", userSet)
		case OperatorOr:
			rego = fmt.Sprintf("{user} in %s", userSet)
		default:
			err = fmt.Errorf("unsupported operator for UserPolicy: %s", p.Operator)
		}
	case RolePolicy:
		if len(p.Value) == 0 {
			return "", fmt.Errorf("policy value cannot be empty for RolePolicy")
		}
		roleSet := arrayToRegoSet(p.Value)
		switch p.Operator {
		case OperatorAnd:
			rego = fmt.Sprintf("roles == %s", string(roleSet))
		case OperatorOr:
			rego = fmt.Sprintf("roles & %s != set()", string(roleSet))
		default:
			err = fmt.Errorf("unsupported operator for RolePolicy: %s", p.Operator)
		}
	case StoragePolicy, TimelinessPolicy, CallPolicy:
		// TODO: Implement Rego generation for StoragePolicy, TimelinessPolicy, and CallPolicy
		break
	default:
		err = fmt.Errorf("unsupported policy type: %T", p)
	}

	return rego, err
}

func arrayToRegoSet(arr []string) string {
	if len(arr) == 0 {
		return "set()"
	}
	jsonArray, err := json.Marshal(arr)
	if err != nil {
		panic(err)
	}
	// Substitute square brackets with curly braces for Rego set syntax
	jsonArray[0] = '{'
	jsonArray[len(jsonArray)-1] = '}'
	return string(jsonArray)
}

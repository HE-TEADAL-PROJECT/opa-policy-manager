package generator

import (
	. "dspn-regogenerator/internal/policy"
	"testing"
)

func TestGenerateRego(t *testing.T) {
	tt := []struct {
		name     string
		policies []any
		expected string
	}{
		{
			name: "UserPolicy with AND operator",
			policies: []any{
				UserPolicy{
					Operator: OperatorAnd,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"user1", "user2"},
					},
				},
			},
			expected: "{user} == {\"user1\",\"user2\"}",
		},
		{
			name: "UserPolicy with OR operator",
			policies: []any{
				UserPolicy{
					Operator: OperatorOr,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"user1", "user2"},
					},
				},
			},
			expected: "{user} in {\"user1\",\"user2\"}",
		},
		{
			name: "RolePolicy with AND operator",
			policies: []any{
				RolePolicy{
					Operator: OperatorAnd,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"role1", "role2"},
					},
				},
			},
			expected: "roles == {\"role1\",\"role2\"}",
		},
		{
			name: "RolePolicy with OR operator",
			policies: []any{
				RolePolicy{
					Operator: OperatorOr,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"role1", "role2"},
					},
				},
			},
			expected: "roles & {\"role1\",\"role2\"} != set()",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			regopolicies, err := GenerateRego(ServiceData{}, tc.policies)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if regopolicies != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, regopolicies)
			}
		})
	}
}

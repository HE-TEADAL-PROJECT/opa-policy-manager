package policy_test

import (
	"dspn-regogenerator/policy"
	"testing"
)

type testCase struct {
	name string
	pol  policy.Policy
	want string
}

func TestUserPolicy(t *testing.T) {
	tests := []testCase{
		{
			name: "Test with AND",
			pol: &policy.UserPolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{"user1", "user2"},
					Operator: policy.OperatorAnd,
				},
			},
			want: "input.user == user1\ninput.user == user2\n",
		},
		{
			name: "Test with OR",
			pol: &policy.UserPolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{"user1", "user2"},
					Operator: policy.OperatorOr,
				},
			},
			want: "input.user in [\"user1\",\"user2\"]\n",
		},
		{
			name: "Test with AND and empty user list",
			pol: &policy.UserPolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{},
					Operator: policy.OperatorAnd,
				},
			},
			want: "",
		},
		{
			name: "Test with OR and empty user list",
			pol: &policy.UserPolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{},
					Operator: policy.OperatorOr,
				},
			},
			want: "input.user in []\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pol.ToRego()
			if got != test.want {
				t.Errorf("ToRego() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRolePolicy(t *testing.T) {
	tests := []testCase{
		{
			name: "Test with AND",
			pol: &policy.RolePolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{"role1", "role2"},
					Operator: policy.OperatorAnd,
				},
			},
			want: "input.role == role1\ninput.role == role2\n",
		},
		{
			name: "Test with OR",
			pol: &policy.RolePolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{"role1", "role2"},
					Operator: policy.OperatorOr,
				},
			},
			want: "input.role in [\"role1\",\"role2\"]\n",
		},
		{
			name: "Test with AND and empty user list",
			pol: &policy.RolePolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{},
					Operator: policy.OperatorAnd,
				},
			},
			want: "",
		},
		{
			name: "Test with OR and empty user list",
			pol: &policy.RolePolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{},
					Operator: policy.OperatorOr,
				},
			},
			want: "input.role in []\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pol.ToRego()
			if got != test.want {
				t.Errorf("ToRego() = %v, want %v", got, test.want)
			}
		})
	}
}

package policy_test

import (
	"dspn-regogenerator/internal/policy"
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
			want: "user == user1\nuser == user2\n",
		},
		{
			name: "Test with OR",
			pol: &policy.UserPolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{"user1", "user2"},
					Operator: policy.OperatorOr,
				},
			},
			want: "user in [\"user1\",\"user2\"]\n",
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
			want: "user in []\n",
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
			want: `count({"role1","role2"}-roles) == 0` + "\n",
		},
		{
			name: "Test with OR",
			pol: &policy.RolePolicy{
				PolicyDetail: policy.PolicyDetail{
					Value:    []string{"role1", "role2"},
					Operator: policy.OperatorOr,
				},
			},
			want: `count({"role1","role2"}&roles) != 0` + "\n",
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
			want: "",
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

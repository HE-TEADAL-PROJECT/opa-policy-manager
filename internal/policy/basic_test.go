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
			want: "user == \"user1\"\nuser == \"user2\"\n",
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

func TestCallTimePolicy(t *testing.T) {
	minH := 8
	maxH := 18
	tests := []testCase{
		{
			name: "Test with AND",
			pol: &policy.CallTimePolicy{
				Operator: policy.OperatorAnd,
				Value: []policy.RateLimitEntry{
					{
						Min:  &minH,
						Max:  &maxH,
						Unit: "h",
					},
				},
			},
			want: "((time.clock(time.now_ns())[0] * 60) + time.clock(time.now_ns())[1]) >= 480\n((time.clock(time.now_ns())[0] * 60) + time.clock(time.now_ns())[1]) <= 1080\n",
		},
		{
			name: "Test with OR",
			pol: &policy.CallTimePolicy{
				Operator: policy.OperatorOr,
				Value: []policy.RateLimitEntry{
					{
						Min:  &minH,
						Max:  &maxH,
						Unit: "h",
					},
				},
			},
			want: "count([w | w := [{\"min\":480,\"max\":1080}][_]; ((time.clock(time.now_ns())[0] * 60) + time.clock(time.now_ns())[1]) >= w.min; ((time.clock(time.now_ns())[0] * 60) + time.clock(time.now_ns())[1]) <= w.max]) > 0\n",
		},
		{
			name: "Test with empty value list",
			pol: &policy.CallTimePolicy{
				Operator: policy.OperatorAnd,
				Value:    []policy.RateLimitEntry{},
			},
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pol.ToRego()
			if got != test.want {
				t.Errorf("ToRego() = %q, want %q", got, test.want)
			}
		})
	}
}


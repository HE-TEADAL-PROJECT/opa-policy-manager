package policy_test

import (
	"dspn-regogenerator/internal/policy"
	"testing"
)

type groupTestCase struct {
	name string
	pol  interface {
		ToRego() []string
	}
	want []string
}

func TestPolicyClause(t *testing.T) {
	tests := []testCase{
		{
			name: "empty",
			pol:  &policy.PolicyClause{},
			want: "",
		},
		{
			name: "user policy",
			pol: &policy.PolicyClause{
				UserPolicy: &policy.UserPolicy{
					PolicyDetail: policy.PolicyDetail{
						Operator: policy.OperatorOr,
						Value:    []string{"user1", "user2"},
					},
				},
			},
			want: "input.user in [\"user1\",\"user2\"]\n",
		},
		{
			name: "role policy",
			pol: &policy.PolicyClause{
				RolePolicy: &policy.RolePolicy{
					PolicyDetail: policy.PolicyDetail{
						Operator: policy.OperatorAnd,
						Value:    []string{"role1", "role2"},
					},
				},
			},
			want: "input.role == role1\ninput.role == role2\n",
		},
		{
			name: "user and role policy",
			pol: &policy.PolicyClause{
				UserPolicy: &policy.UserPolicy{
					PolicyDetail: policy.PolicyDetail{
						Operator: policy.OperatorOr,
						Value:    []string{"user1", "user2"},
					},
				},
				RolePolicy: &policy.RolePolicy{
					PolicyDetail: policy.PolicyDetail{
						Operator: policy.OperatorAnd,
						Value:    []string{"role1", "role2"},
					},
				},
			},
			want: "input.user in [\"user1\",\"user2\"]\ninput.role == role1\ninput.role == role2\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pol.ToRego()
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestGeneralPolicies(t *testing.T) {
	tests := []testCase{
		{
			name: "empty",
			pol:  &policy.GeneralPolicies{},
			want: "",
		},
		{
			name: "one clause",
			pol: &policy.GeneralPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
						RolePolicy: &policy.RolePolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorAnd,
								Value:    []string{"role1", "role2"},
							},
						},
					},
				},
			},
			want: "allow if {\ninput.user in [\"user1\",\"user2\"]\ninput.role == role1\ninput.role == role2\n}\n",
		},
		{
			name: "two clauses",
			pol: &policy.GeneralPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
						RolePolicy: &policy.RolePolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorAnd,
								Value:    []string{"role1", "role2"},
							},
						},
					},
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user3", "user4"},
							},
						},
					},
				},
			},
			// Two clauses should generate separate allow statements (OR condition)
			want: "allow if {\ninput.user in [\"user1\",\"user2\"]\ninput.role == role1\ninput.role == role2\n}\n\nallow if {\ninput.user in [\"user3\",\"user4\"]\n}\n",
		},
		{
			name: "with excluded paths",
			pol: &policy.GeneralPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
						RolePolicy: &policy.RolePolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorAnd,
								Value:    []string{"role1", "role2"},
							},
						},
					},
				},
				SpecializedPaths: map[string]policy.PathPolicies{
					"/path1": {
						Policies: []policy.PolicyClause{
							{
								UserPolicy: &policy.UserPolicy{
									PolicyDetail: policy.PolicyDetail{
										Operator: policy.OperatorOr,
										Value:    []string{"user3", "user4"},
									},
								},
							},
						},
						Path: "/path1",
					},
				},
			},
			want: "allow if {\ninput.user in [\"user1\",\"user2\"]\ninput.role == role1\ninput.role == role2\ninput.path not in [\"/path1\"]\n}\n\nallow if {\ninput.user in [\"user1\",\"user2\"]\ninput.role == role1\ninput.role == role2\ninput.path == \"/path1\"\ninput.user in [\"user3\",\"user4\"]\n}\n",
		},
		{
			name: "only specialized paths",
			pol: &policy.GeneralPolicies{
				SpecializedPaths: map[string]policy.PathPolicies{
					"/path1": {
						Policies: []policy.PolicyClause{
							{
								UserPolicy: &policy.UserPolicy{
									PolicyDetail: policy.PolicyDetail{
										Operator: policy.OperatorOr,
										Value:    []string{"user3", "user4"},
									},
								},
							},
						},
						Path: "/path1",
					},
				},
			},
			want: "allow if {\ninput.path == \"/path1\"\ninput.user in [\"user3\",\"user4\"]\n}\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pol.ToRego()
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestPathPolicies(t *testing.T) {
	tests := []groupTestCase{
		{
			name: "empty",
			pol:  &policy.PathPolicies{},
			want: []string{},
		},
		{
			name: "one clause",
			pol: &policy.PathPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
					},
				},
				Path: "/path1",
			},
			// This should generate the policies with a path condition
			want: []string{"input.path == \"/path1\"\ninput.user in [\"user1\",\"user2\"]\n"},
		},
		{
			name: "two clauses",
			pol: &policy.PathPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
					},
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user3", "user4"},
							},
						},
					},
				},

				Path: "/path1",
			},
			want: []string{"input.path == \"/path1\"\ninput.user in [\"user1\",\"user2\"]\n",
				"input.path == \"/path1\"\ninput.user in [\"user3\",\"user4\"]\n"},
		},
		{
			name: "with specialized methods",
			pol: &policy.PathPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
					},
				},

				Path: "/path1",
				SpecializedMethods: map[string]policy.PathMethodPolicies{
					"GET": {
						Policies: []policy.PolicyClause{
							{
								UserPolicy: &policy.UserPolicy{
									PolicyDetail: policy.PolicyDetail{
										Operator: policy.OperatorOr,
										Value:    []string{"user5", "user6"},
									},
								},
							},
						},
						Path:   "/path1",
						Method: "GET",
					},
				},
			},
			want: []string{"input.path == \"/path1\"\ninput.user in [\"user1\",\"user2\"]\ninput.method not in [\"GET\"]\n",
				"input.path == \"/path1\"\ninput.user in [\"user1\",\"user2\"]\ninput.method == \"GET\"\ninput.user in [\"user5\",\"user6\"]\n"},
		},
		{
			name: "only specialized methods",
			pol: &policy.PathPolicies{
				SpecializedMethods: map[string]policy.PathMethodPolicies{
					"GET": {
						Policies: []policy.PolicyClause{
							{
								UserPolicy: &policy.UserPolicy{
									PolicyDetail: policy.PolicyDetail{
										Operator: policy.OperatorOr,
										Value:    []string{"user5", "user6"},
									},
								},
							},
						},
						Method: "GET",
					},
					"POST": {
						Policies: []policy.PolicyClause{
							{
								UserPolicy: &policy.UserPolicy{
									PolicyDetail: policy.PolicyDetail{
										Operator: policy.OperatorOr,
										Value:    []string{"user7", "user8"},
									},
								},
							},
						},
						Method: "POST",
					},
				},
				Path: "/path1",
			},
			want: []string{"input.path == \"/path1\"\ninput.method == \"GET\"\ninput.user in [\"user5\",\"user6\"]\n",
				"input.path == \"/path1\"\ninput.method == \"POST\"\ninput.user in [\"user7\",\"user8\"]\n"},
		},
		{
			name: "two clauses with two specialized methods",
			pol: &policy.PathPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
					},
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user3", "user4"},
							},
						},
					},
				},
				SpecializedMethods: map[string]policy.PathMethodPolicies{
					"GET": {
						Policies: []policy.PolicyClause{
							{
								RolePolicy: &policy.RolePolicy{
									PolicyDetail: policy.PolicyDetail{
										Operator: policy.OperatorOr,
										Value:    []string{"role1", "role2"},
									},
								},
							},
						},
						Method: "GET",
					},
					"POST": {
						Policies: []policy.PolicyClause{
							{
								RolePolicy: &policy.RolePolicy{
									PolicyDetail: policy.PolicyDetail{
										Operator: policy.OperatorOr,
										Value:    []string{"role3", "role4"},
									},
								},
							},
						},
						Method: "POST",
					},
				},
				Path: "/path1",
			},
			want: []string{"input.path == \"/path1\"\ninput.user in [\"user1\",\"user2\"]\ninput.method not in [\"GET\",\"POST\"]\n",
				"input.path == \"/path1\"\ninput.user in [\"user1\",\"user2\"]\ninput.method == \"GET\"\ninput.role in [\"role1\",\"role2\"]\n",
				"input.path == \"/path1\"\ninput.user in [\"user1\",\"user2\"]\ninput.method == \"POST\"\ninput.role in [\"role3\",\"role4\"]\n",
				"input.path == \"/path1\"\ninput.user in [\"user3\",\"user4\"]\ninput.method not in [\"GET\",\"POST\"]\n",
				"input.path == \"/path1\"\ninput.user in [\"user3\",\"user4\"]\ninput.method == \"GET\"\ninput.role in [\"role1\",\"role2\"]\n",
				"input.path == \"/path1\"\ninput.user in [\"user3\",\"user4\"]\ninput.method == \"POST\"\ninput.role in [\"role3\",\"role4\"]\n",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pol.ToRego()
			for i, g := range got {
				if g != test.want[i] {
					t.Errorf("policy %d:got %q, want %q", i, g, test.want[i])
				}
			}
		})
	}
}

func TestPathMethodPolicies(t *testing.T) {
	tests := []groupTestCase{
		{
			name: "empty",
			pol:  &policy.PathMethodPolicies{},
			want: []string{},
		},
		{
			name: "one clause",
			pol: &policy.PathMethodPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
					},
				},
				Method: "GET",
				Path:   "/path1",
			},
			want: []string{"input.method == \"GET\"\ninput.user in [\"user1\",\"user2\"]\n"},
		},
		{
			name: "two clauses",
			pol: &policy.PathMethodPolicies{
				Policies: []policy.PolicyClause{
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user1", "user2"},
							},
						},
					},
					{
						UserPolicy: &policy.UserPolicy{
							PolicyDetail: policy.PolicyDetail{
								Operator: policy.OperatorOr,
								Value:    []string{"user3", "user4"},
							},
						},
					},
				},
				Method: "POST",
				Path:   "/path1",
			},
			want: []string{"input.method == \"POST\"\ninput.user in [\"user1\",\"user2\"]\n",
				"input.method == \"POST\"\ninput.user in [\"user3\",\"user4\"]\n"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pol.ToRego()
			for i, g := range got {
				if g != test.want[i] {
					t.Errorf("got %q, want %q", g, test.want[i])
				}
			}
		})
	}
}

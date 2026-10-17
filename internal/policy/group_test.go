package policy_test

import (
	"testing"

	"dspn-regogenerator/internal/policy"
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
			want: "user in [\"user1\",\"user2\"]\n",
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
			want: `count({"role1","role2"}-roles) == 0` + "\n",
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
			want: "user in [\"user1\",\"user2\"]\n" + `count({"role1","role2"}-roles) == 0` + "\n",
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
			want: "allow_request if {\nuser in [\"user1\",\"user2\"]\n" + `count({"role1","role2"}-roles) == 0` + "\n" + "}\n",
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
			want: "allow_request if {\nuser in [\"user1\",\"user2\"]\n" + `count({"role1","role2"}-roles) == 0` + "\n" + "}\n\nallow_request if {\nuser in [\"user3\",\"user4\"]\n}\n",
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
			want: "allow_request if {\nuser in [\"user1\",\"user2\"]\n" + `count({"role1","role2"}-roles) == 0` + "\ncount([p | p := [\"^/path1$\"][_]; regex.match(p, path)]) == 0\n}\n\nallow_request if {\nuser in [\"user1\",\"user2\"]\n" + `count({"role1","role2"}-roles) == 0` + "\nregex.match(\"^/path1$\", path)\nuser in [\"user3\",\"user4\"]\n}\n",
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
			want: "allow_request if {\nregex.match(\"^/path1$\", path)\nuser in [\"user3\",\"user4\"]\n}\n",
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
			want: []string{"regex.match(\"^/path1$\", path)\nuser in [\"user1\",\"user2\"]\n"},
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
			want: []string{
				"regex.match(\"^/path1$\", path)\nuser in [\"user1\",\"user2\"]\n",
				"regex.match(\"^/path1$\", path)\nuser in [\"user3\",\"user4\"]\n",
			},
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
			want: []string{
				"regex.match(\"^/path1$\", path)\nuser in [\"user1\",\"user2\"]\nnot method in [\"GET\"]\n",
				"regex.match(\"^/path1$\", path)\nuser in [\"user1\",\"user2\"]\nmethod == \"GET\"\nuser in [\"user5\",\"user6\"]\n",
			},
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
			want: []string{
				"regex.match(\"^/path1$\", path)\nmethod == \"GET\"\nuser in [\"user5\",\"user6\"]\n",
				"regex.match(\"^/path1$\", path)\nmethod == \"POST\"\nuser in [\"user7\",\"user8\"]\n",
			},
		},
		{
			// A path clause that translates to no rego (e.g. an empty placeholder)
			// must not degenerate into a rule allowing every request of the path.
			name: "empty path clause generates no rule",
			pol: &policy.PathPolicies{
				Policies: []policy.PolicyClause{{}},
				Path:     "/path1",
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
			want: []string{"regex.match(\"^/path1$\", path)\nmethod == \"GET\"\nuser in [\"user5\",\"user6\"]\n"},
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
			want: []string{
				"regex.match(\"^/path1$\", path)\nuser in [\"user1\",\"user2\"]\nnot method in [\"GET\",\"POST\"]\n",
				"regex.match(\"^/path1$\", path)\nuser in [\"user3\",\"user4\"]\nnot method in [\"GET\",\"POST\"]\n",
				"regex.match(\"^/path1$\", path)\nuser in [\"user1\",\"user2\"]\nmethod == \"GET\"\n" + `count({"role1","role2"}&roles) != 0` + "\n",
				"regex.match(\"^/path1$\", path)\nuser in [\"user3\",\"user4\"]\nmethod == \"GET\"\n" + `count({"role1","role2"}&roles) != 0` + "\n",
				"regex.match(\"^/path1$\", path)\nuser in [\"user1\",\"user2\"]\nmethod == \"POST\"\n" + `count({"role3","role4"}&roles) != 0` + "\n",
				"regex.match(\"^/path1$\", path)\nuser in [\"user3\",\"user4\"]\nmethod == \"POST\"\n" + `count({"role3","role4"}&roles) != 0` + "\n",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pol.ToRego()
			if len(got) != len(test.want) {
				t.Errorf("got %d rules, want %d: %q", len(got), len(test.want), got)
			}
			for i, g := range got {
				if i >= len(test.want) {
					break
				}
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
			want: []string{"method == \"GET\"\nuser in [\"user1\",\"user2\"]\n"},
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
			want: []string{
				"method == \"POST\"\nuser in [\"user1\",\"user2\"]\n",
				"method == \"POST\"\nuser in [\"user3\",\"user4\"]\n",
			},
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

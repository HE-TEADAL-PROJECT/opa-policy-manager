// Copyright 2025 Matteo Brambilla - TEADAL
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	. "dspn-regogenerator/internal/policy"
	"regexp"
	"testing"
)

func TestGeneratePolicyRego(t *testing.T) {
	tt := []struct {
		name     string
		policies any
		expected string
	}{
		{
			name: "UserPolicy with AND operator",
			policies: UserPolicy{
				Operator: OperatorAnd,
				EnumeratedValue: EnumeratedValue{
					Value: []string{"user1", "user2"},
				},
			},

			expected: "{user} == {\"user1\",\"user2\"}",
		},
		{
			name: "UserPolicy with OR operator",
			policies: UserPolicy{
				Operator: OperatorOr,
				EnumeratedValue: EnumeratedValue{
					Value: []string{"user1", "user2"},
				},
			},
			expected: "user in {\"user1\",\"user2\"}",
		},
		{
			name: "RolePolicy with AND operator",
			policies: RolePolicy{
				Operator: OperatorAnd,
				EnumeratedValue: EnumeratedValue{
					Value: []string{"role1", "role2"},
				},
			},
			expected: "roles == {\"role1\",\"role2\"}",
		},
		{
			name: "RolePolicy with OR operator",
			policies: RolePolicy{
				Operator: OperatorOr,
				EnumeratedValue: EnumeratedValue{
					Value: []string{"role1", "role2"},
				},
			},
			expected: "roles & {\"role1\",\"role2\"} != set()",
		},
		{
			name: "PolicyClause with UserPolicy",
			policies: PolicyClause{
				UserPolicy: &UserPolicy{
					Operator: OperatorAnd,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"user1", "user2"},
					},
				},
			},
			expected: "{user} == {\"user1\",\"user2\"}",
		},
		{
			name: "PolicyClause with UserPolicy and RolePolicy",
			policies: PolicyClause{
				UserPolicy: &UserPolicy{
					Operator: OperatorAnd,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"user1", "user2"},
					},
				},
				RolePolicy: &RolePolicy{
					Operator: OperatorOr,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"role1", "role2"},
					},
				},
			},
			expected: "{user} == {\"user1\",\"user2\"}\nroles & {\"role1\",\"role2\"} != set()",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			regopolicies, err := GeneratePolicyRego(ServiceData{}, tc.policies)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if regopolicies != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, regopolicies)
			}
		})
	}
}

func TestGenerateMethodPolicies(t *testing.T) {
	tt := []struct {
		name     string
		method   string
		policies PathMethodPolicies
		expected []string
	}{
		{
			name:   "MethodPolicies with UserPolicy",
			method: "get",
			policies: PathMethodPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
				},
			},
			expected: []string{"method == \"GET\"\n{user} == {\"user1\",\"user2\"}"},
		},
		{
			name:   "MethodPolicies with multiple clause",
			method: "post",
			policies: PathMethodPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorOr,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user3", "user4"},
							},
						},
					},
				},
			},
			expected: []string{
				"method == \"POST\"\n{user} == {\"user1\",\"user2\"}",
				"method == \"POST\"\nuser in {\"user3\",\"user4\"}",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			regopolicies, err := generatePoliciesForMethod(ServiceData{}, tc.method, tc.policies)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(regopolicies) == 0 {
				t.Fatal("expected non-empty policies")
			}
			for i, rego := range regopolicies {
				if rego != tc.expected[i] {
					t.Errorf("%d: expected %s, got %s", i, tc.expected[i], rego)
				}
			}
		})
	}
}

func TestGeneratePathPolicies(t *testing.T) {
	tt := []struct {
		name     string
		path     string
		policies PathPolicies
		expected []string
	}{
		{
			name: "PathPolicies non specialized with UserPolicy",
			path: "/example",
			policies: PathPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
				},
			},
			expected: []string{
				"path == \"/example\"\n{user} == {\"user1\",\"user2\"}",
			},
		},
		{
			name: "PathPolicies non specialized with multiple clauses",
			path: "/example",
			policies: PathPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user3", "user4"},
							},
						},
					},
				},
			},
			expected: []string{
				"path == \"/example\"\n{user} == {\"user1\",\"user2\"}",
				"path == \"/example\"\n{user} == {\"user3\",\"user4\"}",
			},
		},
		{
			name: "PathPolicies with UserPolicy and empty specialized method clauses",
			path: "/example",
			policies: PathPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
				},
				SpecializedMethods: map[string]PathMethodPolicies{
					"get": {
						Policies: []PolicyClause{},
						Method:   "get",
						Path:     "/example",
					},
				},
			},
			expected: []string{
				"path == \"/example\"\n{user} == {\"user1\",\"user2\"}",
			},
		},
		{
			name: "PathPolicies with UserPolicy and specialized method clause",
			path: "/example",
			policies: PathPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
				},
				SpecializedMethods: map[string]PathMethodPolicies{
					"get": {
						Policies: []PolicyClause{
							{
								RolePolicy: &RolePolicy{
									Operator: OperatorAnd,
									EnumeratedValue: EnumeratedValue{
										Value: []string{
											"role1", "role2",
										},
									},
								},
							},
						},
						Method: "get",
						Path:   "/example",
					},
				},
			},
			expected: []string{
				"path == \"/example\"\n{user} == {\"user1\",\"user2\"}\nmethod == \"GET\"\nroles == {\"role1\",\"role2\"}",
				"path == \"/example\"\n{user} == {\"user1\",\"user2\"}\nnot method in {\"GET\"}",
			},
		},
		{
			name: "PathPolicies with multiple clauses and specialized method clause",
			path: "/example",
			policies: PathPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorOr,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user3", "user4"},
							},
						},
					},
				},
				SpecializedMethods: map[string]PathMethodPolicies{
					"get": {
						Policies: []PolicyClause{
							{
								RolePolicy: &RolePolicy{
									Operator: OperatorAnd,
									EnumeratedValue: EnumeratedValue{
										Value: []string{
											"role1", "role2",
										},
									},
								},
							},
						},
						Method: "get",
						Path:   "/example",
					},
					"post": {
						Policies: []PolicyClause{
							{
								RolePolicy: &RolePolicy{
									Operator: OperatorAnd,
									EnumeratedValue: EnumeratedValue{
										Value: []string{
											"role3", "role4",
										},
									},
								},
							},
						},
						Method: "post",
						Path:   "/example",
					},
				},
			},
			expected: []string{
				"path == \"/example\"\n{user} == {\"user1\",\"user2\"}\nmethod == \"GET\"\nroles == {\"role1\",\"role2\"}",
				"path == \"/example\"\n{user} == {\"user1\",\"user2\"}\nmethod == \"POST\"\nroles == {\"role3\",\"role4\"}",
				"path == \"/example\"\nuser in {\"user3\",\"user4\"}\nmethod == \"GET\"\nroles == {\"role1\",\"role2\"}",
				"path == \"/example\"\nuser in {\"user3\",\"user4\"}\nmethod == \"POST\"\nroles == {\"role3\",\"role4\"}",
				`path == "/example"\n{user} == {"user1","user2"}\nnot method in {("GET","POST"|"POST","GET")}`,
				`path == "/example"\nuser in {"user3","user4"}\nnot method in {("GET","POST"|"POST","GET")}`,
			},
		},
		{
			name: "PathPolicies with no policies but specialized methods",
			path: "/example",
			policies: PathPolicies{
				Policies: []PolicyClause{},
				SpecializedMethods: map[string]PathMethodPolicies{
					"get": {
						Policies: []PolicyClause{
							{
								RolePolicy: &RolePolicy{
									Operator: OperatorAnd,
									EnumeratedValue: EnumeratedValue{
										Value: []string{
											"role1", "role2",
										},
									},
								},
							},
						},
						Method: "get",
					},
					"post": {
						Policies: []PolicyClause{
							{
								RolePolicy: &RolePolicy{
									Operator: OperatorAnd,
									EnumeratedValue: EnumeratedValue{
										Value: []string{
											"role3", "role4",
										},
									},
								},
							},
						},
						Method: "get",
					},
				},
			},
			expected: []string{
				`path == \"/example\"\nnot method in {("GET","POST"|"POST","GET")}`,
				"path == \"/example\"\nmethod == \"GET\"\nroles == {\"role1\",\"role2\"}",
				"path == \"/example\"\nmethod == \"POST\"\nroles == {\"role3\",\"role4\"}",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := generatePoliciesForPath(ServiceData{}, tc.path, tc.policies)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Sort the results to ensure a consistent comparison
			if len(got) != len(tc.expected) {
				t.Errorf("expected %d policies, got %d", len(tc.expected), len(got))
			}

			// Create maps to track matched policies
			matchedGot := make(map[int]bool)
			matchedExpected := make(map[int]bool)

			// Check each expected policy against all got policies
			for i, expected := range tc.expected {
				found := false
				for j, g := range got {
					if matched, err := regexp.MatchString(expected, g); matched && err == nil && !matchedGot[j] {
						matchedGot[j] = true
						matchedExpected[i] = true
						found = true
						break
					} else if err != nil {
						t.Errorf("error matching regex %s against %s: %v", expected, g, err)
					}
				}
				if !found {
					t.Errorf("missing expected policy: %s", expected)
				}
			}

			// Check for unexpected policies
			for j, g := range got {
				if !matchedGot[j] {
					t.Errorf("unexpected policy: %s", g)
				}
			}
		})
	}
}

func TestGenerateServicePolicies(t *testing.T) {
	tt := []struct {
		name     string
		policies GeneralPolicies
		expected string
	}{
		{
			name: "Only 1 general policy clause",
			policies: GeneralPolicies{
				Policies: []PolicyClause{
					{UserPolicy: &UserPolicy{
						Operator: OperatorAnd,
						EnumeratedValue: EnumeratedValue{
							Value: []string{"user1", "user2"},
						},
					},
						RolePolicy: &RolePolicy{
							Operator: OperatorOr,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"role1", "role2"},
							},
						},
					},
				},
			},
			expected: "default global_policy := false\n\n" +
				"global_policy if {\n" +
				"    {user} == {\"user1\",\"user2\"}\n" +
				"    roles & {\"role1\",\"role2\"} != set()\n" +
				"}\n\n" +
				"allow_request := global_policy\n",
		},
		{
			name: "Multiple general policy clauses",
			policies: GeneralPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorOr,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user3", "user4"},
							},
						},
					},
				},
			},
			expected: "default global_policy := false\n\n" +
				"global_policy if {\n" +
				"    {user} == {\"user1\",\"user2\"}\n" +
				"}\n\n" +
				"global_policy if {\n" +
				"    user in {\"user3\",\"user4\"}\n" +
				"}\n\n" +
				"allow_request := global_policy\n",
		},
		{
			name: "Multiple policies clause (some unimplemented yet)",
			policies: GeneralPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
					{
						StorageLocationPolicy: &StoragePolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"location1", "location2"},
							},
						},
					},
				},
			},
			expected: "default global_policy := false\n\n" +
				"global_policy if {\n" +
				"    {user} == {\"user1\",\"user2\"}\n" +
				"}\n\n" +
				"allow_request := global_policy\n",
		},
		{
			name: "General policies with specialized paths",
			policies: GeneralPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
				},
				SpecializedPaths: map[string]PathPolicies{
					"/example": PathPolicies{
						Policies: []PolicyClause{
							{
								RolePolicy: &RolePolicy{
									Operator: OperatorAnd,
									EnumeratedValue: EnumeratedValue{
										Value: []string{"role1", "role2"},
									},
								},
							},
						},
					},
				},
			},
			expected: "default global_policy := false\n\n" +
				"global_policy if {\n" +
				"    {user} == {\"user1\",\"user2\"}\n" +
				"}\n\n" +
				"default allow_request := false\n\n" +
				"allow_request if {\n" +
				"    global_policy\n" +
				"    path == \"/example\"\n" +
				"    roles == {\"role1\",\"role2\"}\n" +
				"}\n\n" +
				"allow_request if {\n" +
				"    global_policy\n" +
				"    not path in {\"/example\"}\n" +
				"}\n",
		},
		{
			name: "General policies with specialized methods",
			policies: GeneralPolicies{
				Policies: []PolicyClause{
					{
						UserPolicy: &UserPolicy{
							Operator: OperatorAnd,
							EnumeratedValue: EnumeratedValue{
								Value: []string{"user1", "user2"},
							},
						},
					},
				},
				SpecializedPaths: map[string]PathPolicies{
					"/example": PathPolicies{
						SpecializedMethods: map[string]PathMethodPolicies{
							"GET": PathMethodPolicies{
								Policies: []PolicyClause{
									{
										RolePolicy: &RolePolicy{
											Operator: OperatorAnd,
											EnumeratedValue: EnumeratedValue{
												Value: []string{"role1", "role2"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "default global_policy := false\n\n" +
				"global_policy if {\n" +
				"    {user} == {\"user1\",\"user2\"}\n" +
				"}\n\n" +
				"default allow_request := false\n\n" +
				"allow_request if {\n" +
				"    global_policy\n" +
				"    path == \"/example\"\n" +
				"    method == \"GET\"\n" +
				"    roles == {\"role1\",\"role2\"}\n" +
				"}\n\n" +
				"allow_request if {\n" +
				"    global_policy\n" +
				"    path == \"/example\"\n" +
				"    not method in {\"GET\"}\n" +
				"}\n\n" +
				"allow_request if {\n" +
				"    global_policy\n" +
				"    not path in {\"/example\"}\n" +
				"}\n",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			regopolicies, err := GenerateServiceRego(ServiceData{}, tc.policies)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if regopolicies != tc.expected {
				t.Errorf("expected\n%s, got\n%s", tc.expected, regopolicies)
			}
		})
	}
}

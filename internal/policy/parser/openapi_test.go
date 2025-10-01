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

package parser

import (
	"dspn-regogenerator/internal/policy"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

const testDataDir = "./../../../testdata/schemas"

func TestParseOpenAPISpec(t *testing.T) {
	tt := []struct {
		name     string
		fileName string
		expected func(*testing.T, *libopenapi.DocumentModel[v3.Document])
	}{
		{
			name:     "Valid OpenAPI Spec",
			fileName: filepath.Join(testDataDir, "httpbin-api.json"),
			expected: func(t *testing.T, documentModel *libopenapi.DocumentModel[v3.Document]) {
				if documentModel == nil {
					t.Fatal("Expected non-nil document model, got nil")
				}
				if documentModel.Model.Info.Title != "httpbin.org" {
					t.Errorf("Expected title 'httpbin.org', got '%s'", documentModel.Model.Info.Title)
				}
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.ReadFile(tc.fileName)
			if err != nil {
				t.Fatalf("Failed to read OpenAPI file: %v", err)
			}
			doc, err := parseOpenAPIDocument(strings.NewReader(string(file)))
			if err != nil {
				t.Fatalf("Failed to parse OpenAPI spec: %v", err)
			}
			tc.expected(t, doc)
		})
	}
}

func TestGetIdentityProvider(t *testing.T) {
	const testKeuycloakURL = "http://localhost/keycloak/realms/master/.well-known/openid-configuration"

	node := &yaml.Node{}
	node.SetString(testKeuycloakURL)

	securitySchemeExtension := orderedmap.New[string, *yaml.Node]()
	securitySchemeExtension.Set(IamExtensionTag, node)
	securitySchemes := orderedmap.New[string, *v3.SecurityScheme]()
	securitySchemes.Set("bearerAuth", &v3.SecurityScheme{
		Type:       "http",
		Scheme:     "bearer",
		Extensions: securitySchemeExtension,
	})
	docModel := &libopenapi.DocumentModel[v3.Document]{
		Model: v3.Document{
			Components: &v3.Components{
				SecuritySchemes: securitySchemes,
			},
		},
	}

	url, err := getIdentityProviderTag(docModel)
	if err != nil {
		t.Fatalf("Failed to get identity provider tag: %v", err)
	}
	if url != testKeuycloakURL {
		t.Errorf("Expected URL %s, got %s", testKeuycloakURL, url)
	}
}

func TestGetPolicies(t *testing.T) {
	tt := []struct {
		name            string
		generalPolicies []policy.PolicyClause
	}{
		{
			name: "OnlyUserGeneralPolicy",
			generalPolicies: []policy.PolicyClause{
				{
					UserPolicy: &policy.UserPolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"user1", "user2"},
						},
					},
				},
			},
		},
		{
			name: "OnlyRoleGeneralPolicy",
			generalPolicies: []policy.PolicyClause{
				{
					RolePolicy: &policy.RolePolicy{
						Operator: policy.OperatorOr,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"role1", "role2"},
						},
					},
				},
			},
		},
		{
			name: "OnlyStorageLocationGeneralPolicy",
			generalPolicies: []policy.PolicyClause{
				{
					StorageLocationPolicy: &policy.StoragePolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"Europe", "USA"},
						},
					},
				},
			},
		},
		{
			name: "OnlyCallGeneralPolicy",
			generalPolicies: []policy.PolicyClause{
				{
					CallPolicy: &policy.CallPolicy{
						Operator: policy.OperatorAnd,
						IntervalValue: policy.IntervalValue{
							Value: []policy.Interval{
								{
									Max:           1000,
									UnitOfMeasure: "call_per_year",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "OnlyTimelinessGeneralPolicy",
			generalPolicies: []policy.PolicyClause{
				{
					TimelinessPolicy: &policy.TimelinessPolicy{
						Operator: policy.OperatorAnd,
						IntervalValue: policy.IntervalValue{
							Value: []policy.Interval{
								{
									Max:           30,
									UnitOfMeasure: "days",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "UserAndRoleGeneralPolicy",
			generalPolicies: []policy.PolicyClause{
				{
					UserPolicy: &policy.UserPolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"user1", "user2"},
						},
					},
					RolePolicy: &policy.RolePolicy{
						Operator: policy.OperatorOr,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"role1", "role2"},
						},
					},
				},
			},
		},
		{
			name: "TwoClauseGeneralPolicy",
			generalPolicies: []policy.PolicyClause{
				{
					UserPolicy: &policy.UserPolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"user1", "user2"},
						},
					},
				},
				{
					UserPolicy: &policy.UserPolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"user3", "user4"},
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Create a sample OpenAPI document with the provided policies
			policies := &policy.GeneralPolicies{
				Policies: tc.generalPolicies,
			}
			tagContent := XTeadalPolicies{
				Policies:    policies.Policies,
				Description: "todo placeholder",
			}
			generalPolicyNode := &yaml.Node{}
			generalPolicyNode.Encode(tagContent)

			extensions := orderedmap.New[string, *yaml.Node]()
			extensions.Set(PolicyExtensionTag, generalPolicyNode)

			docModel := &libopenapi.DocumentModel[v3.Document]{
				Model: v3.Document{
					Paths: &v3.Paths{
						Extensions: extensions,
					},
				},
			}

			// Get the policies
			actualPolicies, err := getPolicies(docModel)
			if err != nil {
				t.Fatalf("Failed to get general policies: %v", err)
			}

			if !reflect.DeepEqual(actualPolicies.Policies, tc.generalPolicies) {
				t.Errorf("Expected policies %v, got %v", tc.generalPolicies, actualPolicies.Policies)
			}
			if len(actualPolicies.Policies) != len(tc.generalPolicies) {
				t.Errorf("Expected %d policies, got %d", len(tc.generalPolicies), len(actualPolicies.Policies))
			}
		})
	}
}

// GoldenData represents the structure to be saved in the golden file
type GoldenData struct {
	URL      string                  `json:"url"`
	Policies *policy.GeneralPolicies `json:"policies"`
}

func TestParseHTTPBinAPI(t *testing.T) {
	// Use the TestUpgrade variable defined at the package level

	httpbinSpecFile := filepath.Join(testDataDir, "httpbin-api.json")
	file, err := os.Open(httpbinSpecFile)
	if err != nil {
		t.Fatalf("Failed to read OpenAPI file: %v", err)
	}
	defer file.Close()

	// Parse the OpenAPI document
	docModel, err := parseOpenAPIDocument(file)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI file: %v", err)
	}
	if docModel == nil {
		t.Fatalf("Expected non-nil result, got nil")
	}

	// Get the identity provider URL
	url, err := getIdentityProviderTag(docModel)
	if err != nil && url == "" {
		t.Fatalf("Failed to get identity provider tag: %v", err)
	}

	// Get the policies
	structuredPolicies, err := getPolicies(docModel)
	if err != nil {
		t.Fatalf("Failed to get general policies: %v", err)
	}

	// Prepare the golden data
	goldenData := GoldenData{
		URL:      url,
		Policies: structuredPolicies,
	}

	// Path for the golden file
	goldenFilePath := filepath.Join(testDataDir, "golden", "httpbin-api.golden")

	if *updateGolden {
		// Generate the golden file
		t.Logf("Updating golden file %s", goldenFilePath)
		goldenJSON, err := json.MarshalIndent(goldenData, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal golden data: %v", err)
		}
		err = os.MkdirAll(filepath.Dir(goldenFilePath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory for golden file: %v", err)
		}

		err = os.WriteFile(goldenFilePath, goldenJSON, 0644)
		if err != nil {
			t.Fatalf("Failed to write golden file: %v", err)
		}
		t.Logf("Golden file updated successfully. Please review %s", goldenFilePath)
	} else {
		// Read the golden file and compare with current results
		goldenFile, err := os.ReadFile(goldenFilePath)
		if err != nil {
			t.Fatalf("Failed to read golden file: %v, run with -upgrade flag to generate it", err)
		}

		var expectedGoldenData GoldenData
		err = json.Unmarshal(goldenFile, &expectedGoldenData)
		if err != nil {
			t.Fatalf("Failed to unmarshal golden data: %v", err)
		}

		// Verify correct reading of x-teadal-IAM-provider tag
		if url != expectedGoldenData.URL {
			t.Errorf("Expected URL %s, got %s", expectedGoldenData.URL, url)
		}

		// Verify correct reading of x-teadal-policies tags
		comparePoliciesArray(t, expectedGoldenData.Policies.Policies, structuredPolicies.Policies, "general policies")

		// Compare specialized paths
		if len(expectedGoldenData.Policies.SpecializedPaths) != len(structuredPolicies.SpecializedPaths) {
			t.Errorf("Expected %d specialized paths, got %d",
				len(expectedGoldenData.Policies.SpecializedPaths),
				len(structuredPolicies.SpecializedPaths))
		} else {
			for path, expectedPathPolicies := range expectedGoldenData.Policies.SpecializedPaths {
				actualPathPolicies, exists := structuredPolicies.SpecializedPaths[path]
				if !exists {
					t.Errorf("Expected path %s not found in actual policies", path)
					continue
				}

				comparePoliciesArray(t, expectedPathPolicies.Policies, actualPathPolicies.Policies,
					fmt.Sprintf("path policies for %s", path))

				// Compare specialized methods
				if len(expectedPathPolicies.SpecializedMethods) != len(actualPathPolicies.SpecializedMethods) {
					t.Errorf("For path %s: expected %d specialized methods, got %d",
						path, len(expectedPathPolicies.SpecializedMethods), len(actualPathPolicies.SpecializedMethods))
				} else {
					for method, expectedMethodPolicies := range expectedPathPolicies.SpecializedMethods {
						actualMethodPolicies, exists := actualPathPolicies.SpecializedMethods[method]
						if !exists {
							t.Errorf("Expected method %s for path %s not found in actual policies", method, path)
							continue
						}

						comparePoliciesArray(t, expectedMethodPolicies.Policies, actualMethodPolicies.Policies,
							fmt.Sprintf("method policies for %s %s", method, path))
					}
				}
			}
		}
	}
}

// comparePolicy is a generic helper function that compares two policy objects
// and logs an error if they're not equal
func comparePolicy(t *testing.T, expected, actual interface{}, policyType string) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %s %v, got %v", policyType, expected, actual)
	}
}

// comparePoliciesArray compares two arrays of policy clauses and reports detailed differences
func comparePoliciesArray(t *testing.T, expected, actual []policy.PolicyClause, arrayName string) {
	if reflect.DeepEqual(expected, actual) {
		return
	}

	if len(expected) != len(actual) {
		t.Errorf("Expected %d %s, got %d", len(expected), arrayName, len(actual))
		return
	}

	for i, expectedPolicy := range expected {
		if i >= len(actual) {
			t.Errorf("Missing policy at index %d in %s", i, arrayName)
			continue
		}

		actualPolicy := actual[i]
		if !reflect.DeepEqual(expectedPolicy, actualPolicy) {
			t.Errorf("Policy mismatch at index %d in %s", i, arrayName)

			// Compare each policy field individually for more detailed error messages
			comparePolicy(t, expectedPolicy.CallPolicy, actualPolicy.CallPolicy, "CallPolicy")
			comparePolicy(t, expectedPolicy.StorageLocationPolicy, actualPolicy.StorageLocationPolicy, "StorageLocationPolicy")
			comparePolicy(t, expectedPolicy.UserPolicy, actualPolicy.UserPolicy, "UserPolicy")
			comparePolicy(t, expectedPolicy.RolePolicy, actualPolicy.RolePolicy, "RolePolicy")
			comparePolicy(t, expectedPolicy.TimelinessPolicy, actualPolicy.TimelinessPolicy, "TimelinessPolicy")
		}
	}
}

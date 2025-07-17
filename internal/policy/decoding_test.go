package policy

import (
	"encoding/json"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDecodeJsonPolicy(t *testing.T) {
	tt := []struct {
		name     string
		input    string
		expected any
	}{
		{
			name:  "Valid UserPolicy",
			input: `{"operator": "AND", "value": ["user1", "user2"]}`,
			expected: UserPolicy{
				Operator: OperatorAnd,
				EnumeratedValue: EnumeratedValue{
					Value: []string{"user1", "user2"},
				},
			},
		},
		{
			name:  "Valid RolePolicy",
			input: `{"operator": "OR", "value": ["admin", "manager"]}`,
			expected: RolePolicy{
				Operator: OperatorOr,
				EnumeratedValue: EnumeratedValue{
					Value: []string{"admin", "manager"},
				},
			},
		},
		{
			name:  "Valid CallPolicy",
			input: `{"value": [{"min": 10, "max": 20, "unit_of_measure": "seconds"}], "operator": "AND"}`,
			expected: CallPolicy{
				Operator: OperatorAnd,
				IntervalValue: IntervalValue{
					Value: []Interval{
						{Min: 10, Max: 20, UnitOfMeasure: "seconds"},
					},
				},
			},
		},
		{
			name:  "Valid TimelinessPolicy",
			input: `{"operator": "AND", "value": [{"min": 0, "max": 3600, "unit_of_measure": "seconds"}]}`,
			expected: TimelinessPolicy{
				Operator: OperatorAnd,
				IntervalValue: IntervalValue{
					Value: []Interval{
						{Min: 0, Max: 3600, UnitOfMeasure: "seconds"},
					},
				},
			},
		},
		{
			name:  "Valid StorageLocationPolicy",
			input: `{"operator": "AND", "value": ["us-west-1"]}`, // Fixed: string array instead of plain string
			expected: StoragePolicy{
				Operator: OperatorAnd,
				EnumeratedValue: EnumeratedValue{
					Value: []string{"us-west-1"},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var result any
			switch expected := tc.expected.(type) {
			case UserPolicy:
				result = &UserPolicy{}
			case RolePolicy:
				result = &RolePolicy{}
			case CallPolicy:
				result = &CallPolicy{}
			case TimelinessPolicy:
				result = &TimelinessPolicy{}
			case StoragePolicy:
				result = &StoragePolicy{}
			default:
				t.Fatalf("Unexpected expected type: %T", expected)
			}

			err := json.Unmarshal([]byte(tc.input), result)
			if err != nil {
				t.Fatalf("DecodePolicy failed: %v", err)
			}

			// Get the actual value that the pointer refers to
			resultValue := reflect.ValueOf(result).Elem().Interface()

			// Compare the value with the expected value
			if !reflect.DeepEqual(resultValue, tc.expected) {
				t.Errorf("DecodePolicy expected %v, got %v", tc.expected, resultValue)
			}
		})
	}
}

func TestDecodeJsonClause(t *testing.T) {
	tt := []struct {
		name     string
		input    string
		expected PolicyClause
	}{
		{
			name:  "Valid PolicyClause with UserPolicy",
			input: `{"user": {"operator": "AND", "value": ["user1", "user2"]}}`,
			expected: PolicyClause{
				UserPolicy: &UserPolicy{
					Operator: OperatorAnd,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"user1", "user2"},
					},
				},
			},
		},
		{
			name:  "Httpbin general clauses",
			input: `{"storage_location":{"operator":"OR","value":["Europe","USA"]},"call":{"value":[{"max":50000,"unit_of_measure":"call_per_year"}]}}`,
			expected: PolicyClause{
				StorageLocationPolicy: &StoragePolicy{
					Operator: OperatorOr,
					EnumeratedValue: EnumeratedValue{
						Value: []string{"Europe", "USA"},
					},
				},
				CallPolicy: &CallPolicy{
					IntervalValue: IntervalValue{
						Value: []Interval{
							{Max: 50000, UnitOfMeasure: "call_per_year"},
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var result PolicyClause
			err := json.Unmarshal([]byte(tc.input), &result)
			if err != nil {
				t.Fatalf("DecodeClause failed: %v", err)
			}

			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("DecodeClause expected %v, got %v", tc.expected, result)
				if !reflect.DeepEqual(result.CallPolicy, tc.expected.CallPolicy) {
					t.Errorf("DecodeClause expected CallPolicy %v, got %v", tc.expected.CallPolicy, result.CallPolicy)
				}
				if !reflect.DeepEqual(result.StorageLocationPolicy, tc.expected.StorageLocationPolicy) {
					t.Errorf("DecodeClause expected CallPolicy %v, got %v", tc.expected.CallPolicy, result.CallPolicy)
				}
				if !reflect.DeepEqual(result.UserPolicy, tc.expected.UserPolicy) {
					t.Errorf("DecodeClause expected UserPolicy %v, got %v", tc.expected.UserPolicy, result.UserPolicy)
				}
			}
		})
	}
}

func TestDecodeYamlPolicy(t *testing.T) {
	tt := []struct {
		name     string
		input    string
		expected any
	}{
		{
			name:  "Valid UserPolicy",
			input: "operator: AND\nvalue:\n  - user1\n  - user2",
			expected: UserPolicy{
				Operator: OperatorAnd,
				EnumeratedValue: EnumeratedValue{
					Value: []string{"user1", "user2"},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var result any
			switch expected := tc.expected.(type) {
			case UserPolicy:
				result = &UserPolicy{}
			case RolePolicy:
				result = &RolePolicy{}
			case CallPolicy:
				result = &CallPolicy{}
			case TimelinessPolicy:
				result = &TimelinessPolicy{}
			case StoragePolicy:
				result = &StoragePolicy{}
			default:
				t.Fatalf("Unexpected expected type: %T", expected)
			}

			err := yaml.Unmarshal([]byte(tc.input), result)
			if err != nil {
				t.Fatalf("DecodeYamlPolicy failed: %v", err)
			}

			// Get the actual value that the pointer refers to
			resultValue := reflect.ValueOf(result).Elem().Interface()

			// Compare the value with the expected value
			if !reflect.DeepEqual(resultValue, tc.expected) {
				t.Errorf("DecodeYamlPolicy expected %v, got %v", tc.expected, resultValue)
			}
		})
	}
}

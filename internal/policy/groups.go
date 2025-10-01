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

package policy

// Represent a policy clauses, which can contains at most one of each type of policy
type PolicyClause struct {
	UserPolicy            *UserPolicy       `yaml:"user" json:"user,omitempty"`
	RolePolicy            *RolePolicy       `yaml:"roles" json:"roles,omitempty"`
	StorageLocationPolicy *StoragePolicy    `yaml:"storage_location" json:"storage_location,omitempty"`
	CallPolicy            *CallPolicy       `yaml:"call" json:"call,omitempty"`
	TimelinessPolicy      *TimelinessPolicy `yaml:"timeliness" json:"timeliness,omitempty"`
}

// GeneralPolicies represents a collection of policy clauses that should applied to all paths and endpoints
// It applies the policies to all endpoints, but it can be extended (AND) with specialized policies for specific paths
type GeneralPolicies struct {
	Policies         []PolicyClause
	SpecializedPaths map[string]PathPolicies
}

func NewGeneralPolicies() *GeneralPolicies {
	return &GeneralPolicies{
		Policies:         make([]PolicyClause, 0),
		SpecializedPaths: make(map[string]PathPolicies),
	}
}

// PathPolicies represents a collection of policy clauses that should applied to a specific path.
type PathPolicies struct {
	Policies           []PolicyClause
	Path               string
	SpecializedMethods map[string]PathMethodPolicies
}

type PathMethodPolicies struct {
	Policies []PolicyClause
	Path     string
	Method   string
}

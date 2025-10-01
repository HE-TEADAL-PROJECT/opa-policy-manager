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

package bundle

import (
	"context"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/policy/parser"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/compose"
)

const bundlePath = "./bundle.tar.gz"

func createTestBundle(t testing.TB) {
	t.Helper()
	service := bundle.NewService("httpbin", &parser.ServiceSpec{
		Policies:         parser.StructuredPolicies{},
		IdentityProvider: "http://127.0.0.1:5555",
	})
	b, err := bundle.New(service)
	if err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}
	repo := bundle.FSRepository{}
	if err := repo.Save(bundlePath, b); err != nil {
		t.Fatalf("failed to save bundle: %v", err)
	}
}

func createStack(t testing.TB) *compose.DockerCompose {
	t.Helper()
	stack, err := compose.NewDockerComposeWith(
		compose.StackIdentifier("test-e2e-bundle"),
		compose.WithStackFiles("./docker-compose.yaml"),
	)
	if err != nil {
		t.Fatalf("Failed to create Docker Compose stack: %v", err)
	}

	if err := stack.Up(context.TODO(), compose.Wait(true)); err != nil {
		t.Fatalf("Failed to start Docker Compose stack: %v", err)
	}
	t.Cleanup(func() {
		if err := stack.Down(context.TODO(), compose.RemoveOrphans(true)); err != nil {
			t.Fatalf("Failed to stop Docker Compose stack: %v", err)
		}
	})
	return stack
}

func TestBundleAuthorization(t *testing.T) {
	createTestBundle(t)
	createStack(t)
}

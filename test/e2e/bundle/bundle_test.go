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

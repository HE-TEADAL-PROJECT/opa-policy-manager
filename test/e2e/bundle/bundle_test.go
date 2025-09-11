package bundle

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/compose"
)

func TestBundleAuthorization(t *testing.T) {
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
}

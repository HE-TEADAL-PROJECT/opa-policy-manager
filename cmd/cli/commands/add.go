package commands

import (
	"dspn-regogenerator/internal/usecases"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	openAPISpec string
)

func loadSpecFile(specFile string) ([]byte, error) {
	// Load the OpenAPI spec file
	specData, err := os.ReadFile(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec file: %v", err)
	}
	return specData, nil
}

var AddCmd = &cobra.Command{
	Use:   "add [--spec <path/to/openapi/spec>] <service name>",
	Short: "Add policies related to a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		if openAPISpec == "" {
			cmd.Help()
			return
		}

		// Load the OpenAPI spec file
		specData, err := loadSpecFile(openAPISpec)
		if err != nil {
			slog.Error("Error loading OpenAPI spec file", "error", err)
			return
		}

		err = usecases.AddService(serviceName, specData)
		if err != nil {
			slog.Error("Error adding service", "serviceName", serviceName, "error", err)
			return
		}
	},
}

func init() {
	AddCmd.Flags().StringVar(&openAPISpec, "spec", "", "OpenAPI spec filename (required)")
	AddCmd.MarkFlagRequired("spec")
}

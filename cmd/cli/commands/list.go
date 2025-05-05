package commands

import (
	"dspn-regogenerator/internal/usecases"
	"fmt"

	"log/slog"

	"github.com/spf13/cobra"
)

var ReservedServiceName = []string{}

var (
	verbose bool
)

var ListCmd = &cobra.Command{
	Use:   "list [-v]",
	Short: "List all available services",
	Long:  `List all available services in the bundle. Use the verbose flag for detailed list of rego files.`,
	Run: func(cmd *cobra.Command, args []string) {
		bundleStructure, err := usecases.GetBundleStructure()
		if err != nil {
			slog.Error("Error getting bundle structure", "error", err)
			return
		}
		slog.Info(fmt.Sprintf("Services: %v", bundleStructure.Services))
		if verbose {
			slog.Info(fmt.Sprintf("Files: %v", bundleStructure.Files))
		}
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

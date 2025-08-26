package commands

import (
	"dspn-regogenerator/internal/usecases"
	"log/slog"

	"github.com/spf13/cobra"
)

var ()

var DeleteCmd = &cobra.Command{
	Use:   "delete <service name>",
	Short: "Delete all policies related to a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		if serviceName == "" {
			cmd.Help()
			return
		}

		err := usecases.DeleteService(serviceName)
		if err != nil {
			slog.Error("Error deleting service", "serviceName", serviceName, "error", err)
		}
	},
}

package main

import (
	"dspn-regogenerator/cmd/cli/commands"
	"dspn-regogenerator/internal/config"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "dspn-regogenerator"}
	rootCmd.AddCommand(commands.AddCmd, commands.ListCmd, commands.DeleteCmd, commands.TestCmd, commands.GetCmd)

	slog.SetDefault(slog.New(NewCliHandler(os.Stderr, config.LogLevel)))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

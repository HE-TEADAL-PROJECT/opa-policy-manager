package main

import (
	"dspn-regogenerator/cmd/cli/commands"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "dspn-regogenerator"}
	rootCmd.AddCommand(commands.AddCmd, commands.ListCmd, commands.DeleteCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

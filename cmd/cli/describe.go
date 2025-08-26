package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe [dsn]",
	Short: "Describe opa bundle referenced by dsn",
	Long:  "Describe opa bundle referenced by dsn.\n" + defaultDsnUsage,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, path, err := getRepositoryAndPath(args)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		bundle, err := repo.Get(path)
		if err != nil {
			fmt.Printf("Error getting bundle %q: %v\n", path, err)
			return
		}
		fmt.Printf("Bundle %q:\n", path)
		fmt.Printf("  Services: %v\n", bundle.Describe())
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

package commands

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"path/filepath"

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
		// Check if the bundle exists on minio, otherwise print an error message
		bundleExists, err := bundle.CheckBundleFileExists(config.LatestBundleName)
		if err != nil {
			cmd.PrintErrf("Error checking if bundle exists: %v\n", err)
			return
		}
		if !bundleExists {
			cmd.PrintErrf("Bundle %s does not exist in bucket %s\n", config.LatestBundleName, config.MinioBucket)
			return
		}

		loadedBundle, err := bundle.LoadBundleFromMinio(config.LatestBundleName)
		if err != nil {
			cmd.PrintErrf("Error loading bundle from Minio: %v\n", err)
			return
		}
		files := bundle.ListBundleFiles(loadedBundle)
		dirs := map[string][]string{}
		for _, dir := range files {
			dirs[filepath.Dir(dir)] = append(dirs[filepath.Dir(dir)], filepath.Base(dir))
		}
		cmd.Println("Services available:", len(dirs))
		for k, v := range dirs {
			cmd.Println("-", k)
			if verbose {
				for _, file := range v {
					cmd.Println("   -", file)
				}
			}
		}
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

package main

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"fmt"

	"github.com/spf13/cobra"
)

type SourceProtocol int

const (
	File SourceProtocol = iota
	Minio
)

const (
	defaultLatestTag = "latest"
)

func parseDsn(dsn string) (SourceProtocol, string) {
	if dsn == "" {
		return Minio, defaultLatestTag
	}
	if len(dsn) > 7 && dsn[:7] == "minio://" {
		return Minio, dsn[7:]
	}
	if len(dsn) > 7 && dsn[:7] == "file://" {
		return File, dsn[7:]
	}
	return File, dsn
}

var describeCmd = &cobra.Command{
	Use:   "describe [dsn]",
	Short: "Describe opa bundle referenced by dsn",
	Long: `Describe opa bundle referenced by dsn.
If dsn is not provided, the latest bundle will be described.
The dsn format is:
  minio://<path> - to reference a specific bundle by its <path> inside the bucket
  minio://latest - to reference the latest bundle
  file://<path> - to reference a local bundle file
  <path> - to reference a local bundle file (same as file://<path>)

To specify further minio config using environment variables:
  MINIO_SERVER - MinIO server address (default: localhost:9000)
  MINIO_ACCESS_KEY - MinIO access key (default: admin)
  MINIO_SECRET_KEY - MinIO secret key (default: adminadmin)
  BUCKET_NAME - MinIO bucket name (default: opa-policy-bundles)
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dsn := ""
		if len(args) > 0 {
			dsn = args[0]
		}
		proto, path := parseDsn(dsn)
		var repo bundle.Repository
		switch proto {
		case Minio:
			var err error
			repo, err = bundle.NewMinioRepositoryFromConfig()
			if err != nil {
				fmt.Printf("Error creating minio repository: %v\n", err)
				return
			}
			if path == defaultLatestTag {
				path = config.LatestBundleName
			}
		case File:
			repo = bundle.FSRepository{}
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

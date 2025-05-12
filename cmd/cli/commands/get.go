package commands

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"log/slog"

	"github.com/spf13/cobra"
)

var (
	outputDir string
)

func init() {
	GetCmd.Flags().StringVar(&outputDir, "output", "./output", "Output path for the latest bundle")
}

var GetCmd = &cobra.Command{
	Use:   "get [--output <path>]",
	Short: "Get latest bundle",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the latest bundle
		minioRepo, err := bundle.NewMinioRepositoryFromConfig()
		if err != nil {
			slog.Error("Error creating minio repository", "error", err)
			return
		}
		b, err := minioRepo.Read(config.LatestBundleName)
		if err != nil {
			slog.Error("Error reading bundle from Minio", "error", err)
			return
		}
		fileRepo := bundle.NewFileSystemRepository(outputDir)
		if err := fileRepo.Write(config.LatestBundleName, *b); err != nil {
			slog.Error("Error writing bundle to file system", "error", err)
			return
		}
		slog.Info("Bundle written to file system successfully", "path", outputDir)
	},
}

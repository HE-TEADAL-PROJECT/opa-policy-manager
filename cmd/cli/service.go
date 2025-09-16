package main

import (
	bundlelib "dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/policy/parser"
	"os"

	"github.com/spf13/cobra"
)

var newBundle bool

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services in the OPA bundle",
	Args:  cobra.MaximumNArgs(1),
}

func init() {
	var addServiceCmd = &cobra.Command{
		Use:   "add service-name spec [dsn]",
		Short: "Add or update a service in the OPA bundle",
		Long:  "Add or update a service in the OPA bundle.\n" + defaultDsnUsage,
		Args:  cobra.RangeArgs(2, 3),
		Run: func(cmd *cobra.Command, args []string) {
			repo, path, err := getRepositoryAndPath(args[2:])
			if err != nil {
				cmd.PrintErrf("%v\n", err)
				os.Exit(1)
			}
			serviceName := args[0]
			specPath := args[1]

			specFile, err := os.Open(specPath)
			if err != nil {
				cmd.PrintErrf("Error opening spec file %q: %v\n", specPath, err)
				os.Exit(1)
			}
			defer specFile.Close()

			spec, err := parser.ParseServiceSpec(specFile)
			if err != nil {
				cmd.PrintErrf("Error parsing spec file %q: %v\n", specPath, err)
				os.Exit(1)
			}

			var bundle *bundlelib.Bundle

			if newBundle {
				bundle, err = bundlelib.New(bundlelib.NewService(serviceName, spec))
				if err != nil {
					cmd.PrintErrf("Error creating new bundle with service %q: %v\n", serviceName, err)
					os.Exit(1)
				}
			} else {
				bundle, err = repo.Get(path)
				if err != nil {
					cmd.PrintErrf("Error getting bundle %q: %v\n", path, err)
					os.Exit(1)
				}
				err = bundle.AddService(*bundlelib.NewService(serviceName, spec))
				if err != nil {
					cmd.PrintErrf("Error adding service %q: %v\n", serviceName, err)
					os.Exit(1)
				}
			}

			err = repo.Save(path, bundle)
			if err != nil {
				cmd.PrintErrf("Error saving bundle %q: %v\n", path, err)
				os.Exit(1)
			}
			cmd.Printf("Service %q added/updated in bundle %q\n", serviceName, path)
		},
	}
	addServiceCmd.Flags().BoolVar(&newBundle, "new", false, "Create a new bundle")

	var removeServiceCmd = &cobra.Command{
		Use:   "remove service-name [dsn]",
		Short: "Remove a service from the OPA bundle",
		Long:  "Remove a service from the OPA bundle.\n" + defaultDsnUsage,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, path, err := getRepositoryAndPath(args[1:])
			if err != nil {
				return err
			}
			serviceName := args[0]
			bundle, err := repo.Get(path)
			if err != nil {
				return err
			}
			err = bundle.RemoveService(serviceName)
			if err != nil {
				return err
			}
			err = repo.Save(path, bundle)
			if err != nil {
				return err
			}
			cmd.Printf("Service %q removed from bundle %q\n", serviceName, path)
			return nil
		},
	}

	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(addServiceCmd)
	serviceCmd.AddCommand(removeServiceCmd)
}

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/octarect/gpkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfg     gpkg.Config
	rootCmd = &cobra.Command{
		Use:   "gpkg",
		Short: "A general package manager",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := "./config.toml"
			viper.SetConfigFile(cfgPath)
			if err := viper.ReadInConfig(); err != nil {
				return err
			}
			if err := viper.Unmarshal(&cfg); err != nil {
				return err
			}
			if cfg.CachePath == "" {
				usrCacheDir, err := os.UserCacheDir()
				if err != nil {
					return err
				}
				cfg.CachePath = filepath.Join(usrCacheDir, "gpkg")
			}

			for _, spec := range cfg.Specs {
				if err := spec.Init(); err != nil {
					return err
				}
			}

			return nil
		},
	}
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version of gpkg",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(gpkg.Version)
		},
	}
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Install or update packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commandUpdate()
		},
	}
	sourceCmd = &cobra.Command{
		Use:   "source",
		Short: "Generate script to Source packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commandSource()
		},
	}
)

func main() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(sourceCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
	return
}

func commandUpdate() error {
	es := gpkg.Reconcile(cfg.GetPackagesPath(), cfg.Specs)
	if len(es) > 0 {
		printReconcileErrors(es)
		return errors.New("failed to update packages")
	}

	return nil
}

var errorFormat = `
Error updating %s:
  => %s
`

func printReconcileErrors(es []*gpkg.ReconcileError) {
	for _, e := range es {
		fmt.Fprintf(os.Stderr, strings.TrimSpace(errorFormat), e.Spec.Name, e.Err)
		fmt.Println()
	}
}

func commandSource() error {
	paths := make([]string, len(cfg.Specs))
	for i, spec := range cfg.Specs {
		paths[i] = filepath.Join(cfg.GetPackagesPath(), spec.GetDirName())
	}
	fmt.Printf(`export PATH="$PATH:%s"`, strings.Join(paths, ":"))
	return nil
}

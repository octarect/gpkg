package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/octarect/gpkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfg     gpkg.Config
	rootCmd = &cobra.Command{
		Use:   "gpkg",
		Short: "A package manager for CLI environment",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if cfgPath == "" {
				if cfgPath, err = defaultConfigPath(); err != nil {
					return err
				}
			}

			if cmd.Use == "init" {
				return nil
			}

			viper.SetConfigFile(cfgPath)
			if err := viper.ReadInConfig(); err != nil {
				return err
			}
			if err := viper.Unmarshal(&cfg, gpkg.DecoderConfigOption); err != nil {
				return err
			}

			if cfg.CachePath == "" {
				if cfg.CachePath, err = defaultCachePath(); err != nil {
					return err
				}
			}
			return nil
		},
	}
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version and exit",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(gpkg.Version)
		},
	}
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize a new config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Initializing %s\n", cfgPath)
			_, err := os.Stat(cfgPath)
			if err == nil && !force {
				yes, err := prompt(fmt.Sprintf("%s already exists. Overwrite it?", cfgPath))
				if err != nil {
					return err
				}
				if !yes {
					return fmt.Errorf("Aborted by user")
				}
			}
			err = gpkg.CreateConfigFile(cfgPath)
			if err != nil {
				return fmt.Errorf("Failed to create a config file with unexpected error: %v", err)
			}
			return nil
		},
	}
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Install or update packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commandUpdate()
		},
	}
	loadCmd = &cobra.Command{
		Use:   "load",
		Short: "Generate script to load packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commandLoad()
		},
	}
	cfgPath string
	force   bool
)

func main() {
	initCmd.Flags().BoolVar(&force, "force", false, "If true, all operations are executed without confirmation.")
	rootCmd.AddCommand(initCmd)

	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(loadCmd)
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "config file (default is $XDG_CONFIG_HOME/gpkg/config.yml)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	return
}

var errorFormat = `
Error updating %s:
  => %s
`

func commandUpdate() error {
	for _, spec := range cfg.Specs {
		ch := make(chan *gpkg.Event)
		bar := newProgressBar(spec.DisplayName())
		go func() {
			for range time.Tick(500 * time.Millisecond) {
				select {
				case ev, ok := <-ch:
					if !ok {
						bar.Bar.Finish()
						return
					}
					switch ev.Type {
					case gpkg.EventStarted:
						fmt.Printf("Installing %s\n", ev.Spec.DisplayName())
						bar.Bar.Start()
					case gpkg.EventDownloadStarted:
						d := ev.Data.(gpkg.EventDataDownload)
						bar.SetTotal(d.ContentLength)
					case gpkg.EventDownloadCompleted:
						bar.Bar.Finish()
					case gpkg.EventPickStarted:
						fmt.Printf("[INFO] Picking %s\n", ev.Spec.Common().Pick)
					}
				}
			}
		}()

		err := gpkg.ReconcilePackage(cfg.GetPackagesPath(), spec, ch, bar)
		close(ch)
		if err != nil {
			return fmt.Errorf(strings.TrimSpace(errorFormat), spec.DisplayName(), err)
		}
		fmt.Println()
	}

	return nil
}

func commandLoad() error {
	paths := make([]string, len(cfg.Specs))
	for i, spec := range cfg.Specs {
		paths[i] = filepath.Join(cfg.GetPackagesPath(), spec.DirName())
	}
	fmt.Printf(`export PATH="$PATH:%s"`, strings.Join(paths, ":"))
	return nil
}

func defaultConfigPath() (string, error) {
	usrCfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(usrCfgDir, "gpkg/config.toml"), nil
}

func defaultCachePath() (string, error) {
	usrCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(usrCacheDir, "gpkg"), nil
}

func prompt(message string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/N]", message)
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("Aborting. %v", err)
		}

		input = strings.ToLower(strings.TrimSpace(input))
		return input == "y", nil
	}
}

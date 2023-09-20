package main

import (
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
		Short: "A general package manager",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := "./config.toml"
			viper.SetConfigFile(cfgPath)
			if err := viper.ReadInConfig(); err != nil {
				return err
			}
			if err := viper.Unmarshal(&cfg, gpkg.DecoderConfigOption); err != nil {
				return err
			}
			if cfg.CachePath == "" {
				usrCacheDir, err := os.UserCacheDir()
				if err != nil {
					return err
				}
				cfg.CachePath = filepath.Join(usrCacheDir, "gpkg")
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
	loadCmd = &cobra.Command{
		Use:   "load",
		Short: "Generate script to load packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commandLoad()
		},
	}
)

func main() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(loadCmd)
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
				case ev, ok := <- ch:
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
					case gpkg.EventCompleted:
						bar.Bar.Finish()
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

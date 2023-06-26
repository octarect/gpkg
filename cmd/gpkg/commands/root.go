package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/octarect/gpkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	RootCmd = &cobra.Command{
		Use:               "gpkg",
		Short:             "A general package manager",
		PersistentPreRunE: globalPreRun,
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

	cfgPath string
)

func init() {
	RootCmd.AddCommand(updateCmd)
	RootCmd.AddCommand(sourceCmd)
	RootCmd.AddCommand(versionCmd)
	RootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "config file (default is $XDG_CONFIG_HOME/gpkg/config.yml)")
}

func globalPreRun(cmd *cobra.Command, args []string) error {
	if err := initConfig(); err != nil {
		return err
	}

	return nil
}

type Config struct {
	Packages []*gpkg.PackageSpec `json:"packages"`
}

var cfg Config

func initConfig() error {
	if cfgPath == "" {
		usrCfgDir, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		cfgPath = filepath.Join(usrCfgDir, "gpkg/config.toml")
	}

	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	return nil
}

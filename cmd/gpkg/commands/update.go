package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/octarect/gpkg"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Install and update packages",
	RunE:  runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	usrCacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("Failed to get a cache path: %s", err)
	}
	cachePath := filepath.Join(usrCacheDir, "gpkg")

	m, err := gpkg.NewManager(cachePath, cfg.Packages)
	if err != nil {
		return fmt.Errorf("Failed to initialize manager: %s", err)
	}

	if err := m.Sync(); err != nil {
		fmt.Println(err)
	}

	return nil
}

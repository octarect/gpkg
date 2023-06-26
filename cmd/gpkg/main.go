package main

import (
	"fmt"

	"os"

	"github.com/octarect/gpkg/cmd/gpkg/commands"
)

func main() {
	if err := commands.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return
}

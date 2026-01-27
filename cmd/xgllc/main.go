package main

import (
	"os"

	"github.com/cwbudde/gll-tools/cmd/xgllc/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

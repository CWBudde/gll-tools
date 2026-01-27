package main

import (
	"os"

	"github.com/cwbudde/gll-tools/cmd/gllinfo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

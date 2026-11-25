package main

import (
	"os"

	"github.com/kava-labs/kvtool/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

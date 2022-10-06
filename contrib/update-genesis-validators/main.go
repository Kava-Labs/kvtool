package main

import (
	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kvtool/contrib/update-genesis-validators/cmd"
)

func main() {
	app.SetSDKConfig()
	cmd.Execute()
}

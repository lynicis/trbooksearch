package main

import "github.com/lynicis/trbooksearch/cmd"

var version = "1.5.0"

func main() {
	cmd.SetVersionInfo(version)
	cmd.Execute()
}

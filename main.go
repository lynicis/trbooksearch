package main

import "github.com/lynicis/trbooksearch/cmd"

var version = "1.3.0"

func main() {
	cmd.SetVersionInfo(version)
	cmd.Execute()
}

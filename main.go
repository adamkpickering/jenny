package main

import "github.com/adamkpickering/jenny/cmd"

var version = "development"

func main() {
	cmd.SetVersionInfo(version)
	cmd.Execute()
}

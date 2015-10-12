package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	args := os.Args[1:]
	cli := &cli.CLI{
		Name:     "Depro",
		Version:  Version,
		Args:     args,
		HelpFunc: cli.BasicHelpFunc("depro"),
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}

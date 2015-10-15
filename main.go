package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"

	"github.com/EMSSConsulting/Depro/common"
	_ "github.com/EMSSConsulting/Depro/executor/shells" // Import the default shell providers
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	args := os.Args[1:]
	cli := &cli.CLI{
		Name:     "Depro",
		Version:  Version,
		Commands: common.Commands(),
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

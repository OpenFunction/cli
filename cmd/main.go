package main

import (
	"github.com/OpenFunction/cli/pkg/cmd"
)

func main() {
	cmds := cmd.NewDefaultCommand()
	cmds.Execute()
}

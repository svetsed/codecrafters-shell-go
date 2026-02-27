package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/internal/cmd/commands"
	"github.com/codecrafters-io/shell-starter-go/internal/utils/path"
)

var builtinCmd = map[string]bool{
	"exit":    true,
	"type":    true,
	"echo":    true,
	"pwd":     true,
	"cd": 	   true,
	"history": true,
}

func (cc *CurrentCmd) ExecBuiltinCmd() (errOutput error) {
	argsStr := cc.argsToString()

	output := ""

	switch cc.Cmd {
	case "exit":
		return
	case "cd":
		tmpArgStr := argsStr
		if strings.HasPrefix(tmpArgStr, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
			}

			tmpArgStr = strings.Replace(tmpArgStr, "~", homeDir, 1)
		}
		if _, err := os.Stat(tmpArgStr); err != nil {
			return fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
		}

		if err := os.Chdir(tmpArgStr); err != nil {
			return fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
		}
	case "pwd":
		curDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		output = fmt.Sprintf("%s", curDir)
	case "echo":
		output = fmt.Sprintf("%s", argsStr)
	case "type":
		if _, ok := builtinCmd[argsStr]; ok {
			output = fmt.Sprintf("%s is a shell builtin", argsStr)
		} else {
			output = path.PrintLookPath(argsStr, path.LookPath(argsStr))
		}
	case "history":
		tmp, err := commands.HandleHistoryCmd(cc.Args)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		output = tmp
	}

	if output != "" {
		_, errOutput = io.WriteString(cc.Stdout, output + "\n") // for echo -n do not work  
	}

	return errOutput
}

func (cc *CurrentCmd) argsToString() string {
	return strings.Join(cc.Args, " ")
}

func CheckIfBuiltinCmd(cmd string) bool {
	if _, exist := builtinCmd[cmd]; !exist {
		return false
	}
	return true
}
package executors

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/internal/utils/path"
)

var builtinCmd = map[string]bool{
	"exit": true,
	"type": true,
	"echo": true,
	"pwd":  true,
	"cd": 	true,
}

type CurrentCmd struct {
	Cmd			 string
	Args  		 []string
	Files 		 []string
	RedirectType string
	Stderr 		 *os.File
	Stdout 		 *os.File
	Flag		 int
}

func HandleInputToStruct(inputSlice []string) *CurrentCmd {
	curCmd := CurrentCmd{
		Cmd: inputSlice[0],
		Args:  make([]string, 0, 4),
		Files: make([]string, 0, 2),
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}

	needWrite := false
	for i := 1; i < len(inputSlice); i++ {
		if needWrite {
			curCmd.Files = append(curCmd.Files, inputSlice[i])
			needWrite = false
		} else if inputSlice[i] == ">" || inputSlice[i] == "1>" || inputSlice[i] == "2>" {
			needWrite = true
			curCmd.RedirectType = inputSlice[i]
			curCmd.Flag = os.O_CREATE | os.O_RDWR
		} else if inputSlice[i] == ">>" || inputSlice[i] == "1>>" || inputSlice[i] == "2>>" {
			needWrite = true
			curCmd.RedirectType = inputSlice[i]
			curCmd.Flag = os.O_CREATE | os.O_RDWR | os.O_APPEND
		} else {
			curCmd.Args = append(curCmd.Args, inputSlice[i])
		}
	}

	return &curCmd
}

func CheckIfBuiltinCmd(cmd string) bool {
	if _, exist := builtinCmd[cmd]; !exist {
		return false
	}
	return true
}

func (cc *CurrentCmd) ExecOtherCommand() error {
	path := path.LookPath(cc.Cmd)
	if path == "" {
		return fmt.Errorf("%s: command not found", cc.Cmd)
	}

	cmdForRun := exec.Command(cc.Cmd, cc.Args...)
	cmdForRun.Stdout = cc.Stdout
	cmdForRun.Stderr = cc.Stderr

	if err := cmdForRun.Run(); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (cc *CurrentCmd) argsToString() string {
	return strings.Join(cc.Args, " ")
}

func (cc *CurrentCmd) ExecBuiltinCmd() (output string, errOutput error) {
	argsStr := cc.argsToString()

	switch cc.Cmd {
	case "cd":
		tmpArgStr := argsStr
		if strings.HasPrefix(tmpArgStr, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
			}
			tmpArgStr = strings.Replace(tmpArgStr, "~", homeDir, 1)
		}
		if _, err := os.Stat(tmpArgStr); err != nil {
			errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
		} else {
			if err = os.Chdir(tmpArgStr); err != nil {
				errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
			}
		}
	case "pwd":
		if curDir, err := os.Getwd(); err == nil {
			output = fmt.Sprintf("%s", curDir)
		} else {
			errOutput = fmt.Errorf("%w", err)
		}
	case "echo":
		output = fmt.Sprintf("%s", argsStr)
	case "type":
		if _, ok := builtinCmd[argsStr]; ok {
			output = fmt.Sprintf("%s is a shell builtin", argsStr)
		} else {
			output = path.PrintLookPath(argsStr, path.LookPath(argsStr))
		}
	}

	return output, errOutput
}
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chzyer/readline"
)

type state int

const (
	stateOutside state = iota
	stateSingleQuote
	stateDoubleQuote
)

type parser struct {
    args    	  []string
    current 	  strings.Builder
	backslashSeen bool
	state   	  state
}

var existCmd = map[string]bool{
	"exit": true,
	"type": true,
	"echo": true,
	"pwd":  true,
	"cd": 	true,
}

type currentCmd struct {
	cmd			 string
	args  		 []string
	files 		 []string
	redirectType string
	stderr 		 *os.File
	stdout 		 *os.File
	flag		 int
}

func (cc *currentCmd) ExecOtherCommand() error {
	path := LookPath(cc.cmd)
	if path == "" {
		return fmt.Errorf("%s: command not found", cc.cmd)
	}

	cmdForRun := exec.Command(cc.cmd, cc.args...)
	cmdForRun.Stdout = cc.stdout
	cmdForRun.Stderr = cc.stderr

	if err := cmdForRun.Run(); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (cc *currentCmd) ArgsToString() string {
	return strings.Join(cc.args, " ")
}

func (cc *currentCmd) ExecSpecificCmd() (output string, errOutput error) {
	argsStr := cc.ArgsToString()

	switch cc.cmd {
	case "cd":
		tmpArgStr := argsStr
		if strings.HasPrefix(tmpArgStr, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.cmd, argsStr)
			}
			tmpArgStr = strings.Replace(tmpArgStr, "~", homeDir, 1)
		}
		if _, err := os.Stat(tmpArgStr); err != nil {
			errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.cmd, argsStr)
		} else {
			if err = os.Chdir(tmpArgStr); err != nil {
				errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.cmd, argsStr)
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
		if _, ok := existCmd[argsStr]; ok {
			output = fmt.Sprintf("%s is a shell builtin", argsStr)
		} else {
			output = PrintLookPath(argsStr, LookPath(argsStr))
		}
	}

	return output, errOutput
}

func HandleInputToStruct(inputSlice []string) *currentCmd {
	curCmd := currentCmd{
		cmd: inputSlice[0],
		args:  make([]string, 0, 4),
		files: make([]string, 0, 2),
		stderr: os.Stderr,
		stdout: os.Stdout,
	}

	needWrite := false
	for i := 1; i < len(inputSlice); i++ {
		if needWrite {
			curCmd.files = append(curCmd.files, inputSlice[i])
			needWrite = false
		} else if inputSlice[i] == ">" || inputSlice[i] == "1>" || inputSlice[i] == "2>" {
			needWrite = true
			curCmd.redirectType = inputSlice[i]
			curCmd.flag = os.O_CREATE | os.O_RDWR
		} else if inputSlice[i] == ">>" || inputSlice[i] == "1>>" || inputSlice[i] == "2>>" {
			needWrite = true
			curCmd.redirectType = inputSlice[i]
			curCmd.flag = os.O_CREATE | os.O_RDWR | os.O_APPEND
		} else {
			curCmd.args = append(curCmd.args, inputSlice[i])
		}
	}

	return &curCmd
}

func completer() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("echo"),
		readline.PcItem("exit"),
	)
}

func main() {
	config := &readline.Config{
		Prompt: "$ ",
		AutoComplete: completer(),
	}
	
	rl, err := readline.NewEx(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	for {

		inputRaw, err := rl.Readline()
		if err != nil {
			// io.EOF (Ctrl+D) / readline.ErrInterrupt (Ctrl+C)
			break
		}

		inputSlice, err := ParseInput(inputRaw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		if inputSlice[0] == "exit" {
			return
		}
		
		curCmd := HandleInputToStruct(inputSlice)

		if len(curCmd.files) > 0 {
			for i, filename := range curCmd.files {
				tmp, err := os.OpenFile(filename, curCmd.flag, 0766)
				if err != nil {
					fmt.Fprintf(curCmd.stderr, "%v\n", err)
				}

				defer tmp.Close()

				if i == len(curCmd.files) - 1 {
					if curCmd.redirectType == "2>" || curCmd.redirectType == "2>>" {
						curCmd.stderr = tmp
					} else  if curCmd.redirectType == ">" || curCmd.redirectType == ">>" || curCmd.redirectType == "1>" || curCmd.redirectType == "1>>" {
						curCmd.stdout = tmp
					} 
				}
			}
		}

		if _, ok := existCmd[curCmd.cmd]; ok {
			output, err := curCmd.ExecSpecificCmd()
			if err != nil {
				fmt.Fprintf(curCmd.stderr, "%v\n", err)
			}

			if output != "" {
				fmt.Fprintf(curCmd.stdout,"%s\n", output)
				if err != nil {
					fmt.Fprintf(curCmd.stderr, "%v\n", err)
				}

			}
		} else {
			err := curCmd.ExecOtherCommand()
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					fmt.Fprintf(curCmd.stderr, "%v\n", err)
				}
			}
		}

	}
}

func ParseInput(input string) ([]string, error) {
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	prsr := parser{
		args: []string{},
		current: strings.Builder{},
		backslashSeen: false,
		state: stateOutside,
	}

	for _, ch := range input {
		if prsr.state == stateOutside {
			if prsr.backslashSeen {
				prsr.current.WriteRune(ch)
				prsr.backslashSeen = false
			} else if ch == '\\' {
				prsr.backslashSeen = true
			} else if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
				if prsr.current.Len() > 0 {
					prsr.args = append(prsr.args, prsr.current.String())
					prsr.current.Reset()
				}
			} else if ch == '\'' {
				prsr.state = stateSingleQuote
			} else if ch == '"' {
				prsr.state = stateDoubleQuote
			} else if ch != '\\' {
				prsr.current.WriteRune(ch)
			}
		} else if prsr.state == stateSingleQuote {
			if ch == '\''{
				prsr.state = stateOutside
			} else {
				prsr.current.WriteRune(ch)
			}
		} else if prsr.state == stateDoubleQuote {
			if prsr.backslashSeen {
				if ch == '\\' || ch == '"' { // $, `
					prsr.current.WriteRune(ch)
				} else {
					prsr.current.WriteRune('\\')
					prsr.current.WriteRune(ch)
				}
				prsr.backslashSeen = false
			} else {
				if ch == '\\' {
					prsr.backslashSeen = true
				} else if ch == '"' {
					prsr.state = stateOutside
				} else {
					prsr.current.WriteRune(ch)
				}
			}
		}
	}

	if prsr.current.Len() > 0 {
		prsr.args = append(prsr.args, prsr.current.String())
		prsr.current.Reset()
	}


	return prsr.args, nil
}

func PrintLookPath(cmd, path string) string {
	if path == "" {
		return fmt.Sprintf("%s: not found", cmd)
	}

	return fmt.Sprintf("%s is %s", cmd, path)
}

func LookPath(filename string) string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return ""
	}

	listPath := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, dir := range listPath {
		path := filepath.Join(dir, filename)
		info, err := os.Stat(path)
		if err != nil  {
			continue
		} 

		if !info.IsDir() {
			if isExec := isExecutable(path, info); isExec {
				return path
			}
		}
	}

	return ""
}

func isExecutable(path string, info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		ext := filepath.Ext(path)
		winExecExts := []string{".exe", ".com", ".bat", ".cmd"}
		for _, e := range winExecExts {
			if strings.EqualFold(ext, e) {
				return true
			}
		}

		return false
	} else {
		mode := info.Mode()
		return mode&0111 != 0
	}
}
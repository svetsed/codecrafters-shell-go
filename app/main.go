package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

type currentCmd struct {
	cmd			 string
	args  		 []string
	files 		 []string
	redirectType string
}

var existCmd = map[string]bool{
	"exit": true,
	"type": true,
	"echo": true,
	"pwd":  true,
	"cd": 	true,
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

// // closing file if have been mistake
// func create_files(filenames []string) ([]*os.File, error) {
// 	if len(filenames) == 0 {
// 		return nil, fmt.Errorf("no files for creating")
// 	}

// 	files := []*os.File{}

// 	for _, filename := range filenames {
// 		tmp, err := os.Create(filename)
// 		if err != nil {
// 			_ = close_files(files)
// 			return nil, fmt.Errorf("%v\n", err)
// 		}

// 		files = append(files, tmp)
// 	}

// 	return files, nil
// }

// func close_files(files []*os.File) error {
// 	if files == nil {
// 		return fmt.Errorf("no files for closing")
// 	}

// 	for _, file := range files {
// 		_ = file.Close() // check err
// 	}

// 	return nil
// }

func main() {
	for {
		fmt.Print("$ ")
		inputRaw, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		inputSlice, err := ParseInput(inputRaw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		if inputSlice[0] == "exit" {
			return
		}
		
		curCmd := currentCmd{
			cmd: inputSlice[0],
			args:  make([]string, 0, 4),
			files: make([]string, 0, 2),
		}

		needWrite := false
		for i := 1; i < len(inputSlice); i++ {
			if needWrite {
				curCmd.files = append(curCmd.files, inputSlice[i])
				needWrite = false
			} else if inputSlice[i] == ">" || inputSlice[i] == "1>" || inputSlice[i] == "2>" {
				needWrite = true
				curCmd.redirectType = inputSlice[i]
			} else {
				curCmd.args = append(curCmd.args, inputSlice[i])
			}
		}


		argsStr := strings.Join(curCmd.args, " ")

		var stderr *os.File = os.Stderr
		var stdout *os.File = os.Stdout

		if len(curCmd.files) > 0 {
			for i, filename := range curCmd.files {
				tmp, err := os.Create(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
				}

				defer tmp.Close()

				if i == len(curCmd.files) - 1 {
					if curCmd.redirectType == "2>" {
						stderr = tmp
					} else  if curCmd.redirectType == ">" ||  curCmd.redirectType == "1>" {
						stdout = tmp
					} 
				}
			}
		}

		if _, ok := existCmd[curCmd.cmd]; ok {
			output, err := ExecSpecificCmd(curCmd.cmd, argsStr)
			if err != nil {
				fmt.Fprintf(stderr, "%v\n", err)
			}

			if output != "" {
				fmt.Fprintf(stdout,"%s\n", output)
				if err != nil {
					fmt.Fprintf(stderr, "%v\n", err)
				}

			}
		} else {
			err := ExecOtherCommand(curCmd.cmd, curCmd.args, curCmd.files, curCmd.redirectType)
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					fmt.Fprintf(stderr, "%v\n", err)
				}
			}
		}

	}
}

func ExecOtherCommand(cmd string, argsSlice, filesSlice []string, redirectType string) error {
	path := LookPath(cmd)
	if path == "" {
		return fmt.Errorf("%s: command not found", cmd)
	}

	var stdout *os.File = os.Stdout
	var stderr *os.File = os.Stderr

	for i, filename := range filesSlice {
		tmp, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		defer tmp.Close()

		if i == len(filesSlice) - 1 {
			if redirectType == "2>" {
				stderr = tmp
			} else  if redirectType == ">" ||  redirectType == "1>" {
				stdout = tmp
			} 
		}
	}

	cmdForRun := exec.Command(cmd, argsSlice...)
	cmdForRun.Stdout = stdout
	cmdForRun.Stderr = stderr

	if err := cmdForRun.Run(); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
} 


func ExecSpecificCmd(cmd string, argsStr string) (output string, errOutput error) {
	switch cmd {
	case "cd":
		tmpArgStr := argsStr
		if strings.HasPrefix(tmpArgStr, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				errOutput = fmt.Errorf("%s: %s: No such file or directory", cmd, argsStr)
			}
			tmpArgStr = strings.Replace(tmpArgStr, "~", homeDir, 1)
		}
		if _, err := os.Stat(tmpArgStr); err != nil {
			errOutput = fmt.Errorf("%s: %s: No such file or directory", cmd, argsStr)
		} else {
			if err = os.Chdir(tmpArgStr); err != nil {
				errOutput = fmt.Errorf("%s: %s: No such file or directory", cmd, argsStr)
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
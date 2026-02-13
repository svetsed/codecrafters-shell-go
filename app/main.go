package main

import (
	"bufio"
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

func PrintLookPath(cmd, path string) {
	if path == "" {
		fmt.Printf("%s: not found\n", cmd)
	} else {
		fmt.Printf("%s is %s\n", cmd, path)
	}
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

func main() {
	existCmd := map[string]bool{
		"exit": true,
		"type": true,
		"echo": true,
		"pwd":  true,
		"cd": 	true,
	}

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

		cmd := inputSlice[0]
		argsStr := strings.Join(inputSlice[1:], " ")

		switch cmd {
		case "cd":
			tmpArgStr := argsStr
			if strings.HasPrefix(tmpArgStr, "~") {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					fmt.Printf("%s: %s: No such file or directory\n", cmd, argsStr)
				}
				tmpArgStr = strings.Replace(tmpArgStr, "~", homeDir, 1)
			}

			if _, err = os.Stat(tmpArgStr); err != nil {
				fmt.Printf("%s: %s: No such file or directory\n", cmd, argsStr)
			} else {
				if err = os.Chdir(tmpArgStr); err != nil {
					fmt.Printf("%s: %s: No such file or directory\n", cmd, argsStr)
				}
			}
		case "pwd":
			if curDir, err := os.Getwd(); err == nil {
				fmt.Printf("%s\n", curDir)
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
		case "exit":
			os.Exit(0)
		case "echo":
			fmt.Printf("%s\n", argsStr)
		case "type":
			if _, ok := existCmd[argsStr]; ok {
				fmt.Printf("%s is a shell builtin\n", argsStr)
			} else {
				PrintLookPath(argsStr, LookPath(argsStr))
			}
		case "cat":
			cmdForRun := exec.Command("cat", inputSlice[1:]...)
			cmdForRun.Stdout = os.Stdout
			cmdForRun.Stderr = os.Stderr
			if err = cmdForRun.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		default:
			path := LookPath(cmd)
			if path == "" {
				fmt.Printf("%s: command not found\n", cmd)
			} else {
				cmdForRun := exec.Command(cmd, inputSlice[1:]...)
				cmdForRun.Stdout = os.Stdout
				cmdForRun.Stderr = os.Stderr
				if err = cmdForRun.Run(); err != nil {
					fmt.Fprintln(os.Stderr, err)
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
				if ch == '\\' || ch == '"' {
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
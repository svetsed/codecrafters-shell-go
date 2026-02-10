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
	}

	for {
		fmt.Print("$ ")
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		input = strings.TrimSpace(input)
		args := strings.Fields(input)

		if len(args) == 0 {
			continue
		}

		cmd := args[0]
		argsStr := strings.Join(args[1:], " ")

		switch cmd {
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
		default:
			path := LookPath(cmd)
			if path == "" {
				fmt.Printf("%s: command not found\n", cmd)
			} else {
				cmdForRun := exec.Command(cmd, args[1:]...)
				cmdForRun.Stdout = os.Stdout
				cmdForRun.Stderr = os.Stderr
				if err = cmdForRun.Run(); err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
			}

		}
	}
}

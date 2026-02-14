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

		// возможно надо сделать свой тип в котором будет аргументы, файл куда записывать и было ли перенаправление
		// и ее уже возвращать
		// команда
		// аргументы
		// было ли перенаправление
		// файлы куда записывать
		// ошибки
		// какой вывод туда записывать
type Cmd struct {
	cmd string
	args []string
	filename string
	needRedirect bool
}


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

		// получить инпут и  разобрать его на кусочки
		// получает какую-то основную команду из первого аргумента
		// если ее нет, то ошибку выдает
		// если есть, то идет но конца аргументов?
		// если перенаправление, то создает файл или перезаписывает
		// каждый аргумент отдает команде и записывает вывод ее, смотрит была ли ошибка?
		// если ошибка, то выводит в консоль
		// иначе записывает или перезапизаписывает в файл

		// надо вывести весь вывод в одну функцию, которая будет принимать флаг перенаправлять или нет, куда писать (имя файла) и сообщение

		cmd := inputSlice[0]

		if cmd == "exit" {
			return
		}

		filesSlice := make([]string, 0, 2)
		argsSlice := make([]string, 0, 4)
		needWrite := false
		for i := 1; i < len(inputSlice); i++ {
			if needWrite {
				filesSlice = append(filesSlice, inputSlice[i])
				needWrite = false
			} else if inputSlice[i] == ">" || inputSlice[i] == "1>" {
				needWrite = true
			} else {
				argsSlice = append(argsSlice, inputSlice[i])
			}
		}


		argsStr := strings.Join(argsSlice, " ")

		if _, ok := existCmd[cmd]; ok {
			output, err := ExecSpecificCmd(cmd, argsStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}

			if output != "" {
				if len(filesSlice) == 0 {
					fmt.Printf("%s\n", output)
				} else {
					var whereWrite *os.File = os.Stdout

					for i, filename := range filesSlice {
					tmp, err := os.Create(filename)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
					}

					defer tmp.Close()

					if i == len(filesSlice) - 1 {
						whereWrite = tmp
					}

					_, err = whereWrite.WriteString(output + "\n")
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
					}
				}
	}
			}
		} else {
			err := ExecOtherCommand(cmd, argsSlice, filesSlice)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		}


	}
}

// реализовать: она должна записать в файл если ее попросят
func ExecOtherCommand(cmd string, argsSlice, filesSlice []string) error {
	path := LookPath(cmd)
	if path == "" {
		return fmt.Errorf("%s: command not found", cmd)
	}

	var whereWrite *os.File = os.Stdout


	for i, filename := range filesSlice {
		tmp, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		defer tmp.Close()

		if i == len(filesSlice) - 1 {
			whereWrite = tmp
		}
	}

	cmdForRun := exec.Command(cmd, argsSlice...)
	cmdForRun.Stdout = whereWrite
	cmdForRun.Stderr = os.Stderr

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
	// case "cat":
	// 	cmdForRun := exec.Command("cat", inputSlice[1:]...)
	// 	cmdForRun.Stdout = os.Stdout
	// 	cmdForRun.Stderr = os.Stderr
	// 	if err = cmdForRun.Run(); err != nil {
	// 		fmt.Fprintln(os.Stderr, err)
	// 	}
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
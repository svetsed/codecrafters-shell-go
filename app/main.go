package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/chzyer/readline"
	"github.com/codecrafters-io/shell-starter-go/internal/completer"
	"github.com/codecrafters-io/shell-starter-go/internal/executors"
	"github.com/codecrafters-io/shell-starter-go/internal/handlers"
)

func main() {
	rl, err := readline.NewEx(&readline.Config{
		Prompt: "$ ",
		AutoComplete: completer.NewCmdCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt: "exit",
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return
	}

	defer func() {
		if err := rl.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing readline: %v\n", err)
		}
	}()

	for {
		inputRaw, err := rl.Readline()
		if err != nil {
			// io.EOF (Ctrl+D) / readline.ErrInterrupt (Ctrl+C)
			break
		}

		inputSlice, err := handlers.ParseInput(inputRaw)
		// skip empty input
		if err != nil {
			continue
		}

		if inputSlice[0] == "exit" {
			return
		}


		curCmd := executors.HandleInputToStruct(inputSlice)

		if len(curCmd.Files) > 0 {
			for i, filename := range curCmd.Files {
				tmp, err := os.OpenFile(filename, curCmd.Flag, 0766)
				if err != nil {
					fmt.Fprintf(curCmd.Stderr, "%v\n", err)
				}

				defer tmp.Close()

				if i == len(curCmd.Files) - 1 {
					if curCmd.RedirectType == "2>" || curCmd.RedirectType == "2>>" {
						curCmd.Stderr = tmp
					} else  if curCmd.RedirectType == ">" || curCmd.RedirectType == ">>" || curCmd.RedirectType == "1>" || curCmd.RedirectType == "1>>" {
						curCmd.Stdout = tmp
					} 
				}
			}
		}

		if executors.CheckIfBuiltinCmd(curCmd.Cmd) {
			output, err := curCmd.ExecBuiltinCmd()
			if err != nil {
				fmt.Fprintf(curCmd.Stderr, "%v\n", err)
			}

			if output != "" {
				fmt.Fprintf(curCmd.Stdout,"%s\n", output)
				if err != nil {
					fmt.Fprintf(curCmd.Stderr, "%v\n", err)
				}

			}
		} else {
			err := curCmd.ExecOtherCommand()
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					fmt.Fprintf(curCmd.Stderr, "%v\n", err)
				}
			}
		}
	}
}
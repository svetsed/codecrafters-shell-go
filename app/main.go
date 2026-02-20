package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

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

		inputSliceCmds, err := handlers.ParseInput(inputRaw)
		// skip empty input
		if err != nil {
			continue
		}
		
		if len(inputSliceCmds) == 1 && inputSliceCmds[0][0] == "exit" {
			return
		}

		cmds := executors.HandleInputToCmds(inputSliceCmds)

		if cmds.CountCmd == 2 { 

			r, w, err := os.Pipe()
			if err != nil {
				continue // fmt.Fprintf(curCmd.Stderr, "%v\n", err)
			}

			cmds.Cmds[0].Stdout = w
			cmds.Cmds[1].Stdin = r

			if executors.CheckIfBuiltinCmd(cmds.Cmds[1].Cmd) { // if right cmd dont read stdin
				devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
				if err != nil {
					w.Close()
					r.Close()
					continue
				}

				defer func () {
					devNull.Close()
				}()

				cmds.Cmds[0].Stdout = devNull
			}

			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer w.Close()
				err := cmds.Cmds[0].Run()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer r.Close()
				err := cmds.Cmds[1].Run()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
			}()

			wg.Wait()


		} else if cmds.CountCmd == 1 {
			curCmd := cmds.Cmds[0]
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
				err := curCmd.ExecBuiltinCmd()
				if err != nil {
					fmt.Fprintf(curCmd.Stderr, "%v\n", err)
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
}
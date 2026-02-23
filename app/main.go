package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/chzyer/readline"
	"github.com/codecrafters-io/shell-starter-go/internal/commands/history"
	"github.com/codecrafters-io/shell-starter-go/internal/completer"
	"github.com/codecrafters-io/shell-starter-go/internal/executors"
	"github.com/codecrafters-io/shell-starter-go/internal/handlers"
)

func main() {
	historyFile, err := history.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	defer historyFile.ClearHistory()

	executors.HistoryFile = historyFile

	rl, err := readline.NewEx(&readline.Config{
		Prompt: "$ ",
		AutoComplete: completer.NewCmdCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt: "exit",
		HistoryFile: historyFile.HistoryPath,
		DisableAutoSaveHistory: true,
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
		

		historyFile.Mu.Lock()
		if err := rl.SaveHistory(inputRaw); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			historyFile.Mu.Unlock()
			return
		}

		if inputRaw != "" {
			historyFile.CounterLine++
		}
		historyFile.Mu.Unlock()
		

		inputSliceCmds, err := handlers.ParseInput(inputRaw)
		// skip empty input
		if err != nil {
			continue
		}

		if len(inputSliceCmds) == 1 && inputSliceCmds[0][0] == "exit" {
			return
		}

		cmds := executors.HandleInputToCmds(inputSliceCmds)

		for _, cmd := range cmds.Cmds {
			if err := cmd.SetupRedirection(); err != nil {
				fmt.Fprintf(cmd.Stderr, "%v\n", err)
        		continue
			}
			defer cmd.CloseFiles()
		}

		if cmds.CountCmd > 1 {
			cmds.ExecPipeline()

		} else if cmds.CountCmd == 1 {
			curCmd := cmds.Cmds[0]

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


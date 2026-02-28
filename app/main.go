package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/chzyer/readline"
	"github.com/codecrafters-io/shell-starter-go/internal/completer"
	"github.com/codecrafters-io/shell-starter-go/internal/cmd"
	"github.com/codecrafters-io/shell-starter-go/internal/cmd/commands"
	"github.com/codecrafters-io/shell-starter-go/internal/history"
	"github.com/codecrafters-io/shell-starter-go/internal/parser"
)



func main() {
	history := history.NewHistory()

	// load old history
	historyFilename := os.Getenv("HISTFILE")
	if historyFilename != "" {
		err := history.ReadHistoryFromFile(historyFilename)
		if err != nil {
			return
		}

		defer history.AppendHistoryToFile(historyFilename)
	}

	commands.History = &history

	rl, err := readline.NewEx(&readline.Config{
		Prompt: "$ ",
		AutoComplete: completer.NewCmdCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt: "exit",
		Listener: readline.FuncListener(history.WalkByHistory),
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
		
		if inputRaw == "" {
			continue
		}

		history.PushBackOneLine(inputRaw, true)
		
		inputSliceCmds, err := parser.ParseInput(inputRaw)
		// skip empty input
		if err != nil {
			continue
		}

		if len(inputSliceCmds) == 1 && inputSliceCmds[0][0] == "exit" {
			return
		}

		cmds := parser.HandleInputToCmds(inputSliceCmds)

		for _, curCmd := range cmds.Cmds {
			if err := curCmd.SetupRedirection(); err != nil {
				fmt.Fprintf(curCmd.Stderr, "%v\n", err)
        		continue
			}
			defer curCmd.CloseFiles()
		}

		if cmds.CountCmd > 1 {
			cmds.ExecPipeline()
		} else if cmds.CountCmd == 1 {
			curCmd := cmds.Cmds[0]

			if cmd.CheckIfBuiltinCmd(curCmd.Cmd) {
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
package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/internal/cmd"
	"github.com/codecrafters-io/shell-starter-go/internal/pipeline"
)

type state int

const (
	stateOutside state = iota
	stateSingleQuote			// '' don't work \
	stateDoubleQuote			// "" \ work for \ " ($ ` not implemented)
)

type parser struct {
    args    	  [][]string
    current 	  strings.Builder
	backslashSeen bool
	state   	  state
}

// ParseInput is the main parser, divides into pipe and argument, taking into account the state.
func ParseInput(input string) ([][]string, error) {
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	prsr := parser{
		args: make([][]string, 1),
		current: strings.Builder{},
		backslashSeen: false,
		state: stateOutside,
	}

	indexCmd := 0
	for _, ch := range input {
		switch prsr.state {
		case stateOutside:
			if prsr.backslashSeen {
				prsr.current.WriteRune(ch)
				prsr.backslashSeen = false
			} else if ch == '\\' {
				prsr.backslashSeen = true
			} else if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '|' { // arg or pipe is over
				if prsr.current.Len() > 0 {
					// for safe access by index
					if indexCmd >= len(prsr.args) {
						for i := len(prsr.args); i <= indexCmd; i++ {
							prsr.args = append(prsr.args, []string{})
						}
					}
					prsr.args[indexCmd] = append(prsr.args[indexCmd], prsr.current.String())
					prsr.current.Reset()
				}

				if ch == '|' {
					indexCmd++
				}

			} else if ch == '\'' {
				prsr.state = stateSingleQuote
			} else if ch == '"' {
				prsr.state = stateDoubleQuote
			} else if ch != '\\' { // handle separately
				prsr.current.WriteRune(ch)
			}

		case stateSingleQuote:
			if ch == '\''{
				prsr.state = stateOutside
			} else {
				prsr.current.WriteRune(ch)
			}
		case stateDoubleQuote:
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
		// for safe access by index
		if indexCmd >= len(prsr.args) {
			for i := len(prsr.args); i <= indexCmd; i++ {
				prsr.args = append(prsr.args, []string{})
			}
		}
		prsr.args[indexCmd] = append(prsr.args[indexCmd], prsr.current.String())
		prsr.current.Reset()
	}


	return prsr.args, nil
}

// HandleInputToCmds completes processing after ParseInput in to struct Cmds.
func HandleInputToCmds(inputCmdsSlice [][]string) *pipeline.Cmds {
	c := pipeline.Cmds{
		Cmds: make([]*cmd.CurrentCmd, 0, 2),
	}

	for _, cmd := range inputCmdsSlice {
		c.Cmds = append(c.Cmds, handleInputToOneCmd(cmd))
		c.CountCmd++
	}

	return &c
}

// handleInputToOneCmd create CurrentCmd, fills in the fields if there were redirects.
func handleInputToOneCmd(inputSlice []string) *cmd.CurrentCmd {
	curCmd := cmd.CurrentCmd{
		Cmd: inputSlice[0],
		Args:  make([]string, 0, 4),
		Stderr: os.Stderr,
		Stdout: os.Stdout,
		Stdin: os.Stdin,
		Redirect: cmd.Redirect{
			Files: make([]string, 0, 2),
		},
	}

	needWrite := false
	for i := 1; i < len(inputSlice); i++ {
		if needWrite {
			curCmd.Files = append(curCmd.Files, inputSlice[i])
			needWrite = false
		} else if inputSlice[i] == ">" || inputSlice[i] == "1>" || inputSlice[i] == "2>" {
			// can be rewritten if file exist
			needWrite = true
			curCmd.RedirectType = inputSlice[i]
			curCmd.Flag = os.O_CREATE | os.O_RDWR
		} else if inputSlice[i] == ">>" || inputSlice[i] == "1>>" || inputSlice[i] == "2>>" {
			// needs to be added to the end of the file f file exist
			needWrite = true
			curCmd.RedirectType = inputSlice[i]
			curCmd.Flag = os.O_CREATE | os.O_RDWR | os.O_APPEND
		} else {
			curCmd.Args = append(curCmd.Args, inputSlice[i])
		}
	}

	return &curCmd
}
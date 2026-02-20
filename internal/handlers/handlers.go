package handlers

import (
	"fmt"
	"strings"
)

type state int

const (
	stateOutside state = iota
	stateSingleQuote
	stateDoubleQuote
)

type parser struct {
    args    	  [][]string
    current 	  strings.Builder
	backslashSeen bool
	state   	  state
}

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
			} else if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '|' {
				if prsr.current.Len() > 0 {
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
			} else if ch != '\\' {
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

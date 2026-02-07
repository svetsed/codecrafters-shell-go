package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
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
		case "exit":
			os.Exit(0)
		case "echo":
			fmt.Printf("%s\n", argsStr)
		default:
			fmt.Printf("%s: command not found\n", cmd)
		}
	}
}

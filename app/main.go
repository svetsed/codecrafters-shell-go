package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Print("$ ")

	cmd, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	cmd = strings.TrimSuffix(cmd, "\r")
	cmd = strings.TrimSuffix(cmd, "\n")

	fmt.Printf("%s: command not found\n", cmd)
}

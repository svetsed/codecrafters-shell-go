[![progress-banner](https://backend.codecrafters.io/progress/shell/a0ce141b-bdea-4feb-a164-d00a0ece7416)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)

# Build Your Own Shell in Go 

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org/)
[![CodeCrafters](https://img.shields.io/badge/CodeCrafters-Shell%20Challenge-ff69b4.svg)](https://codecrafters.io/)

A POSIX-compliant shell implementation built from scratch in Go as part of the CodeCrafters "Build Your Own Shell" challenge. This project demonstrates core systems programming concepts including command parsing, process execution, and shell built-in commands.

## Features

- **Interactive REPL** – Read-Eval-Print Loop using `readline` library
- **Built-in Commands** – `cd`, `pwd`, `echo`, `exit`, `type` and more
- **External Program Execution** – Runs any program available in PATH
- **Command Parsing** – Handles single and double quotes, escape sequences, and spaces
- **Path Resolution** – Searches PATH for executables
- **Redirection** – Support for redirecting a command's output to a file (> or 1>, >>, 2>, 2>>)
- **Command and Argument Completion on TAB** - Supports with LCP logic and sound
- **Pipelines** – Support for more than 2 commands in pipeline (connecting the output of each command to the input of the next one)
- **Custom History Support** – My own implementation using linked list

Background Jobs soon!


### Installation and running

```bash
git clone https://github.com/svetsed/codecrafters-shell-go.git
cd codecrafters-shell-go
go build -o goshell ./app
./goshell
```

### If you want to try too

This is a starting point for Go solutions to the
["Build Your Own Shell" Challenge](https://app.codecrafters.io/courses/shell/overview).

### Note

I push to this repository automatically using the codecrafters utility, but unfortunately, i can't do more meaningful commits there.

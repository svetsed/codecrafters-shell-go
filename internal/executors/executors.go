package executors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/codecrafters-io/shell-starter-go/internal/utils/path"
)

var builtinCmd = map[string]bool{
	"exit": true,
	"type": true,
	"echo": true,
	"pwd":  true,
	"cd": 	true,
}

type Cmds struct {
	Cmds	 []*CurrentCmd
	CountCmd int
	Wg 		 sync.WaitGroup
}

type CurrentCmd struct {
	Cmd			 string
	Args  		 []string
	Files 		 []string   // save just filename
	RedirectType string
    Stdin  		 io.Reader
    Stdout 		 io.Writer
    Stderr 		 io.Writer
	Flag		 int		// for openning file
	filesToClose []*os.File
}

func HandleInputToCmds(inputCmdsSlice [][]string) *Cmds {
	c := Cmds{
		Cmds: make([]*CurrentCmd, 0, 2),
	}

	for _, cmd := range inputCmdsSlice {
		c.Cmds = append(c.Cmds, handleInputToOneCmd(cmd))
		c.CountCmd++
	}

	return &c
}

func handleInputToOneCmd(inputSlice []string) *CurrentCmd {
	curCmd := CurrentCmd{
		Cmd: inputSlice[0],
		Args:  make([]string, 0, 4),
		Files: make([]string, 0, 2),
		Stderr: os.Stderr,
		Stdout: os.Stdout,
		Stdin: os.Stdin,
	}

	needWrite := false
	for i := 1; i < len(inputSlice); i++ {
		if needWrite {
			curCmd.Files = append(curCmd.Files, inputSlice[i])
			needWrite = false
		} else if inputSlice[i] == ">" || inputSlice[i] == "1>" || inputSlice[i] == "2>" {
			needWrite = true
			curCmd.RedirectType = inputSlice[i]
			curCmd.Flag = os.O_CREATE | os.O_RDWR
		} else if inputSlice[i] == ">>" || inputSlice[i] == "1>>" || inputSlice[i] == "2>>" {
			needWrite = true
			curCmd.RedirectType = inputSlice[i]
			curCmd.Flag = os.O_CREATE | os.O_RDWR | os.O_APPEND
		} else {
			curCmd.Args = append(curCmd.Args, inputSlice[i])
		}
	}

	return &curCmd
}

func (c *Cmds) CreatePipeline() (readers []*os.File, writers []*os.File, err error) {
	readers = make([]*os.File, c.CountCmd)
	writers = make([]*os.File, c.CountCmd)

	for i:= 0; i < c.CountCmd - 1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			if i > 0 {
				for j := i - 1; j >= 0; j-- {
					if readers[j] != nil {
						readers[j].Close()
					}
					if writers[j] != nil {
						writers[j].Close()
					}
				}
			}
			return nil, nil, err
		}

		writers[i] = w
		readers[i+1] = r
	}

	return readers, writers, nil
}

// TODO check redirection
func(c *Cmds) SetupCmdPipe(i int, readers, writers []*os.File, devNull *os.File) {
	// if readers == nil || writers == nil || devNull == nil  || i < 0 {
	// 	return fmt.Errorf("received incorrect data")
	// }

	cmd := c.Cmds[i]

	if readers[i] != nil {
		cmd.Stdin = readers[i]
	}

	if cmd.RedirectType == ">" || cmd.RedirectType == ">>" || cmd.RedirectType == "1>" || cmd.RedirectType == "1>>" {
		writers[i].Close()
	} else if i < c.CountCmd - 1 {
		nextCmd := c.Cmds[i+1]
		if CheckIfBuiltinCmd(nextCmd.Cmd) {
			cmd.Stdout = devNull
			writers[i].Close()
		} else {
			cmd.Stdout = writers[i]
		}
	} else {
		cmd.Stdout = os.Stdout
	}
}

func (c *Cmds) ExecPipeline() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open devnull: %v\n", err)
		return
	}
	defer devNull.Close()

	readers, writers, err := c.CreatePipeline()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pipeline: %v\n", err)
        return
	}

	for i := range c.Cmds {
		c.SetupCmdPipe(i, readers, writers, devNull)
	}

	type errorCmd struct {
		i   int
		err error
	}

 	errCh := make(chan errorCmd, c.CountCmd)

	for i := range c.Cmds {
		i := i
		c.Wg.Add(1)

		go func() {
			defer c.Wg.Done()

			defer func() {
				if readers[i] != nil {
					readers[i].Close()
				}
				if writers[i] != nil {
					writers[i].Close()
				}
			}()

			select {
			case <- ctx.Done():
				return
			default:
			}

			err := c.Cmds[i].Run()
			if err != nil {
				if strings.Contains(err.Error(), "broken pipe") {
					if CheckIfBuiltinCmd(c.Cmds[i].Cmd) {
						return
					}
				}

				select {
				case errCh <- errorCmd{i: i, err: err}:
					cancel()
				default:
				}
			}
		}()
	}

	go func() {
		c.Wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		var exitErr *exec.ExitError
		if !errors.As(err.err, &exitErr) {
			fmt.Fprintf(c.Cmds[err.i].Stderr, "%v\n", err.err)
		}
	}
}

func (cc *CurrentCmd) CorrectRedirectType() bool {
	availRedType := map[string]bool{
		">":   true,
		">>":  true,
		"1>":  true,
		"1>>": true,
		"2>":  true,
		"2>>": true,
	}

	if _, exist := availRedType[cc.RedirectType]; exist {
		return true
	}

	return false
}

func (cc *CurrentCmd) SetupRedirection() error {
	if len(cc.Files) == 0 && cc.RedirectType == ""  {
		return nil
	}

	if !cc.CorrectRedirectType() {
		return fmt.Errorf("unknown redirect type: %s", cc.RedirectType)
	}

	files := make([]*os.File, 0, len(cc.Files))
	for _, filename := range cc.Files {
		f, err := os.OpenFile(filename, cc.Flag, 0766)
		if err != nil {
			for _, opened := range files {
				opened.Close()
			}
			return fmt.Errorf("%v\n", err)
		}
		files = append(files, f)
	}

	lastFile := files[len(files)-1]

	switch cc.RedirectType {
	case "2>", "2>>":
		cc.Stderr = lastFile
	case ">", ">>", "1>", "1>>":
		cc.Stdout = lastFile
	}

	cc.filesToClose = files[:len(files)-1]

	return nil
}

func (cc *CurrentCmd) CloseFiles() {
	if cc.filesToClose != nil {
		for _, f := range cc.filesToClose {
			f.Close()
		}
		cc.filesToClose = nil
	}

}

func CheckIfBuiltinCmd(cmd string) bool {
	if _, exist := builtinCmd[cmd]; !exist {
		return false
	}
	return true
}

func (cc *CurrentCmd) Run() error {
	if CheckIfBuiltinCmd(cc.Cmd) {
		return cc.ExecBuiltinCmd()
	}

	cmd, err := cc.BuildCmd()
	if err != nil {
		return err
	}

	 if err := cmd.Start(); err != nil {
        return err
    }
    
    return cmd.Wait()
}

func (cc *CurrentCmd) BuildCmd() (*exec.Cmd, error) {
	path := path.LookPath(cc.Cmd)
	if path == "" {
		return nil, fmt.Errorf("%s: command not found", cc.Cmd)
	}

	cmd := exec.Command(cc.Cmd, cc.Args...)
	cmd.Stdin  = cc.Stdin
	cmd.Stdout = cc.Stdout
	cmd.Stderr = cc.Stderr

	return cmd, nil
}

func (cc *CurrentCmd) ExecOtherCommand() error {
	cmdForRun, err := cc.BuildCmd()
	if err != nil {
		return err
	}

	if err := cmdForRun.Run(); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (cc *CurrentCmd) argsToString() string {
	return strings.Join(cc.Args, " ")
}

func (cc *CurrentCmd) ExecBuiltinCmd() (errOutput error) {
	argsStr := cc.argsToString()

	output := ""

	switch cc.Cmd {
	case "cd":
		tmpArgStr := argsStr
		if strings.HasPrefix(tmpArgStr, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
			}
			tmpArgStr = strings.Replace(tmpArgStr, "~", homeDir, 1)
		}
		if _, err := os.Stat(tmpArgStr); err != nil {
			errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
		} else {
			if err = os.Chdir(tmpArgStr); err != nil {
				errOutput = fmt.Errorf("%s: %s: No such file or directory", cc.Cmd, argsStr)
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
		if _, ok := builtinCmd[argsStr]; ok {
			output = fmt.Sprintf("%s is a shell builtin", argsStr)
		} else {
			output = path.PrintLookPath(argsStr, path.LookPath(argsStr))
		}
	}

	if output != "" {
		_, errOutput = io.WriteString(cc.Stdout, output + "\n") // for echo -n do not work  
	}

	return errOutput
}
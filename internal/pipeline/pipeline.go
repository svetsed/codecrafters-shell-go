package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/codecrafters-io/shell-starter-go/internal/cmd"
)

type Cmds struct {
	Cmds	 	[]*cmd.CurrentCmd
	CountCmd 	int
	Wg 		 	sync.WaitGroup
}

func (c *Cmds) CreatePipeline() (readers []*os.File, writers []*os.File, err error) {
	readers = make([]*os.File, c.CountCmd)
	writers = make([]*os.File, c.CountCmd)

	for i:= 0; i < c.CountCmd - 1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			// close that already open
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

func(c *Cmds) SetupCmdPipe(i int, readers, writers []*os.File, devNull *os.File) {
	// if readers == nil || writers == nil || devNull == nil  || i < 0 {
	// 	return fmt.Errorf("received incorrect data")
	// }

	curCmd := c.Cmds[i]

	if readers[i] != nil {
		curCmd.Stdin = readers[i]
	}

	if curCmd.RedirectType == ">" || curCmd.RedirectType == ">>" || curCmd.RedirectType == "1>" || curCmd.RedirectType == "1>>" {
		// if redirect, writers don't need.
		writers[i].Close()
	} else if i < c.CountCmd - 1 {
		nextCmd := c.Cmds[i+1]
		if cmd.CheckIfBuiltinCmd(nextCmd.Cmd) {
			// builtins cmd don't read -> write in dev/null
			curCmd.Stdout = devNull
			writers[i].Close()
		} else {
			curCmd.Stdout = writers[i]
		}
	} else {
		// last cmd writes in stdout
		curCmd.Stdout = os.Stdout
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
					if cmd.CheckIfBuiltinCmd(c.Cmds[i].Cmd) {
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

	// ignore ExitError (exit status ...)
	for err := range errCh {
		var exitErr *exec.ExitError
		if !errors.As(err.err, &exitErr) {
			fmt.Fprintf(c.Cmds[err.i].Stderr, "%v\n", err.err)
		}
	}
}
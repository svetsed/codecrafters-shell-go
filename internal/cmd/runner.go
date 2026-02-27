package cmd

import (
	"fmt"
	"os/exec"

	"github.com/codecrafters-io/shell-starter-go/internal/utils/path"
)

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
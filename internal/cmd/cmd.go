package cmd

import (
	"io"
	"os"
)

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
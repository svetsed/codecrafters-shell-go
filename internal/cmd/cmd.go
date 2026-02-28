package cmd

import (
	"io"
	"os"
)

type CurrentCmd struct {
	Cmd			 string
	Args  		 []string
    Stdin  		 io.Reader
    Stdout 		 io.Writer
    Stderr 		 io.Writer
	Redirect
}

type Redirect struct {
	Files 		 []string   // save just filename
	RedirectType string		// > >> 1> 1>> 2> 2>>
	Flag		 int		// for openning file, example os.O_CREATE | os.O.RD_WR
	filesToClose []*os.File
}
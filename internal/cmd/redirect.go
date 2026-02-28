package cmd

import (
	"fmt"
	"os"
)

var availRedType = map[string]bool{
	">":   true,
	">>":  true,
	"1>":  true,
	"1>>": true,
	"2>":  true,
	"2>>": true,
}

// CorrectRedirectType checks the correctness of the specified redirect type.
func (cc *CurrentCmd) CorrectRedirectType() bool {
	if _, exist := availRedType[cc.RedirectType]; exist {
		return true
	}

	return false
}

// SetupRedirection configures where output should be redirected if a redirect was specified.
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
			// close that already open
			for _, opened := range files {
				opened.Close()
			}
			return fmt.Errorf("%v\n", err)
		}
		files = append(files, f)
	}

	// if several files were specified, then all are opened,
	// but the last one is written
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
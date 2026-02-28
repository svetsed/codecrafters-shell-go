package completer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/internal/utils/path"
)

type cmdCompleter struct {
	lastPrefix 		string
	lenPrefixInRune int
	matches	   		[]Match
	tab 			int	 	  // count tab
	builtins  		[]string
	externals   	[]string  // executable files founds in PATHs
	searchDir    	string	  // not "" when the user includes a path
	loadedExt		bool	  // flag for load externals just one in the session
	searchCmd		bool	  // flag to determine where exactly we will look
}

type Match struct {
	matchStr string
	isDir 	 bool    // if dir = / else " " for single matches
}

func NewCmdCompleter() *cmdCompleter {
	cc := &cmdCompleter{
		matches: []Match{},
		tab: 0,
		builtins: []string{"echo", "exit"},
		externals: []string{},
	}

	return cc
}

// ScanExternals searches for unique executable files from PATH and save in slice cc.externals.
func (cc *cmdCompleter) ScanExternals() {
	listDirs := path.GetListPath()
	if listDirs == nil {
		return
	}

	uniq := make(map[string]bool)
	for _, dir := range listDirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			info, err := file.Info()
			if err != nil {
				continue
			}

			fileStr := file.Name()
			fullPath := filepath.Join(dir, fileStr)

			if path.IsExecutable(fullPath, info) { // or? if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
				if _, exist := uniq[fileStr]; !exist {
					uniq[fileStr] = true
					cc.externals = append(cc.externals, fileStr)
				}
			}
		}
	}

	cc.loadedExt = true
}

// GetMatches searches for unique matches with the prefix.
func (cc *cmdCompleter) GetMatches() {
	uniqMatches := make(map[string]bool)
		
	// search cmd
	for _, cmd := range cc.builtins {
		if strings.HasPrefix(cmd, cc.lastPrefix) {
			if _, exist := uniqMatches[cmd]; !exist {
				uniqMatches[cmd] = true
				cc.matches = append(cc.matches, Match{matchStr: cmd})
			}

		}
	}
	for _, cmd := range cc.externals {
		if strings.HasPrefix(cmd, cc.lastPrefix) {
			if _, exist := uniqMatches[cmd]; !exist {
				uniqMatches[cmd] = true
				cc.matches = append(cc.matches, Match{matchStr: cmd})
			}
		}
	}
}
// LongestCommonPrefix searches in matches longest common prefix.
func (cc *cmdCompleter) LongestCommonPrefix() []rune {
	firstStr := []rune(cc.matches[0].matchStr)

	for i, ch := range firstStr {
		for _, str := range cc.matches[1:] {
			tmpStrInRune := []rune(str.matchStr) 
			if i >= len(tmpStrInRune) {
				return firstStr[:i]
			}
			if tmpStrInRune[i] != ch {
				return firstStr[:i]
			} 
		}
	}

	return firstStr
}

func (cc *cmdCompleter) SearchMatchInCurrentDir() {
	curDir, err := os.Getwd()
	if err != nil {
		return
	}

	if cc.searchDir != "" {
		curDir = filepath.Join(curDir, cc.searchDir)
	}

	dirEntry, err := os.ReadDir(curDir)
	if err != nil {
		return
	}

	uniqMatches := make(map[string]bool)
	for _, d := range dirEntry {
		dName := d.Name()
		if strings.HasPrefix(dName, cc.lastPrefix) {
			if _, exist := uniqMatches[dName]; !exist {
				uniqMatches[dName] = true
				tmp := Match {
					matchStr: dName,
					isDir: d.IsDir(),
				}
				cc.matches = append(cc.matches, tmp)
			}
		}
	}
}

func (cc *cmdCompleter) FindPrefix(lineStr string) string {
	// cmd1 a b | cmd2 a b | ...
	strForSearchSpace := lineStr
	if strings.Contains(lineStr, "|") {
		cmds := strings.Split(lineStr, "|")
		if len(cmds) >= 2 {
			strForSearchSpace = cmds[len(cmds)-1]
		}
	}

	var prefix string
	cc.searchCmd = true

	// cmd2 a b
	sliceLine := strings.Split(strForSearchSpace, " ")
	if len(sliceLine) == 0 { // no space -> getting line[:pos]
		prefix = lineStr
	} else if len(sliceLine) >= 2 {
		prefix = sliceLine[len(sliceLine)-1]
		cc.searchCmd = false
	} else if len(sliceLine) == 1 {
		prefix = sliceLine[0]
	}

	// a/b/c -> a/b/ - search dir; c - prefix
	if strings.Contains(prefix, string(os.PathSeparator)) {
		lastSep := strings.LastIndex(prefix, string(os.PathSeparator))
		if lastSep != -1 {
			cc.searchDir = prefix[:lastSep]
			prefix = prefix[lastSep+1:]
		}
	}

	return prefix
}

func (cc *cmdCompleter) MatchesJoin(sep string) string {
	if sep == "" {
		sep = "  "
	}

	buf := strings.Builder{}
	for _, match := range cc.matches {
		dir := ""
		if match.isDir {
			dir = "/"
		}
		buf.WriteString(match.matchStr + dir + sep)
	}
	return buf.String()
}

func (cc *cmdCompleter) SortMatches() {
	sort.Slice(cc.matches, func(i, j int) bool {
		return cc.matches[i].matchStr < cc.matches[j].matchStr
	})
}
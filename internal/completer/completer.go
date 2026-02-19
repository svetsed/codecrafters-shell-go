package completer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/internal/utils/path"
)

type cmdCompleter struct {
	lastPrefix string
	lenPrefixInRune int
	matches	   []string
	tab 		int
	builtins  	[]string
	externals   []string
	loadedExt	bool
}

func NewCmdCompleter() *cmdCompleter {
	cc := &cmdCompleter{
		matches: []string{},
		builtins: []string{"echo", "exit"},
		externals: []string{},
	}

	return cc
}

func (cc *cmdCompleter) scanExternals() {
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
			info, err := file.Info()
			if err != nil {
				continue
			}

			if file.IsDir() {
				continue
			}

			fileStr := file.Name()
			fullPath := filepath.Join(dir, fileStr)

			if path.IsExecutable(fullPath, info) { // if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
				if _, exist := uniq[fileStr]; !exist {
					uniq[fileStr] = true
					cc.externals = append(cc.externals, fileStr)
				}
			}
		}
	}

	cc.loadedExt = true
}

func (cc *cmdCompleter) GetMatches() {
	uniqMatches := make(map[string]bool)
	for _, cmd := range cc.builtins {
		if strings.HasPrefix(cmd, cc.lastPrefix) {
			if _, exist := uniqMatches[cmd]; !exist {
				uniqMatches[cmd] = true
				cc.matches = append(cc.matches, cmd)
			}

		}
	}

	for _, cmd := range cc.externals {
		if strings.HasPrefix(cmd, cc.lastPrefix) {
			if _, exist := uniqMatches[cmd]; !exist {
				uniqMatches[cmd] = true
				cc.matches = append(cc.matches, cmd)
			}
		}
	}
}

func (cc *cmdCompleter) LongestCommonPrefix() []rune {
	firstStr := []rune(cc.matches[0])

	for i, ch := range firstStr {
		for _, str := range cc.matches[1:] {
			tmpStrInRune := []rune(str) 
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

func (cc *cmdCompleter) Do(line []rune, pos int) ([][]rune, int) {
	lineStr := string(line[:pos])
	lastSpace := strings.LastIndex(string(line[:pos]), " ")
	var prefix string

	if lastSpace == -1 { // no space -> getting line[:pos]
		prefix = lineStr
	} else {
		prefix = string(line[lastSpace+1:pos])
	}

	if prefix == "" {
		fmt.Print("\x07")
		return nil, 0
	}

	if cc.lastPrefix == prefix && cc.tab == 1 {
		fmt.Printf("\n%s\n", strings.Join(cc.matches, "  "))
		fmt.Print("$ " + lineStr)
		return nil, 0
	}

	cc.tab = 0
	cc.lastPrefix = prefix
	cc.lenPrefixInRune = len([]rune(prefix))
	cc.matches = []string{}

	if !cc.loadedExt {
		cc.scanExternals()
	}

	cc.GetMatches()

	if len(cc.matches) == 0 {
		fmt.Print("\x07")
		return nil, 0 
	}

	if len(cc.matches) == 1 {
		ending := []rune(cc.matches[0][cc.lenPrefixInRune:])
		ending = append(ending, ' ')

		return [][]rune{ending}, cc.lenPrefixInRune
	}


	sort.Strings(cc.matches)
	commonPrefix := cc.LongestCommonPrefix()

	if len(commonPrefix) > cc.lenPrefixInRune {
		ending := commonPrefix[cc.lenPrefixInRune:]
		return [][]rune{ending}, cc.lenPrefixInRune
	} else {
		if cc.tab == 0 {
			fmt.Print("\x07")
			cc.tab = 1
		}
	}
	return nil, 0
}
package completer

import "fmt"

// Do implement AutoCompleter readline then user press TAB.
func (cc *cmdCompleter) Do(line []rune, pos int) ([][]rune, int) {
	lineStr := string(line[:pos])

	// search prefix
	prefix := cc.FindPrefix(lineStr)

	// too many option
	if prefix == "" && cc.searchCmd {
		fmt.Print("\x07")
		return nil, 0
	}

	if cc.lastPrefix == prefix && cc.tab == 1 {
		fmt.Printf("\n%s\n", cc.MatchesJoin("  "))
		fmt.Print("$ " + lineStr)
		return nil, 0
	}

	// refresh data for new prefix
	cc.tab = 0
	cc.lastPrefix = prefix
	cc.lenPrefixInRune = len([]rune(prefix))
	cc.matches = []Match{}

	if cc.searchCmd && !cc.loadedExt {
		cc.ScanExternals()
	}

	if !cc.searchCmd {
		cc.SearchMatchInCurrentDir()
	} else {
		cc.GetMatches() // search in externals and builtin
	}

	if len(cc.matches) == 0  {
		fmt.Print("\x07")
		return nil, 0 
	}

	// print ending, no full
	if len(cc.matches) == 1 {
		ending := []rune(cc.matches[0].matchStr[cc.lenPrefixInRune:])
		sign := ' '
		if cc.matches[0].isDir {
			sign = '/'
		}
		ending = append(ending, sign)

		return [][]rune{ending}, cc.lenPrefixInRune
	}

	cc.SortMatches()
	commonPrefix := cc.LongestCommonPrefix()

	// may print common prefix (ending again)
	if len(commonPrefix) > cc.lenPrefixInRune {
		ending := commonPrefix[cc.lenPrefixInRune:]
		return [][]rune{ending}, cc.lenPrefixInRune
	} else {
		// print matches to the next tab
		if cc.tab == 0 {
			fmt.Print("\x07")
			cc.tab = 1
		}
	}
	return nil, 0
}
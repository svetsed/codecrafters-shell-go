package commands

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/codecrafters-io/shell-starter-go/internal/history"
)

var History *history.History

func HandleHistoryCmd(args []string) (string, error) {
	if len(args) >= 1 {
		tmp, err := HistoryCmdWithArgs(args)
		if err != nil {
			return "", err
		}
			return tmp, nil
	} else {
		tmp := History.ReadHistoryWithFormat()
		return tmp, nil
	}
}

func HistoryCmdWithArgs(args []string) (string, error) {
	n, err := strconv.Atoi(args[0])
	if err != nil {
		if len(args) < 2 {
			return "", fmt.Errorf("incorrect input: missing file")
		}
		switch args[0] {
		case "-r":
			err := History.ReadHistoryFromFile(args[1])
			if err != nil {
				return "", err
			}
		case "-w":
			err := History.WriteHistoryToFile(args[1])
			if err != nil {
				if !errors.Is(err, history.HistoryIsEmpty) {
					return "", err
				} else {
					return "", nil
				}
			}
		case "-a":
			err := History.AppendHistoryToFile(args[1])
			if err != nil {
				if !errors.Is(err, history.HistoryIsEmpty) && !errors.Is(err, history.NoNewRecords) {
					return "", err
				} else {
					return "", nil
				}
			}
		default:
			return "", fmt.Errorf("unknown args")
		}
	} else {
		tmp, err := History.ReadHistoryLastNWithFormat(n)
		if err != nil {
			return "", err
		}
		
		return tmp, nil
	}

	return "", nil
}
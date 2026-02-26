package history

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/chzyer/readline"
)

var (
	HistoryIsEmpty = errors.New("history is empty")
	NoNewRecords   = errors.New("no new records")
)



type HistoryItem struct {
	Prev *HistoryItem
	Next *HistoryItem
	Line string
}

type History struct {
	Head 			*HistoryItem
	Tail 			*HistoryItem
	Counter 		int			  // total number of records (not clear)
	CountNewRecords int			  // cleared then written to file (history -a <>)
	Mu 				sync.RWMutex
	Walk
}

type Walk struct {
	Current *HistoryItem
}

func NewHistory() History {
	return History{}
}

func (h *History) PushFrontOneLine(line string) {
	if line == "" {
		return
	}

	newHead := &HistoryItem{
		Line: line,
	}

	h.Mu.Lock()
	defer h.Mu.Unlock()
	if h.Head == nil {
		h.Head = newHead
		h.Tail = newHead
	} else {
		newHead.Next = h.Head
		h.Head.Prev = newHead
		h.Head = newHead
	}
	h.Counter++
	h.CountNewRecords++
}

func (h *History) PushFront(lines string) {
	if lines == "" {
		return
	}

	sliceLines := strings.Split(lines, "\n")

	for _, line := range sliceLines {
		h.PushFrontOneLine(line)
	}
}

func (h *History) PushBackOneLine(line string) {
	if line == "" {
		return
	}

	newTail := &HistoryItem{
		Line: line,
	}

	h.Mu.Lock()
	defer h.Mu.Unlock()
	if h.Tail == nil {
		h.Head = newTail
		h.Tail = newTail
	} else {
		newTail.Prev = h.Tail
		h.Tail.Next = newTail
		h.Tail = newTail
	}

	h.Counter++
	h.CountNewRecords++
	h.Walk.Current = nil
}

func (h *History) PushBack(lines string) {
	if lines == "" {
		return
	}

	sliceLines := strings.Split(lines, "\n")

	for _, line := range sliceLines {
		h.PushBackOneLine(line)
	}
}

func (h *History) Front() (string, bool) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	if h.Head == nil {
		return "", false
	}

	return h.Head.Line, true
}

func (h *History) Back() (string, bool) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	if h.Tail == nil {
		return "", false
	}

	return h.Tail.Line, true
}

// ReadFromHead returns records from history in slice.
// May return nil if history is empty. Don't forget to check.
func (h *History) ReadFromHead() []string {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	if h.Head == nil {
		return nil
	}

	sliceLines := make([]string, 0, 1)

	if h.Head.Next == nil {
		sliceLines = append(sliceLines, h.Head.Line)
	}


	current := h.Head
	for current != nil {
		sliceLines = append(sliceLines, current.Line)
		current = current.Next
	}

	return sliceLines
}

// ReadHistoryWithFormat is a wrapper for output to return all entries in history of the format.
func (h *History) ReadHistoryWithFormat() string {
	sliceLines := h.ReadFromHead()
	if sliceLines == nil {
		return ""
	}

	return PrintHistoryWithFormatASC(sliceLines, 1)
}

// PrintHistoryWithFormatASC returns records in the format(without quotes): "    1  echo hello\n".
func PrintHistoryWithFormatASC(sliceLines []string, i int) string {
	buf := strings.Builder{}
	for _, line := range sliceLines {
		buf.WriteString(fmt.Sprintf("    %d  %s\n", i, line))
		i++
	}

	return strings.TrimRight(buf.String(), "\n\r\t")
}

// ReadHistoryLastNWithFormat is a wrapper for output to return Last N entries in history of the format.
func (h *History) ReadHistoryLastNWithFormat(n int) (string, error) {
	sliceLines, err := h.ReadFromTailLastN(n)
	if err != nil {
		return "", err
	}
	if sliceLines == nil {
		return "", err
	}

	i:= h.Counter - n + 1
	return PrintHistoryWithFormatASC(sliceLines, i), nil
} 

// ReadFromTailLastN returns last n records from history in slice.
// If N is greater than the total number of records, it will be called ReadFromHead.
// May return nil if history is empty. Don't forget to check.
func (h *History) ReadFromTailLastN(n int) ([]string, error) {
	h.Mu.RLock()
	if h.Tail == nil {
		h.Mu.RUnlock()
		return nil, nil
	}

	if n >= h.Counter  {
		h.Mu.RUnlock()
		return h.ReadFromHead(), nil
	}

	defer h.Mu.RUnlock()

	if n < 0 {
		return nil, fmt.Errorf("invalid n")
	}

	if n == 0 {
		return nil, nil
	}

	sliceLines := make([]string, 0, 1)

	if h.Tail.Prev == nil {
		sliceLines = append(sliceLines, h.Tail.Line)
		return sliceLines, nil
	}

	current := h.Tail
	searchingElem := n - 1

	for current != nil && searchingElem != 0 {
		current = current.Prev
		searchingElem--
	}

	if searchingElem != 0 {
		return nil,  fmt.Errorf("element don't found")
	}

	for current != nil {
		sliceLines = append(sliceLines, current.Line)
		current = current.Next
	}

	return sliceLines, nil
}

func (h *History) ReadFromTailWithFormat() string {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	if h.Tail == nil {
		return ""
	}

	if h.Tail.Prev == nil {
		return fmt.Sprintf("    %d  %s", h.Counter, h.Tail.Line)
	}

	buf := strings.Builder{}
	current := h.Tail
	i := h.Counter

	for current != nil {
		buf.WriteString(fmt.Sprintf("    %d  %s\n", i, current.Line))
		current = current.Prev
		i--
	}

	return strings.TrimRight(buf.String(), "\n\r\t")
}

func (h *History) WalkByHistory(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
	// fmt.Printf("key: %q, inESC: %v, buf: %q\n", key, h.Walk.InESC, string(h.Walk.Buf))
	switch key {
	case readline.CharPrev: // 16 \x10
		return h.handleUp()
	case readline.CharNext: // 14 \x0e
		return h.handleDown()
	default:
		return nil, 0, false
	}
}

func (h *History) handleUp() (newLine []rune, newPos int, ok bool) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if h.Current == nil {
		h.Current = h.Tail
		if h.Current != nil {
			line := []rune(h.Current.Line)
			return line, len(line), true
		}
	}

	if h.Current != nil {
		if h.Current.Prev != nil {
			h.Current = h.Current.Prev
			line := []rune(h.Current.Line)
			return line, len(line), true
		}
	}

	return nil, 0, false
}

func (h *History) handleDown() (newLine []rune, newPos int, ok bool) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	if h.Current == nil {
		h.Current = h.Tail
	}

	if h.Current != nil {
		if h.Current.Next != nil {
			h.Current = h.Current.Next
			line := []rune(h.Current.Line)
			return line, len(line), true
		} else {
			h.Current = nil
			return []rune(""), 0, true
		}
	}

	return nil, 0, false
}

func(h *History) ClearCountNewRecords() {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	h.CountNewRecords = 0
}

func(h *History) CheckCountNewRecords() int {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	return h.CountNewRecords
}

// ReadHistoryFromFile reads history from file and append in the end of history.
func (h *History) ReadHistoryFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	h.PushBack(string(data))

	return nil
}

// WriteHistoryToFIle writes history to file, overwriting its contents, if it was not empty.
// Creates a file, if it does not exist.
// Returns the error HistoryIsEmpty, if history is empty.
func (h *History) WriteHistoryToFile(filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE | os.O_RDWR, 0766)
	if err != nil {
		return fmt.Errorf("could not open file")
	}
	defer f.Close()

	sliceLines := h.ReadFromHead()
	if sliceLines == nil {
		return HistoryIsEmpty
	}

	for _, line := range sliceLines {
		f.WriteString(line + "\n")
	}

	return nil
}

// AppendHistoryToFile adds new records to the end of the file.
// Creates a file, if it does not exist.
// Returns the error NoNewRecords, if there are no new records.
// Returns the error HistoryIsEmpty, if history is empty.
func (h *History) AppendHistoryToFile(filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE | os.O_APPEND | os.O_RDWR, 0766)
	if err != nil {
		return fmt.Errorf("could not open file")
	}
	defer f.Close()

	count := h.CheckCountNewRecords()

	if count == 0 {
		return NoNewRecords
	}
	
	sliceLines, err := h.ReadFromTailLastN(count)
	if err != nil {
		return err
	}

	if sliceLines == nil {
		return HistoryIsEmpty
	}

	for _, line := range sliceLines {
		f.WriteString(line + "\n")
	}

	h.ClearCountNewRecords()

	return nil
}

// Todo
// // delete First
// // delete Last
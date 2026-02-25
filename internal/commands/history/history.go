package history

import (
	"fmt"
	"strings"
	"sync"

	"github.com/chzyer/readline"
)

type HistoryItem struct {
	Prev *HistoryItem
	Next *HistoryItem
	Line string
}

type History struct {
	Head 	*HistoryItem
	Tail 	*HistoryItem
	Counter int
	Mu 		sync.RWMutex
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

func (h *History) ReadFromHeadWithFormat() string {
	sliceLines := h.ReadFromHead()
	if sliceLines == nil {
		return ""
	}

	return h.PrintFromHeadWithFormat(sliceLines)
}

func (h *History) PrintFromHeadWithFormat(sliceLines []string) string {
	buf := strings.Builder{}
	for i, line := range sliceLines {
		i++
		buf.WriteString(fmt.Sprintf("    %d  %s\n", i, line))
	}

	return strings.TrimRight(buf.String(), "\n\r\t")
}

func (h *History) ReadFromTail() string {
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


func (h *History) ReadFromTailLastN(n int) (string, error) {
	h.Mu.RLock()
	if h.Tail == nil {
		h.Mu.RUnlock()
		return "", nil
	}

	if n >= h.Counter  {
		h.Mu.RUnlock()
		return h.ReadFromHeadWithFormat(), nil
	}

	defer h.Mu.RUnlock()

	if n < 0 {
		return "", fmt.Errorf("invalid option")
	}

	if n == 0 {
		return "", nil
	}

	if h.Tail.Prev == nil {
		return fmt.Sprintf("    %d  %s\n", h.Counter, h.Tail.Line), nil
	}

	current := h.Tail
	searchingElem := n - 1

	for current != nil && searchingElem != 0 {
		current = current.Prev
		searchingElem--
	}

	if searchingElem != 0 {
		return "",  fmt.Errorf("element don't found")
	}

	buf := strings.Builder{}
	i := h.Counter - n + 1
	for current != nil {
		buf.WriteString(fmt.Sprintf("    %d  %s\n", i, current.Line))
		current = current.Next
		i++
	}

	return strings.TrimRight(buf.String(), "\n\r\t"), nil
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

// Todo
// // delete First
// // delete Last
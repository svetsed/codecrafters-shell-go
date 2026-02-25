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
	MuWalk	sync.RWMutex
	Current *HistoryItem
	InESC   bool
	Buf     []rune
}

func NewHistory() History {
	return History{
		Walk: Walk {
			Buf: make([]rune, 0),
		},
	}
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

	h.Walk.MuWalk.Lock()
	h.Walk.Current = h.Tail
	h.Walk.MuWalk.Unlock()
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

func (h *History) ReadFromHead() string {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	if h.Head == nil {
		return ""
	}

	if h.Head.Next == nil {
		return fmt.Sprintf("    %d  %s", 1, h.Head.Line)
	}

	buf := strings.Builder{}
	current := h.Head
	i := 0
	for current != nil {
		i++
		buf.WriteString(fmt.Sprintf("    %d  %s\n", i, current.Line))
		current = current.Next
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
		return h.ReadFromHead(), nil
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
	
	
	// if h.Walk.InESC {
	// 	h.Walk.Buf = append(h.Walk.Buf, key)

	// 	if len(h.Walk.Buf) == 3  && h.Walk.Buf[0]== readline.CharEsc && h.Walk.Buf[1] == readline.CharEscapeEx {
	// 		h.Walk.InESC = false
	// 		lastCh := h.Walk.Buf[2]
	// 		h.Walk.Buf = make([]rune, 0)
	// 		switch lastCh {
	// 		case 'A':
	// 			return h.handleUp()
	// 		case 'B':
	// 			return h.handleDown()
	// 		default:
	// 			return nil, 0, false
	// 		}
	// 	}

	// 	if len(h.Walk.Buf) > 3 || (len(h.Walk.Buf) == 2 && h.Walk.Buf[1] != readline.CharEscapeEx) {
	// 		h.Walk.InESC = false
	// 		h.Walk.Buf = make([]rune, 0)
	// 		return nil, 0, false
	// 	}

	// 	return line, pos, true
	// }
	
	// if key == readline.CharEsc && !h.Walk.InESC {
	// 	h.Walk.InESC = true
	// 	h.Walk.Buf = append(h.Walk.Buf, key)
	// 	return line, pos, true
	// }

	// return nil, 0, false
}

func (h *History) handleUp() (newLine []rune, newPos int, ok bool) {
	h.Walk.MuWalk.Lock()
	defer h.Walk.MuWalk.Unlock()
	if h.Current != nil {
		line := []rune(h.Current.Line)
		if h.Current.Prev != nil {
			h.Current = h.Current.Prev
		}
		return line, len(line), true
	}

	return nil, 0, false
}

func (h *History) handleDown() (newLine []rune, newPos int, ok bool) {
	h.Walk.MuWalk.Lock()
	defer h.Walk.MuWalk.Unlock()
	if h.Current != nil {
		if h.Current.Next != nil {
			h.Current = h.Current.Next
			line := []rune(h.Current.Line)
			return line, len(line), true
		} else {
			return []rune(""), 0, true
		}
	}

	return nil, 0, false
}



// Todo
// // delete First
// // delete Last


// type History struct {
// 	HistoryPath string
// 	Mu 			sync.RWMutex
// 	File		*os.File
// 	CounterLine int 
// }

// func New(path string) (*History, error) {

// 	hp := History{}
// 	if path == "" {
// 		tmpFile, err := os.CreateTemp("", "history-*.tmp")
// 		if err != nil {
// 			return nil, fmt.Errorf("error creating temp file: %v", err)
// 		}
// 		hp.HistoryPath = tmpFile.Name()
// 		hp.File = tmpFile
// 	} else {
// 		hp.HistoryPath = path

// 		f, err := os.OpenFile(hp.HistoryPath, os.O_CREATE | os.O_APPEND | os.O_RDWR, 0766)
// 		if err != nil {
// 			return nil, fmt.Errorf("error opening history file: %s: %v", hp.HistoryPath, err)
// 		}

// 		hp.File = f
// 	}

// 	return &hp, nil
// }

// func (h *History) CloseHistory() error {
// 	h.Mu.Lock()
// 	defer h.Mu.Unlock()
// 	if h.File != nil {
// 		return h.File.Close()
// 	}
// 	return nil
// }

// func (h *History) ReadHistoryAndCut(n int) (string, error) {
// 	if n < 0 {
// 		return "", fmt.Errorf("invalid number")
// 	}

// 	fullStr, err := h.ReadHistory()
// 	if err != nil {
// 		return "", err
// 	}

// 	sliceStr := strings.Split(fullStr, "\n")
// 	if len(sliceStr) == 0 {
// 		return "", nil
// 	}

// 	total := len(sliceStr)
// 	i := 0
// 	if n >= total {
// 		i = 0
// 	} else {
// 		i = total - n
// 	}

// 	buf := strings.Builder{}
// 	for ; i < total; i++ {
// 		buf.WriteString(fmt.Sprintf("    %d  %s\n", i+1, sliceStr[i]))
// 	}

// 	output := strings.TrimRight(buf.String(), "\n\r\t")

// 	return output, nil
// }


// func (h *History) ReadHistory() (string, error) {
// 	h.Mu.RLock()
// 	defer h.Mu.RUnlock()

// 	if h.File == nil {
// 		return "", fmt.Errorf("error reading history file: file don't exist")
// 	}


// 	data, err := os.ReadFile(h.HistoryPath)
// 	if err != nil {
// 		return "", fmt.Errorf("error reading history file: %v", err)
// 	}

// 	// fileInfo, err := h.File.Stat()
// 	// if err != nil {
//     //     return "", fmt.Errorf("error getting file info: %v", err)
//     // }

// 	// buffer := make([]byte, fileInfo.Size())

// 	// n, err := h.File.ReadAt(buffer, 0)
// 	// if err != nil && err != io.EOF {
// 	// 	return "", fmt.Errorf("error reading file: %v", err)
// 	// }

// 	content := string(data)

// 	content = strings.TrimRight(content, "\n\r\t")

// 	return content, nil	
// }

// func (h *History) ReadHistoryWithFormat() (string, error) {
// 	fullStr, err := h.ReadHistory()
// 	if err != nil {
// 		return "", err
// 	}

// 	sliceStr := strings.Split(fullStr, "\n")
// 	if len(sliceStr) == 0 {
// 		return "", nil
// 	}

// 	buf := strings.Builder{}
// 	for i:= 0; i < len(sliceStr); i++ {
// 		buf.WriteString(fmt.Sprintf("    %d  %s\n", i+1, sliceStr[i]))
// 	}

// 	output := strings.TrimRight(buf.String(), "\n\r\t")

// 	return output, nil
// }

// func(h *History) AddInHistory(pathToFile string) error {
// 	if pathToFile == "" {
// 		return nil
// 	}

// 	data, err := os.ReadFile(pathToFile)
// 	if err != nil {
// 		return fmt.Errorf("error reading file: %v", err)
// 	}

// 	if string(data) == "" {
// 		return nil
// 	}

// 	h.Mu.Lock()
// 	defer h.Mu.Unlock()

// 	_, err = h.File.Write(data)
// 	if err != nil {
// 		return fmt.Errorf("error writing data in history file: %v", err)
// 	}

// 	// h.CounterLine++л
// 	return nil
// }

// func(h *History) SaveHistoryWithFormat(line string) error {
// 	h.Mu.Lock()
// 	defer h.Mu.Unlock()

// 	if line == "" {
// 		return nil
// 	}

// 	if h.File == nil {
// 		return fmt.Errorf("error writing line in history file: file don't exist")
// 	}

// 	h.CounterLine++
// 	newNistoryLine := fmt.Sprintf("    %d  %s\n", h.CounterLine, line)

// 	_, err := h.File.WriteString(newNistoryLine)
// 	if err != nil {
// 		return fmt.Errorf("error writing line in history file: %v", err)
// 	}

// 	return nil
// }

// func (h *History) ClearHistory() error {
// 	h.Mu.Lock()
// 	defer h.Mu.Unlock()

// 	if h.File == nil {
// 		return fmt.Errorf("error clearing history: file doesn't exist")
// 	}

// 	if err := h.File.Close(); err != nil {
// 		return fmt.Errorf("error closing history file: %v", err)
// 	}

// 	if err := os.Remove(h.HistoryPath); err != nil {
// 		return fmt.Errorf("error removing history file: %v", err)
// 	}

// 	f, err := os.OpenFile(h.HistoryPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0766)
// 	    if err != nil {
//         return fmt.Errorf("error creating new history file: %v", err)
//     }

// 	h.File = f
// 	h.CounterLine = 0

// 	return nil
// }


// func (h *History) RemoveHistory() error {
// 	h.Mu.Lock()
// 	defer h.Mu.Unlock()

// 	if h.File == nil {
// 		return fmt.Errorf("error clearing history: file doesn't exist")
// 	}

// 	if err := h.File.Close(); err != nil {
// 		return fmt.Errorf("error closing history file: %v", err)
// 	}

// 	if err := os.Remove(h.HistoryPath); err != nil {
// 		return fmt.Errorf("error removing history file: %v", err)
// 	}

// 	return nil
// }

package history

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type History struct {
	HistoryPath string
	Mu 			sync.RWMutex
	File		*os.File
	CounterLine int 
}

func New(path string) (*History, error) {
	hp := History{}
	if path == "" {
		hp.HistoryPath = "./history.tmp"
	} else {
		hp.HistoryPath = path
	}

	f, err := os.OpenFile(hp.HistoryPath, os.O_CREATE | os.O_APPEND | os.O_RDWR, 0766)
	if err != nil {
		return nil, fmt.Errorf("error opening history file: %s: %v", hp.HistoryPath, err)
	}

	hp.File = f

	return &hp, nil
}

func (h *History) CloseHistory() error {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	if h.File != nil {
		return h.File.Close()
	}
	return nil
}

func (h *History) ReadHistoryAndCut(n int) (string, error) {
	if n < 0 {
		return "", fmt.Errorf("invalid number")
	}

	fullStr, err := h.ReadHistory()
	if err != nil {
		return "", err
	}

	sliceStr := strings.Split(fullStr, "\n")
	if len(sliceStr) == 0 {
		return "", nil
	}

	total := len(sliceStr)
	i := 0
	if n >= total {
		i = 0
	} else {
		i = total - n
	}

	buf := strings.Builder{}
	for ; i < total; i++ {
		buf.WriteString(fmt.Sprintf("    %d  %s\n", i+1, sliceStr[i]))
	}

	output := strings.TrimRight(buf.String(), "\n\r\t")

	return output, nil
}


func (h *History) ReadHistory() (string, error) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	if h.File == nil {
		return "", fmt.Errorf("error reading history file: file don't exist")
	}

	fileInfo, err := h.File.Stat()
	if err != nil {
        return "", fmt.Errorf("error getting file info: %v", err)
    }

	buffer := make([]byte, fileInfo.Size())

	n, err := h.File.ReadAt(buffer, 0)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	content := string(buffer[:n])

	content = strings.TrimRight(content, "\n\r\t")

	return content, nil	
}

func (h *History) ReadHistoryWithFormat() (string, error) {
	fullStr, err := h.ReadHistory()
	if err != nil {
		return "", err
	}

	sliceStr := strings.Split(fullStr, "\n")
	if len(sliceStr) == 0 {
		return "", nil
	}

	buf := strings.Builder{}
	for i:= 0; i < len(sliceStr); i++ {
		buf.WriteString(fmt.Sprintf("    %d  %s\n", i+1, sliceStr[i]))
	}

	output := strings.TrimRight(buf.String(), "\n\r\t")

	return output, nil
}

func(h *History) SaveHistoryWithFormat(line string) error {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if line == "" {
		return nil
	}

	if h.File == nil {
		return fmt.Errorf("error writing line in history file: file don't exist")
	}

	h.CounterLine++
	newNistoryLine := fmt.Sprintf("    %d  %s\n", h.CounterLine, line)

	_, err := h.File.WriteString(newNistoryLine)
	if err != nil {
		return fmt.Errorf("error writing line in history file: %v", err)
	}

	return nil
}

// TODO Clear or Reset History

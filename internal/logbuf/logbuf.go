// Package logbuf keeps the last of what magpie has said, so the Logs view
// can show it and a bug can be read where it happened. Every line also
// goes to a file beside the settings, for a bug that ends the app.
package logbuf

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// The levels a line carries.
const (
	Info  = "INFO"
	Warn  = "WARN"
	Error = "ERROR"
)

// Line is one thing magpie said.
type Line struct {
	Seq   int64     `json:"seq"`
	At    time.Time `json:"at"`
	Level string    `json:"level"`
	Msg   string    `json:"msg"`
}

// keep is how many lines are held for the view: days of a busy agent's
// calls, at a few hundred bytes each.
const keep = 2000

var (
	mu    sync.Mutex
	lines []Line
	seq   int64
	file  *os.File
	dir   string
)

// Init opens the log file under d (a settings folder); the view still
// works when it can't be opened, just without the file to hand over.
func Init(d string) {
	dir = d
	if d == "" {
		return
	}
	if err := os.MkdirAll(filepath.Join(d, "logs"), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(d, "logs", "magpie.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	file = f
}

// Path is the log file, "" when there is none.
func Path() string {
	if file == nil {
		return ""
	}
	return filepath.Join(dir, "logs", "magpie.log")
}

// Add writes one line at a level.
func Add(level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	mu.Lock()
	defer mu.Unlock()
	seq++
	ln := Line{Seq: seq, At: time.Now(), Level: level, Msg: msg}
	lines = append(lines, ln)
	if len(lines) > keep {
		lines = lines[len(lines)-keep:]
	}
	if file != nil {
		_, _ = fmt.Fprintf(file, "%s %-5s %s\n", ln.At.Format("2006-01-02 15:04:05.000"), level, msg)
	}
}

// Infof, Warnf and Errorf write one line at that level.
func Infof(format string, args ...any)  { Add(Info, format, args...) }
func Warnf(format string, args ...any)  { Add(Warn, format, args...) }
func Errorf(format string, args ...any) { Add(Error, format, args...) }

// Since gives the lines newer than a seq, oldest first.
func Since(after int64) []Line {
	mu.Lock()
	defer mu.Unlock()
	out := []Line{}
	for _, l := range lines {
		if l.Seq > after {
			out = append(out, l)
		}
	}
	return out
}

// All is every line held.
func All() []Line { return Since(0) }

// Last is the newest seq, 0 when nothing has been said.
func Last() int64 {
	mu.Lock()
	defer mu.Unlock()
	return seq
}

// Clear drops the lines held; the file is kept, as it is the one thing
// left of a bug that has already happened.
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	lines = nil
}

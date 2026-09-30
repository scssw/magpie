package gui

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/yetone/magpie/internal/logbuf"
)

// errNoLogFile is the answer when magpie has nowhere to keep one.
var errNoLogFile = errors.New("no log file: magpie could not open one beside the settings")

// logRoutes serves the Logs view: what magpie has said lately, with the
// file it also keeps, so a bug can be read where it happened.
func logRoutes(mux *http.ServeMux, w Windows) {
	mux.HandleFunc("GET /api/logs", func(rw http.ResponseWriter, r *http.Request) {
		after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		lines := logbuf.All()
		if after > 0 {
			lines = logbuf.Since(after)
		}
		if level := strings.ToUpper(r.URL.Query().Get("level")); level != "" && level != "ALL" {
			kept := make([]logbuf.Line, 0, len(lines))
			for _, l := range lines {
				if l.Level == level {
					kept = append(kept, l)
				}
			}
			lines = kept
		}
		if lines == nil {
			lines = []logbuf.Line{}
		}
		writeJSON(rw, map[string]any{
			"lines": lines,
			"last":  logbuf.Last(),
			"path":  logbuf.Path(),
		})
	})
	// one line as text, for the clipboard: the file's own shape, so what is
	// pasted into an issue is what the file says
	mux.HandleFunc("POST /api/logs/copy", func(rw http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		for _, l := range logbuf.All() {
			b.WriteString(l.At.Format("2006-01-02 15:04:05.000"))
			b.WriteString(" ")
			b.WriteString(l.Level)
			b.WriteString(" ")
			b.WriteString(l.Msg)
			b.WriteString("\n")
		}
		writeJSON(rw, map[string]bool{"ok": w.Copy(b.String())})
	})
	mux.HandleFunc("POST /api/logs/clear", func(rw http.ResponseWriter, r *http.Request) {
		logbuf.Clear()
		writeJSON(rw, map[string]bool{"ok": true})
	})
	// the file itself, in the system's file manager: it survives a restart,
	// unlike what the view holds
	mux.HandleFunc("POST /api/logs/reveal", func(rw http.ResponseWriter, r *http.Request) {
		p := logbuf.Path()
		if p == "" {
			fail(rw, errNoLogFile)
			return
		}
		if err := w.OpenFolder(p); err != nil {
			fail(rw, err)
			return
		}
		rw.WriteHeader(http.StatusNoContent)
	})
}

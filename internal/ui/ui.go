// Package ui gives the FYRwall CLI an informative, visual experience:
// colored status glyphs, box-drawing tables, section banners, progress
// spinners and pager integration for long output (spec section 47 UX
// standards applied to the terminal).
package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ANSI helpers. Colors auto-disable when not attached to a TTY or when
// NO_COLOR is set (accessibility and CI friendliness).
var (
	useColor = detectColor()

	Bold  = colorize("\033[1m")
	Dim   = colorize("\033[2m")
	Red   = colorize("\033[31m")
	Green = colorize("\033[32m")
	Amber = colorize("\033[33m")
	Blue  = colorize("\033[34m")
	Cyan  = colorize("\033[36m")
	Reset = colorize("\033[0m")
)

func detectColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if fi, err := os.Stdout.Stat(); err == nil {
		return fi.Mode()&os.ModeCharDevice != 0
	}
	return false
}

func colorize(code string) string {
	if !useColor {
		return ""
	}
	return code
}

// PrintStatusLine renders one check result line with a color glyph.
func PrintStatusLine(w io.Writer, status, code, message string) {
	glyph := GlyphPass
	color := Green
	switch status {
	case "WARN":
		glyph, color = GlyphWarn, Amber
	case "FAIL":
		glyph, color = GlyphFail, Red
	case "SKIP":
		glyph, color = GlyphSkip, Dim
	}
	fmt.Fprintf(w, " %s%-4s%s %-16s %s\n", color, glyph, Reset, Bold+code+Reset, message)
}

// Status glyphs for check results.
const (
	GlyphPass = "OK  "
	GlyphWarn = "WARN"
	GlyphFail = "FAIL"
	GlyphSkip = "SKIP"
)

// Banner prints a section header.
func Banner(w io.Writer, title string) {
	line := strings.Repeat("-", len(title)+4)
	fmt.Fprintf(w, "\n%s+%s+\n", Dim, Reset)
	fmt.Fprintf(w, "%s| %s%s |\n", Dim, title, Reset)
	fmt.Fprintf(w, "%s+%s+\n", Dim, Reset)
	_ = line
}

// KV prints an aligned key/value pair.
func KV(w io.Writer, key, value string) {
	fmt.Fprintf(w, "  %s%-22s%s %s\n", Dim, key, Reset, value)
}

// Table renders a simple bordered table; widths derive from content.
func Table(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = visibleLen(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && visibleLen(cell) > widths[i] {
				widths[i] = visibleLen(cell)
			}
		}
	}
	sep := "+"
	for _, wd := range widths {
		sep += strings.Repeat("-", wd+2) + "+"
	}
	fmt.Fprintln(w, sep)
	fmt.Fprint(w, "|")
	for i, h := range headers {
		fmt.Fprintf(w, " %-*s |", widths[i]+len(h)-visibleLen(h), h)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, sep)
	for _, row := range rows {
		fmt.Fprint(w, "|")
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			fmt.Fprintf(w, " %-*s |", widths[i]+len(cell)-visibleLen(cell), cell)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, sep)
}

// visibleLen ignores ANSI escape sequences when measuring.
func visibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}

// Pager pipes content through $PAGER (less -R) when the output is long
// and stdout is a TTY; otherwise it prints directly. Long output (rule
// lists, logs) stays readable without flooding the terminal.
func Pager(content string) {
	if !isTTY() || len(content) < 4000 {
		fmt.Print(content)
		return
	}
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -R"
	}
	cmd := exec.Command("sh", "-c", pager)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Print(content) // pager missing: fall back to direct print
	}
}

// Spinner renders an animated progress indicator for long operations
// (update apply, extension install).
type Spinner struct {
	stop chan struct{}
	done chan struct{}
}

// StartSpinner begins a background spinner with the given label.
func StartSpinner(label string) *Spinner {
	if !isTTY() {
		fmt.Println(label + "...")
		return &Spinner{stop: make(chan struct{}), done: make(chan struct{})}
	}
	s := &Spinner{stop: make(chan struct{}), done: make(chan struct{})}
	frames := []string{"|", "/", "-", "\\"}
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Printf("\r  %s %s%s\n", Green+"done"+Reset, label, strings.Repeat(" ", 10))
				return
			default:
				fmt.Printf("\r  %s %s", Cyan+frames[i%len(frames)]+Reset, label)
				i++
				time.Sleep(120 * time.Millisecond)
			}
		}
	}()
	return s
}

// Stop ends the spinner.
func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

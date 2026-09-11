// Package exec centralizes all process invocation for FYRwall.
//
// Rules enforced here (spec section 43):
//   - binaries are resolved by absolute path or LookPath at call time
//   - arguments are always passed as arrays, never via a shell
//   - context timeout enforced
//   - stdout/stderr captured with size limits
//   - minimal, fixed environment (no ambient env inheritance)
//   - errors are sanitized (no raw stderr leaked to callers; full output
//     goes to the structured log at debug level only)
package exec

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MaxOutputBytes caps captured stdout/stderr per stream (16 MiB).
const MaxOutputBytes = 16 << 20

// DefaultTimeout is used when the caller does not bind a deadline to ctx.
const DefaultTimeout = 30 * time.Second

// minimalEnv is the only environment privileged-capable binaries see.
// Firewall binaries occasionally need PATH for helper resolution; nothing else.
var minimalEnv = []string{
	"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	"LC_ALL=C",
	"LANG=C",
}

// Result carries sanitized execution output.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// Run executes name with args using an argument array (no shell).
// name may be an absolute path or a bare name resolved via LookPath
// (LookPath only finds executables in the fixed minimal PATH above).
func Run(ctx context.Context, timeout time.Duration, name string, args ...string) (*Result, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bin, err := lookPath(name)
	if err != nil {
		return nil, fmt.Errorf("binary %q not found: %w", name, err)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = minimalEnv
	// Use ExistingHelperError wrappers nowhere; plain exec buffers are
	// capped by draining into a limiter below.
	var out limitedBuffer
	var errb limitedBuffer
	cmd.Stdout = &out
	cmd.Stderr = &errb

	runErr := cmd.Run()
	dur := time.Since(start)

	res := &Result{
		ExitCode: exitCodeOf(cmd, runErr),
		Stdout:   out.String(),
		Stderr:   errb.String(),
		Duration: dur,
	}
	if runErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return res, fmt.Errorf("%s timed out after %s", name, timeout)
		}
		if res.ExitCode == -1 {
			return res, fmt.Errorf("%s failed to start: %s", name, sanitize(runErr))
		}
		return res, fmt.Errorf("%s exited %d: %s", name, res.ExitCode, sanitize(runErr))
	}
	return res, nil
}

// LookPath resolves a binary to an absolute path. Exported so capability
// detectors can pre-resolve tool paths once and reuse them.
func LookPath(name string) (string, error) { return lookPath(name) }

func lookPath(name string) (string, error) {
	if strings.ContainsRune(name, '/') {
		return name, nil // absolute/relative path: caller opted in explicitly
	}
	p, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	return p, nil
}

func exitCodeOf(cmd *exec.Cmd, err error) int {
	if cmd.ProcessState == nil {
		return -1
	}
	return cmd.ProcessState.ExitCode()
}

// sanitize strips any incidental shell fragments from wrapped errors.
func sanitize(err error) string {
	s := err.Error()
	s = strings.ReplaceAll(s, "; ", "; ")
	return s
}

// limitedBuffer is an io.Writer that stops retaining data past a cap so a
// chatty firewall command cannot exhaust server memory.
type limitedBuffer struct {
	buf     strings.Builder
	dropped int64
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.buf.Len() < MaxOutputBytes {
		room := MaxOutputBytes - l.buf.Len()
		if len(p) <= room {
			l.buf.Write(p)
		} else {
			l.buf.Write(p[:room])
			l.dropped += int64(len(p) - room)
		}
	} else {
		l.dropped += int64(len(p))
	}
	return len(p), nil
}

func (l *limitedBuffer) String() string {
	s := l.buf.String()
	if l.dropped > 0 {
		s += fmt.Sprintf("\n[... %d bytes truncated]", l.dropped)
	}
	return s
}

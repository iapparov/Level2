package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()

	_ = w.Close()
	os.Stdout = old
	s := <-outC
	return s
}

func TestIsBuiltIn(t *testing.T) {
	if !isBuiltIn("cd") {
		t.Fatal("cd should be builtin")
	}
	if isBuiltIn("ls") {
		t.Fatal("ls should not be builtin")
	}
}

func TestSplitLogical(t *testing.T) {
	input := "false || echo a && echo b"
	got := splitLogical(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 logical cmds, got %d", len(got))
	}
	if got[0].op != "" || got[1].op != "||" || got[2].op != "&&" {
		t.Fatalf("unexpected ops: %#v", got)
	}
	if got[1].cmd != "echo a" || got[2].cmd != "echo b" {
		t.Fatalf("unexpected cmds: %#v", got)
	}
}

func TestParseRedirections_CreateOutFile(t *testing.T) {
	td := t.TempDir()
	fn := filepath.Join(td, "out.txt")
	args := []string{"echo", "hi", ">", fn}
	clean, in, out, err := parseRedirections(args)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(clean) != 2 || clean[0] != "echo" || clean[1] != "hi" {
		t.Fatalf("clean args wrong: %v", clean)
	}
	f, ok := out.(*os.File)
	if !ok {
		t.Fatalf("expected out to be *os.File")
	}
	_, _ = f.WriteString("hello\n")
	_ = f.Close()
	b, err := os.ReadFile(fn)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if strings.TrimSpace(string(b)) != "hello" {
		t.Fatalf("file content mismatch: %q", string(b))
	}
	if in != os.Stdin {
		t.Fatalf("expected stdin to be os.Stdin")
	}
}

func TestParseRedirections_InputMissing(t *testing.T) {
	_, _, _, err := parseRedirections([]string{"cat", "<", "/no/such/file"})
	if err == nil {
		t.Fatal("expected error for missing input file")
	}
}

func TestRunBuiltIn_EchoAndPwdAndCd(t *testing.T) {
	var buf bytes.Buffer
	if err := runBuiltIn([]string{"echo", "a", "b"}, &buf); err != nil {
		t.Fatalf("echo failed: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "a b" {
		t.Fatalf("echo output wrong: %q", buf.String())
	}

	orig, _ := os.Getwd()
	td := t.TempDir()
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	buf.Reset()
	if err := runBuiltIn([]string{"pwd"}, &buf); err != nil {
		t.Fatalf("pwd failed: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	realOut, _ := filepath.EvalSymlinks(out)
	realTd, _ := filepath.EvalSymlinks(td)

	if realOut != realTd {
		t.Fatalf("pwd expected %q got %q", realTd, realOut)
	}
	_ = os.Chdir(orig)
}

func TestRunBuiltIn_CdErrors(t *testing.T) {
	err := runBuiltIn([]string{"cd", "/no/such/dir"}, os.Stdout)
	if err == nil {
		t.Fatal("expected error from cd to non-existent dir")
	}
}

func TestRunBuiltIn_KillErrors(t *testing.T) {
	if err := runBuiltIn([]string{"kill"}, os.Stdout); err == nil {
		t.Fatal("expected error for kill without pid")
	}
}

func TestRunLogical_SingleBuiltinRedirect(t *testing.T) {
	td := t.TempDir()
	fn := filepath.Join(td, "f.txt")
	if err := runLogical("echo hello > " + fn); err != nil {
		t.Fatalf("runLogical failed: %v", err)
	}
	b, err := os.ReadFile(fn)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if strings.TrimSpace(string(b)) != "hello" {
		t.Fatalf("file content wrong: %q", string(b))
	}
}

func TestRunLogical_PipelineExternal_Capture(t *testing.T) {
	out := captureStdout(func() {
		if err := runLogical(`echo "one two three" | wc -w`); err != nil {
			t.Fatalf("runLogical pipeline failed: %v", err)
		}
	})

	got := strings.TrimSpace(out)
	if got != "3" && !strings.HasSuffix(got, "3") {
		t.Fatalf("unexpected wc output: %q", out)
	}
}

func TestRunPipeline_RedirectInPipeline(t *testing.T) {
	td := t.TempDir()
	fn := filepath.Join(td, "p.txt")
	if err := runLogical("echo foo > " + fn + ` | wc -c`); err != nil {
		t.Fatalf("runLogical failed: %v", err)
	}
	b, err := os.ReadFile(fn)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if strings.TrimSpace(string(b)) != "foo" {
		t.Fatalf("file content wrong: %q", string(b))
	}
}

func TestCombinedLogicalBehaviour(t *testing.T) {
	out := captureStdout(func() {
		got := splitLogical("false || echo a && echo b")
		if len(got) != 3 {
			t.Fatalf("unexpected parsed cmds: %#v", got)
		}
		for idx, c := range got {
			if err := runLogical(c.cmd); err != nil {
				if idx+1 < len(got) && got[idx+1].op == "&&" {
					break
				}
			} else {
				if idx+1 < len(got) && got[idx+1].op == "||" {
					break
				}
			}
		}
	})
	s := strings.TrimSpace(out)
	if !strings.Contains(s, "a") || !strings.Contains(s, "b") {
		t.Fatalf("expected a and b in output, got: %q", out)
	}
}

func TestRunBuiltIn_Ps(t *testing.T) {
	var buf bytes.Buffer
	if err := runBuiltIn([]string{"ps"}, &buf); err != nil {
		t.Fatalf("ps failed: %v", err)
	}
	s := buf.String()
	if !strings.Contains(s, "PID") && !strings.Contains(s, "COMMAND") {
		t.Fatalf("unexpected ps output: %q", s)
	}
}

func TestRunBuiltIn_KillProcess(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Skip("sleep not available, skipping kill test")
	}

	pid := cmd.Process.Pid

	err := runBuiltIn([]string{"kill", strconv.Itoa(pid)}, os.Stdout)
	if err != nil {
		t.Fatalf("kill failed: %v", err)
	}

	done := make(chan error)
	go func() { done <- cmd.Wait() }()

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("process not killed within timeout")
	case <-done:
	}
}

func TestUnixShell_ExecutesSimpleCommands(t *testing.T) {
	r, w, _ := os.Pipe()

	oldStdin := os.Stdin
	os.Stdin = r

	out := captureStdout(func() {
		go func() {
			_, _ = fmt.Fprintln(w, "echo test123")
			_, _ = fmt.Fprintln(w, "pwd")
			_ = w.Close()
		}()

		UnixShell()
	})

	os.Stdin = oldStdin

	if !strings.Contains(out, "test123") {
		t.Fatalf("expected echo output in shell output, got: %q", out)
	}
	if !strings.Contains(out, "/") {
		t.Fatalf("expected pwd output in shell output, got: %q", out)
	}
}

func TestSigCancel_SendsSIGINT(t *testing.T) {
	sigc := make(chan os.Signal, 1)

	cmd := exec.Command("sleep", "3")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	currentMu.Lock()
	current = []*exec.Cmd{cmd}
	currentMu.Unlock()

	done := make(chan struct{})
	go func() {
		SigCancel(sigc)
		close(done)
	}()

	sigc <- os.Interrupt

	time.Sleep(300 * time.Millisecond)

	err := cmd.Wait()
	if err == nil {
		t.Fatalf("expected process to be interrupted, but exited cleanly")
	}

	close(sigc)
	<-done
}

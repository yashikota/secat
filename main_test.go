package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func runCLI(args []string, stdin string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRunFile(t *testing.T) {
	p := writeFile(t, ".env", "A=abcdefgh\n")
	code, out, _ := runCLI([]string{p}, "")
	if code != 0 || out != "A=ab******\n" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestRunStdin(t *testing.T) {
	code, out, _ := runCLI(nil, "B=hunter2\n")
	if code != 0 || out != "B=h******\n" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestRunDashStdin(t *testing.T) {
	code, out, _ := runCLI([]string{"-"}, "B=hunter2\n")
	if code != 0 || out != "B=h******\n" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestRunMultipleFiles(t *testing.T) {
	a := writeFile(t, "a.env", "A=abcdefgh\n")
	b := writeFile(t, "b.env", "B=abcdefgh\n")
	code, out, _ := runCLI([]string{a, b}, "")
	if code != 0 || out != "A=ab******\nB=ab******\n" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestRunStateResetBetweenFiles(t *testing.T) {
	a := writeFile(t, "a.env", "A=\"unterminated\n")
	b := writeFile(t, "b.env", "C=abcdefgh\n")
	code, out, _ := runCLI([]string{a, b}, "")
	if code != 0 || !strings.HasSuffix(out, "C=ab******\n") {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestRunMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.env")
	ok := writeFile(t, "ok.env", "A=abcdefgh\n")
	code, out, errOut := runCLI([]string{missing, ok}, "")
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if out != "A=ab******\n" {
		t.Errorf("out = %q, want the readable file's output", out)
	}
	if !strings.Contains(errOut, "secat: ") || !strings.Contains(errOut, "missing.env") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestRunDirectoryArgument(t *testing.T) {
	code, _, errOut := runCLI([]string{t.TempDir()}, "")
	if code != 1 || errOut == "" {
		t.Errorf("code=%d stderr=%q, want 1 and a message", code, errOut)
	}
}

func TestRunFlags(t *testing.T) {
	code, out, _ := runCLI([]string{"--ratio", "0.5", "--max", "10"}, "A=abcdefgh\n")
	if code != 0 || out != "A=abcd****\n" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestRunInvalidRatio(t *testing.T) {
	code, _, errOut := runCLI([]string{"--ratio", "1.5"}, "")
	if code != 2 || !strings.Contains(errOut, "ratio") {
		t.Errorf("code=%d stderr=%q", code, errOut)
	}
}

func TestRunInvalidMax(t *testing.T) {
	code, _, errOut := runCLI([]string{"--max", "-1"}, "")
	if code != 2 || !strings.Contains(errOut, "max") {
		t.Errorf("code=%d stderr=%q", code, errOut)
	}
}

func TestRunUnknownFlag(t *testing.T) {
	if code, _, _ := runCLI([]string{"--nope"}, ""); code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
}

func TestRunVersion(t *testing.T) {
	code, out, _ := runCLI([]string{"--version"}, "")
	if code != 0 || out != "secat dev\n" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestRunHelp(t *testing.T) {
	code, _, errOut := runCLI([]string{"-h"}, "")
	if code != 0 || !strings.Contains(errOut, "Usage") {
		t.Errorf("code=%d stderr=%q", code, errOut)
	}
}

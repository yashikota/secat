package mask

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzStreamInvariants(f *testing.F) {
	inputs, _ := filepath.Glob(filepath.Join("testdata", "*.in"))
	for _, in := range inputs {
		if src, err := os.ReadFile(in); err == nil {
			f.Add(string(src))
		}
	}
	for _, s := range []string{
		"PORT=3000\n",
		"KEY=\"ab\\\"cd\"\r\n",
		"key: |\n  secret\nnext: v",
		"- k: v\n  - \"x\",\n",
		"-----BEGIN X-----\nabc\n-----END X-----\n",
		"K=\xff\xfe\xfd\n",
		"s = '''\nx\n'''",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data string) {
		var out bytes.Buffer
		if err := Stream(strings.NewReader(data), &out, Default); err != nil {
			t.Fatalf("Stream: %v", err)
		}
		inLines := strings.Split(data, "\n")
		outLines := strings.Split(out.String(), "\n")
		if len(inLines) != len(outLines) {
			t.Fatalf("line count changed: %d -> %d", len(inLines), len(outLines))
		}
		for i := range inLines {
			in, got := []rune(inLines[i]), []rune(outLines[i])
			if len(in) != len(got) {
				t.Fatalf("line %d rune count changed: %q -> %q", i, inLines[i], outLines[i])
			}
			for j := range in {
				if got[j] != in[j] && got[j] != '*' {
					t.Fatalf("line %d rune %d changed to %q, not '*': %q -> %q", i, j, got[j], inLines[i], outLines[i])
				}
			}
		}
	})
}

func FuzzEnvValueLeak(f *testing.F) {
	for _, s := range []string{"", "3000", "hunter2", "sk-proj-AbCdEf1234567890", "a=b", "postgres://u:p@h/db", "パスワード12"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, v string) {
		if strings.ContainsAny(v, "\n\r\"'`#\\ \t") || strings.HasPrefix(v, "{") || strings.HasPrefix(v, "[") {
			t.Skip()
		}
		got := NewMasker(Default).Line("K=" + v)
		want := "K=" + Value(v, Default)
		if got != want {
			t.Fatalf("Line(%q) = %q, want %q", "K="+v, got, want)
		}

		masked := []rune(strings.TrimPrefix(got, "K="))
		visible := 0
		for _, r := range masked {
			if r != '*' {
				visible++
			}
		}
		if limit := min(len(masked)/4, Default.Max); visible > limit {
			t.Fatalf("Line(%q) shows %d runes, limit %d", "K="+v, visible, limit)
		}
	})
}

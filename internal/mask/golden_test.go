package mask

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden")

func TestGolden(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "*.in"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no testdata/*.in files")
	}
	for _, in := range inputs {
		t.Run(filepath.Base(in), func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := Stream(bytes.NewReader(src), &out, Default); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "SECRET") {
				t.Errorf("output leaks a SECRET marker:\n%s", out.String())
			}

			golden := strings.TrimSuffix(in, ".in") + ".golden"
			if *update {
				if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run with -update to create it)", err)
			}
			if out.String() != string(want) {
				t.Errorf("mismatch with %s\n got:\n%s\nwant:\n%s", golden, out.String(), want)
			}
		})
	}
}

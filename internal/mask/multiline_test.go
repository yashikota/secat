package mask

import (
	"slices"
	"strings"
	"testing"
)

func maskLines(m *Masker, in []string) []string {
	out := make([]string, len(in))
	for i, l := range in {
		out[i] = m.Line(l)
	}
	return out
}

var multilineCases = []struct {
	name    string
	in      []string
	want    []string
	secrets []string
}{
	{
		name: "env private key in double quotes",
		in: []string{
			`KEY="-----BEGIN RSA PRIVATE KEY-----`,
			`MIIEow`,
			`-----END RSA PRIVATE KEY-----"`,
			`NEXT=abcdefgh`,
		},
		want: []string{
			`KEY="` + Value("-----BEGIN RSA PRIVATE KEY-----", Default),
			Full("MIIEow"),
			Full("-----END RSA PRIVATE KEY-----") + `"`,
			`NEXT=ab******`,
		},
		secrets: []string{"MIIEow"},
	},
	{
		name: "closing quote keeps trailing syntax",
		in: []string{
			`  "k": "abcd`,
			`  efgh\"ij",`,
			`  "n": 1234`,
		},
		want: []string{
			`  "k": "a***`,
			Full(`  efgh\"ij`) + `",`,
			`  "n": 1***`,
		},
		secrets: []string{"efgh"},
	},
	{
		name:    "single quote spans lines",
		in:      []string{`K='abcd`, `secretline'`, `N=abcdefgh`},
		want:    []string{`K='a***`, Full("secretline") + `'`, `N=ab******`},
		secrets: []string{"secretline"},
	},
	{
		name: "yaml literal block",
		in: []string{
			`key: |`,
			`  line1secret`,
			``,
			`  line2secret`,
			`other: hunter2`,
		},
		want: []string{
			`key: |`,
			Full("  line1secret"),
			``,
			Full("  line2secret"),
			`other: h******`,
		},
		secrets: []string{"line1secret", "line2secret"},
	},
	{
		name: "yaml nested folded block",
		in: []string{
			`a:`,
			`  b: >-`,
			`    foldedsecret`,
			`  c: abcdefgh`,
		},
		want: []string{
			`a:`,
			`  b: >-`,
			Full("    foldedsecret"),
			`  c: ab******`,
		},
		secrets: []string{"foldedsecret"},
	},
	{
		name:    "yaml block with indicators and comment",
		in:      []string{`k: |+2 # keep`, `  kept`, `z: abcdefgh`},
		want:    []string{`k: |+2 # keep`, Full("  kept"), `z: ab******`},
		secrets: []string{"kept"},
	},
	{
		name:    "yaml block in list mapping",
		in:      []string{`- k: |`, `    vsecret`, `- o: abcdefgh`},
		want:    []string{`- k: |`, Full("    vsecret"), `- o: ab******`},
		secrets: []string{"vsecret"},
	},
	{
		name:    "yaml block as list item",
		in:      []string{`items:`, `  - |`, `    itemsecret`, `  - abcdefgh`},
		want:    []string{`items:`, `  - |`, Full("    itemsecret"), `  - ab******`},
		secrets: []string{"itemsecret"},
	},
	{
		name: "bare pem block",
		in: []string{
			`-----BEGIN CERTIFICATE-----`,
			`MIIBsecret`,
			`-----END CERTIFICATE-----`,
			`after: abcdefgh`,
		},
		want: []string{
			Full("-----BEGIN CERTIFICATE-----"),
			Full("MIIBsecret"),
			Full("-----END CERTIFICATE-----"),
			`after: ab******`,
		},
		secrets: []string{"MIIBsecret", "CERTIFICATE"},
	},
	{
		name: "unquoted pem value",
		in:   []string{`CERT=-----BEGIN CERTIFICATE-----`, `MIIBsecret`, `-----END CERTIFICATE-----`, `N=abcdefgh`},
		want: []string{
			`CERT=` + Value("-----BEGIN CERTIFICATE-----", Default),
			Full("MIIBsecret"),
			Full("-----END CERTIFICATE-----"),
			`N=ab******`,
		},
		secrets: []string{"MIIBsecret"},
	},
	{
		name:    "toml multiline basic string",
		in:      []string{`s = """`, `secretline`, `"""`, `t = 1234`},
		want:    []string{`s = """`, Full("secretline"), `"""`, `t = 1***`},
		secrets: []string{"secretline"},
	},
	{
		name:    "toml multiline literal string",
		in:      []string{`s = '''abcd`, `secretline'''`, `t = 1234`},
		want:    []string{`s = '''a***`, Full("secretline") + `'''`, `t = 1***`},
		secrets: []string{"secretline"},
	},
	{
		name: "toml triple quote on one line",
		in:   []string{`s = """abcdefgh"""`, `t = 1234`},
		want: []string{`s = """ab******"""`, `t = 1***`},
	},
	{
		name:    "backslash continuation",
		in:      []string{`KEY=abc\`, `def\`, `ghi`, `N=abcdefgh`},
		want:    []string{`KEY=a***`, Full(`def\`), Full("ghi"), `N=ab******`},
		secrets: []string{"def", "ghi"},
	},
	{
		name: "pipe value in env is not a yaml block",
		in:   []string{`K=|`, `NEXT=abcdefgh`},
		want: []string{`K=*`, `NEXT=ab******`},
	},
}

func TestMultiline(t *testing.T) {
	for _, tt := range multilineCases {
		t.Run(tt.name, func(t *testing.T) {
			got := maskLines(NewMasker(Default), tt.in)
			if !slices.Equal(got, tt.want) {
				t.Errorf("\n got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestMultilineNoLeak(t *testing.T) {
	for _, tt := range multilineCases {
		t.Run(tt.name, func(t *testing.T) {
			out := strings.Join(maskLines(NewMasker(Default), tt.in), "\n")
			for _, s := range tt.secrets {
				if strings.Contains(out, s) {
					t.Errorf("output leaks %q:\n%s", s, out)
				}
			}
		})
	}
}

func TestMaskerIndependent(t *testing.T) {
	first := NewMasker(Default)
	first.Line(`KEY="unterminated`)
	if got := first.Line(`N=abcdefgh`); got != Full(`N=abcdefgh`) {
		t.Fatalf("precondition: first masker should still be inside the quote, got %q", got)
	}
	if got := NewMasker(Default).Line(`N=abcdefgh`); got != `N=ab******` {
		t.Errorf("new masker = %q, want %q", got, `N=ab******`)
	}
}

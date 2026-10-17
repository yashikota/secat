package mask

import "testing"

func TestLine(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		// passthrough
		{"empty", ``, ``},
		{"hash comment", `# comment KEY=secret`, `# comment KEY=secret`},
		{"indented hash comment", `  # KEY=secret`, `  # KEY=secret`},
		{"ini comment", `; ini comment`, `; ini comment`},
		{"slash comment", `// json comment`, `// json comment`},
		{"section", `[default]`, `[default]`},
		{"toml array table", `[[servers]]`, `[[servers]]`},
		{"open brace", `{`, `{`},
		{"close brace comma", `},`, `},`},
		{"open bracket", `[`, `[`},
		{"close bracket comma", `],`, `],`},
		{"plain text", `just some text`, `just some text`},
		{"yaml key no value", `key:`, `key:`},
		{"yaml parent", `parent:`, `parent:`},
		{"yaml object open", `obj: {`, `obj: {`},
		{"yaml list open", `list: [`, `list: [`},
		{"url without delimiter", `url http://example.com`, `url http://example.com`},
		{"yaml doc separator", `---`, `---`},

		// env
		{"env", `PORT=3000`, `PORT=3***`},
		{"env password", `DB_PASSWORD=hunter2`, `DB_PASSWORD=h******`},
		{"env empty", `EMPTY=`, `EMPTY=`},
		{"env export", `export TOKEN=abcdefgh`, `export TOKEN=ab******`},
		{"env equals in value", `K=a=b`, `K=***`},
		{"env url value", `URL=postgres://u:p@h/db`, `URL=post***************`},
		{"env double quoted", `KEY="hunter2"`, `KEY="h******"`},
		{"env single quoted", `KEY='hunter2'`, `KEY='h******'`},
		{"env backtick quoted", "KEY=`hunter2`", "KEY=`h******`"},
		{"env escaped quote", `KEY="ab\"cd"`, `KEY="a*****"`},
		{"env empty quotes", `KEY=""`, `KEY=""`},
		{"env inline comment", `KEY=abc #comment`, `KEY=*** #comment`},
		{"env hash in value", `KEY=abc#def`, `KEY=a******`},
		{"env quoted with comment", `KEY="abcd" # note`, `KEY="a***" # note`},
		{"env indented", `  INDENTED=abcdefgh`, `  INDENTED=ab******`},
		{"env spaces around equals", `KEY = abcdefgh`, `KEY = ab******`},
		{"env trailing spaces", `KEY=abcdefgh  `, `KEY=ab******  `},

		// ini / toml / aws
		{"aws", `aws_secret_access_key = wJalrXUtnFEMI`, `aws_secret_access_key = wJa**********`},
		{"toml string", `name = "hunter2"`, `name = "h******"`},
		{"toml int", `port = 8080`, `port = 8***`},
		{"dotted key", `a.b.c = abcdefgh`, `a.b.c = ab******`},

		// yaml
		{"yaml", `password: hunter2`, `password: h******`},
		{"yaml nested", `  nested: abcdefgh`, `  nested: ab******`},
		{"yaml bool", `enabled: true`, `enabled: t***`},
		{"yaml inline comment", `key: abcdefgh # note`, `key: ab****** # note`},
		{"yaml quoted key", `"my key": abcdefgh`, `"my key": ab******`},
		{"yaml single quoted key", `'k': abcdefgh`, `'k': ab******`},
		{"yaml list scalar", `- sk-abcdefgh`, `- sk*********`},
		{"yaml list quoted", `  - "tok-12345678"`, `  - "tok*********"`},
		{"yaml list kv", `- key: hunter2`, `- key: h******`},
		{"yaml list inline map", `- {a: b}`, `- {*****`},
		{"yaml list open map", `- {`, `- {`},
		{"yaml list comment", `- # note`, `- # note`},
		{"yaml inline array", `keys: [abc, defgh]`, `keys: [ab*********`},

		// json
		{"json string", `  "token": "abcdefgh",`, `  "token": "ab******",`},
		{"json number", `  "port": 8080,`, `  "port": 8***,`},
		{"json null", `  "n": null`, `  "n": n***`},
		{"json object open", `  "obj": {`, `  "obj": {`},
		{"json array open", `  "arr": [`, `  "arr": [`},
		{"json inline array", `  "keys": ["sk-abc"],`, `  "keys": ["********,`},
		{"json compact", `"a":"abcdefgh"`, `"a":"ab******"`},
		{"json array string element", `  "tok-12345678",`, `  "tok*********",`},
		{"json array number element", `  12345678`, `  12******`},
		{"json array bool element", `  true,`, `  t***,`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMasker(Default).Line(tt.in); got != tt.want {
				t.Errorf("Line(%q)\n got %q\nwant %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLineOptions(t *testing.T) {
	o := Options{Ratio: 0, Max: 0}
	tests := []struct{ in, want string }{
		{`KEY=abc`, `KEY=***`},
		{`  "a": "b"`, `  "a": "*"`},
	}
	for _, tt := range tests {
		if got := NewMasker(o).Line(tt.in); got != tt.want {
			t.Errorf("Line(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

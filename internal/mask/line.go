package mask

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// sectionRe matches INI/TOML section headers such as [default] or [[servers]].
	sectionRe = regexp.MustCompile(`^\[\[?[^\[\],]+\]\]?$`)
	// blockRe matches a YAML block scalar header such as |, >-, or |+2 # note.
	blockRe = regexp.MustCompile(`^[|>][0-9]?[+-]?[0-9]?\s*(#.*)?$`)
)

type state int

const (
	stateNone         state = iota
	stateQuote              // inside a quoted value that did not close on its line
	stateYAMLBlock          // inside a YAML block scalar
	statePEM                // between -----BEGIN and -----END
	stateContinuation       // after an unquoted value ending in a backslash
)

// Masker masks one line at a time, carrying multi-line values across lines.
// Use one Masker per file.
type Masker struct {
	opts  Options
	state state
	// closer is the quote that ends stateQuote.
	closer string
	// indent is the width of the current line's leading whitespace; in
	// stateYAMLBlock it is the indent of the line that opened the block.
	indent int
}

// NewMasker returns a Masker that masks values with o.
func NewMasker(o Options) *Masker {
	return &Masker{opts: o}
}

// Line masks the values in line, which must not contain a line terminator.
func (m *Masker) Line(line string) string {
	switch m.state {
	case stateQuote:
		i := findCloser(line, m.closer)
		if i < 0 {
			return Full(line)
		}
		m.state = stateNone
		return Full(line[:i]) + line[i:]
	case stateYAMLBlock:
		body := strings.TrimLeft(line, " \t")
		if body == "" {
			return line
		}
		if len(line)-len(body) > m.indent {
			return Full(line)
		}
		m.state = stateNone
	case statePEM:
		if strings.Contains(line, "-----END") {
			m.state = stateNone
		}
		return Full(line)
	case stateContinuation:
		if !strings.HasSuffix(strings.TrimRight(line, " \t"), `\`) {
			m.state = stateNone
		}
		return Full(line)
	}

	out := m.parse(line)
	if m.state == stateNone && strings.Contains(line, "-----BEGIN") && !strings.Contains(line, "-----END") {
		m.state = statePEM
	}
	return out
}

func (m *Masker) parse(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "-----BEGIN") {
		return Full(line)
	}
	if trimmed == "" || isComment(trimmed) || isStructural(trimmed) {
		return line
	}

	body := strings.TrimLeft(line, " \t")
	prefix := line[:len(line)-len(body)]
	m.indent = len(prefix)

	if strings.HasPrefix(body, "- ") {
		item := strings.TrimLeft(body[1:], " \t")
		if blockRe.MatchString(item) {
			// The block belongs to the item, so its lines are indented past the dash.
			m.state = stateYAMLBlock
			return line
		}
		prefix += body[:len(body)-len(item)]
		m.indent = len(prefix)
		if out, ok := m.keyValue(item); ok {
			return prefix + out
		}
		if strings.HasPrefix(item, "#") {
			return line
		}
		return prefix + m.value(item, true)
	}

	if out, ok := m.keyValue(body); ok {
		return prefix + out
	}
	if isJSONScalar(body) {
		return prefix + m.value(body, true)
	}
	return line
}

// keyValue masks s if it has the form [export ]key<sep>value, where sep is
// '=' or ':' followed by a space or the end of the line.
func (m *Masker) keyValue(s string) (string, bool) {
	start := 0
	if strings.HasPrefix(s, "export ") {
		start = len(s) - len(strings.TrimLeft(s[len("export"):], " \t"))
	}

	end, quotedKey := scanKey(s, start)
	if end == start {
		return "", false
	}

	sep := end
	for sep < len(s) && (s[sep] == ' ' || s[sep] == '\t') {
		sep++
	}
	if sep == len(s) {
		return "", false
	}
	switch s[sep] {
	case '=':
	case ':':
		// A quoted key is unambiguous, so compact JSON ("a":"b") is allowed.
		next := sep + 1
		if !quotedKey && next < len(s) && s[next] != ' ' && s[next] != '\t' {
			return "", false
		}
	default:
		return "", false
	}

	v := sep + 1
	for v < len(s) && (s[v] == ' ' || s[v] == '\t') {
		v++
	}
	if s[sep] == ':' && blockRe.MatchString(s[v:]) {
		m.state = stateYAMLBlock
		return s, true
	}
	jsonKey := quotedKey && s[start] == '"'
	return s[:v] + m.value(s[v:], jsonKey), true
}

// scanKey returns the end of the key starting at s[start] and whether the key
// is quoted. end == start means no key was found.
func scanKey(s string, start int) (end int, quoted bool) {
	if start >= len(s) {
		return start, false
	}
	if q := s[start]; q == '"' || q == '\'' {
		if i := closingQuote(s[start:], q); i > 0 {
			return start + i + 1, true
		}
		return start, false
	}
	end = start
	for end < len(s) && isKeyChar(s[end]) {
		end++
	}
	return end, false
}

func isKeyChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		c == '_' || c == '.' || c == '-' || c == '/'
}

// value masks a value as found after a key separator or as a bare element.
// trailingComma reports whether a trailing ',' is JSON syntax, not data.
func (m *Masker) value(v string, trailingComma bool) string {
	if v == "" {
		return v
	}
	if v[0] == '{' || v[0] == '[' {
		// An opener alone starts a block whose members follow on later lines.
		// An empty {} or [] holds nothing to hide.
		after := strings.TrimSpace(v[1:])
		if after == "" || strings.HasPrefix(after, "#") || isEmptyStructure(v) {
			return v
		}
	}
	if q := v[0]; q == '"' || q == '\'' || q == '`' {
		closer := string(q)
		if strings.HasPrefix(v, `"""`) || strings.HasPrefix(v, `'''`) {
			closer = v[:3]
		}
		inner := v[len(closer):]
		if i := findCloser(inner, closer); i >= 0 {
			return closer + Value(inner[:i], m.opts) + inner[i:]
		}
		m.state, m.closer = stateQuote, closer
		return closer + Value(inner, m.opts)
	}

	core, suffix := v, ""
	if i := inlineComment(core); i >= 0 {
		core, suffix = core[:i], core[i:]
	}
	trimmed := strings.TrimRight(core, " \t")
	core, suffix = trimmed, core[len(trimmed):]+suffix
	if trailingComma && strings.HasSuffix(core, ",") {
		core, suffix = core[:len(core)-1], ","+suffix
	}
	if strings.HasSuffix(core, `\`) && strings.TrimSpace(suffix) == "" {
		m.state = stateContinuation
	}
	return Value(core, m.opts) + suffix
}

// findCloser returns the index of the first closer in s, or -1.
// Backslash escapes are honored only for double-quote closers.
func findCloser(s, closer string) int {
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && closer[0] == '"':
			i++
		case strings.HasPrefix(s[i:], closer):
			return i
		}
	}
	return -1
}

// closingQuote returns the index in s of the quote closing s[0], or -1.
func closingQuote(s string, q byte) int {
	if i := findCloser(s[1:], string(q)); i >= 0 {
		return i + 1
	}
	return -1
}

// inlineComment returns the index of a '#' comment preceded by whitespace in
// an unquoted value, or -1.
func inlineComment(v string) int {
	for i := 1; i < len(v); i++ {
		if v[i] == '#' && (v[i-1] == ' ' || v[i-1] == '\t') {
			return i - 1
		}
	}
	return -1
}

// isEmptyStructure reports whether v is {} or [] (inner spaces and a trailing
// ',' allowed).
func isEmptyStructure(v string) bool {
	v = strings.TrimSuffix(strings.TrimSpace(v), ",")
	if len(v) < 2 {
		return false
	}
	pair := string(v[0]) + string(v[len(v)-1])
	return (pair == "{}" || pair == "[]") && strings.TrimSpace(v[1:len(v)-1]) == ""
}

func isComment(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") ||
		strings.HasPrefix(trimmed, "//")
}

func isStructural(trimmed string) bool {
	switch strings.TrimSuffix(trimmed, ",") {
	case "{", "}", "[", "]", "{}", "[]":
		return true
	}
	return sectionRe.MatchString(trimmed)
}

// isJSONScalar reports whether s is a lone JSON array element: a string,
// number, true, false or null, optionally followed by ','.
func isJSONScalar(s string) bool {
	s = strings.TrimSuffix(strings.TrimRight(s, " \t"), ",")
	switch s {
	case "true", "false", "null":
		return true
	}
	if len(s) >= 2 && s[0] == '"' && closingQuote(s, '"') == len(s)-1 {
		return true
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

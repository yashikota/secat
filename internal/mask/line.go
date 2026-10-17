package mask

import (
	"regexp"
	"strconv"
	"strings"
)

// sectionRe matches INI/TOML section headers such as [default] or [[servers]].
var sectionRe = regexp.MustCompile(`^\[\[?[^\[\],]+\]\]?$`)

// Masker masks one line at a time. Use one Masker per file.
type Masker struct {
	opts Options
}

// NewMasker returns a Masker that masks values with o.
func NewMasker(o Options) *Masker {
	return &Masker{opts: o}
}

// Line masks the values in line, which must not contain a line terminator.
func (m *Masker) Line(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || isComment(trimmed) || isStructural(trimmed) {
		return line
	}

	body := strings.TrimLeft(line, " \t")
	prefix := line[:len(line)-len(body)]

	if strings.HasPrefix(body, "- ") {
		item := strings.TrimLeft(body[1:], " \t")
		prefix += body[:len(body)-len(item)]
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
		after := strings.TrimSpace(v[1:])
		if after == "" || strings.HasPrefix(after, "#") {
			return v
		}
	}
	if q := v[0]; q == '"' || q == '\'' || q == '`' {
		if i := closingQuote(v, q); i > 0 {
			return string(q) + Value(v[1:i], m.opts) + v[i:]
		}
		return string(q) + Value(v[1:], m.opts)
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
	return Value(core, m.opts) + suffix
}

// closingQuote returns the index in s of the quote closing s[0], or -1.
// Backslash escapes are honored only inside double quotes.
func closingQuote(s string, q byte) int {
	for i := 1; i < len(s); i++ {
		switch {
		case s[i] == '\\' && q == '"':
			i++
		case s[i] == q:
			return i
		}
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

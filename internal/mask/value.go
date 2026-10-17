// Package mask hides secret values in key-value text while keeping its layout.
package mask

import (
	"fmt"
	"math"
	"strings"
)

// Options controls how much of each value stays visible.
type Options struct {
	// Ratio is the fraction of a value's runes shown from its head (0 to 1).
	Ratio float64
	// Max caps the number of visible runes.
	Max int
}

// Default shows a quarter of each value, at most 4 runes.
var Default = Options{Ratio: 0.25, Max: 4}

// Validate reports whether o is usable.
func (o Options) Validate() error {
	if !(o.Ratio >= 0 && o.Ratio <= 1) {
		return fmt.Errorf("ratio must be between 0 and 1, got %v", o.Ratio)
	}
	if o.Max < 0 {
		return fmt.Errorf("max must be 0 or greater, got %d", o.Max)
	}
	return nil
}

// Value keeps the first min(floor(n*Ratio), Max) runes of s and replaces the
// rest with '*', so the result has the same rune count as s.
func Value(s string, o Options) string {
	r := []rune(s)
	keep := min(int(math.Floor(float64(len(r))*o.Ratio)), o.Max)
	for i := keep; i < len(r); i++ {
		r[i] = '*'
	}
	return string(r)
}

// Full keeps the leading spaces and tabs of s and replaces every other rune
// with '*'.
func Full(s string) string {
	body := strings.TrimLeft(s, " \t")
	indent := s[:len(s)-len(body)]
	return indent + strings.Repeat("*", len([]rune(body)))
}

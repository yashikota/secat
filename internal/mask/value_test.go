package mask

import (
	"math"
	"strings"
	"testing"
)

func TestValue(t *testing.T) {
	long := strings.Repeat("x", 100)
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"a", "*"},
		{"ab", "**"},
		{"abc", "***"},
		{"3000", "3***"},
		{"abcde", "a****"},
		{"hunter2", "h******"},
		{"sk-proj-AbCdEf1234567890", "sk-p" + strings.Repeat("*", 20)},
		{long, "xxxx" + strings.Repeat("*", 96)},
		{"パスワード12", "パ******"},
	}
	for _, tt := range tests {
		if got := Value(tt.in, Default); got != tt.want {
			t.Errorf("Value(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValueOptions(t *testing.T) {
	tests := []struct {
		opts     Options
		in, want string
	}{
		{Options{Ratio: 0, Max: 4}, "abcdefgh", "********"},
		{Options{Ratio: 1, Max: 100}, "abc", "abc"},
		{Options{Ratio: 1, Max: 0}, "abc", "***"},
		{Options{Ratio: 0.5, Max: 4}, "abcdef", "abc***"},
	}
	for _, tt := range tests {
		if got := Value(tt.in, tt.opts); got != tt.want {
			t.Errorf("Value(%q, %+v) = %q, want %q", tt.in, tt.opts, got, tt.want)
		}
	}
}

func TestValidate(t *testing.T) {
	valid := []Options{
		{Ratio: 0, Max: 4},
		{Ratio: 1, Max: 4},
		{Ratio: 0.25, Max: 0},
	}
	for _, o := range valid {
		if err := o.Validate(); err != nil {
			t.Errorf("Validate(%+v) = %v, want nil", o, err)
		}
	}
	invalid := []Options{
		{Ratio: -0.01, Max: 4},
		{Ratio: 1.01, Max: 4},
		{Ratio: math.NaN(), Max: 4},
		{Ratio: 0.25, Max: -1},
	}
	for _, o := range invalid {
		if err := o.Validate(); err == nil {
			t.Errorf("Validate(%+v) = nil, want error", o)
		}
	}
}

func TestFull(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"  abc", "  ***"},
		{"\tあい", "\t**"},
		{"", ""},
		{"   ", "   "},
		{"a b", "***"},
	}
	for _, tt := range tests {
		if got := Full(tt.in); got != tt.want {
			t.Errorf("Full(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Command secat prints key-value files like cat, masking most of each value.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yashikota/secat/internal/mask"
)

// version is set by GoReleaser via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("secat", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: secat [flags] [file ...]")
		fmt.Fprintln(stderr, "Print key-value files with values masked. Reads stdin when no file or - is given.")
		fs.PrintDefaults()
	}
	opts := mask.Default
	fs.Float64Var(&opts.Ratio, "ratio", opts.Ratio, "fraction of each value shown from its head (0 to 1)")
	fs.IntVar(&opts.Max, "max", opts.Max, "maximum number of visible characters per value")
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintf(stdout, "secat %s\n", version)
		return 0
	}
	if err := opts.Validate(); err != nil {
		fmt.Fprintf(stderr, "secat: %v\n", err)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"-"}
	}
	code := 0
	for _, p := range paths {
		if err := catFile(p, stdin, stdout, opts); err != nil {
			fmt.Fprintf(stderr, "secat: %s: %v\n", p, err)
			code = 1
		}
	}
	return code
}

func catFile(path string, stdin io.Reader, stdout io.Writer, opts mask.Options) error {
	if path == "-" {
		return mask.Stream(stdin, stdout, opts)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return mask.Stream(f, stdout, opts)
}

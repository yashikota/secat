package mask

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// Stream masks r line by line into w, keeping each line's terminator ("\n",
// "\r\n" or none at EOF) exactly as read. Lines may be of any length.
func Stream(r io.Reader, w io.Writer, o Options) error {
	m := NewMasker(o)
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)
	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			body := strings.TrimSuffix(line, "\n")
			if len(body) < len(line) {
				body = strings.TrimSuffix(body, "\r")
			}
			if _, werr := bw.WriteString(m.Line(body) + line[len(body):]); werr != nil {
				return werr
			}
		}
		if errors.Is(err, io.EOF) {
			return bw.Flush()
		}
		if err != nil {
			return err
		}
	}
}

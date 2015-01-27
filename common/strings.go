package common

import (
	"bufio"
	"io"
	"strings"
)

func ForEachLine(r io.Reader, f func(string)) error {
	br := bufio.NewReader(r)
	for {
		line,err := br.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		line = strings.TrimSuffix(line, "\n")
		f(line)
	}
	return nil
}

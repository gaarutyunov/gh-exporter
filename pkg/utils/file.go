package utils

import (
	"bufio"
	"bytes"
	"io"
)

const lineBreak = '\n'

func LineCounter(r io.Reader) (int, error) {
	var count int

	buf := make([]byte, bufio.MaxScanTokenSize)

	for {
		bufferSize, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}

		var buffPosition int
		for {
			i := bytes.IndexByte(buf[buffPosition:], lineBreak)
			if i == -1 || bufferSize == buffPosition {
				break
			}
			buffPosition += i + 1
			count++
		}
		if err == io.EOF {
			break
		}
	}

	return count, nil
}

func IterLines(r io.Reader, iter func(line string) error) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		err := iter(scanner.Text())
		if err != nil {
			return err
		}
	}

	return scanner.Err()
}

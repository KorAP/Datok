package datok

import (
	"bufio"
	"io"
	"strconv"
)

type TokenWriter struct {
	SentenceEnd func(int)
	TextEnd     func(int)
	Flush       func() error
	Token       func(int, []rune)
}

func NewTokenWriter(w io.Writer) *TokenWriter {
	writer := bufio.NewWriter(w)

	return &TokenWriter{
		SentenceEnd: func(_ int) {
			writer.WriteRune('\n')
		},
		TextEnd: func(_ int) {
			writer.WriteRune('\n')
			writer.Flush()
		},
		Token: func(offset int, buf []rune) {
			writer.WriteString(string(buf[offset:]))
			writer.WriteRune('\n')
		},
		Flush: func() error {
			return writer.Flush()
		},
	}
}

// Create a new token writer based on the options
func NewTokenWriterFromOptions(w io.Writer, positionFlag bool, tokenFlag bool, newlineAfterEot bool) *TokenWriter {
	writer := bufio.NewWriter(w)
	posC := 0
	pos := make([]int, 0, 200)

	tw := &TokenWriter{}

	if positionFlag {
		tw.Token = func(offset int, buf []rune) {

			// TODO:
			//   Store in []uint16
			//   and write to string

			// Accept newline after EOT
			if newlineAfterEot && posC == 0 && buf[0] == '\n' && writer.Buffered() != 0 {
				posC--
			}

			posC += offset
			pos = append(pos, posC)
			posC += len(buf) - offset
			pos = append(pos, posC)

			if tokenFlag {
				writer.WriteString(string(buf[offset:]))
				writer.WriteRune('\n')
			}
		}
	} else {
		tw.Token = func(offset int, buf []rune) {
			writer.WriteString(string(buf[offset:]))
			writer.WriteRune('\n')
		}
	}

	tw.SentenceEnd = func(_ int) {
		writer.WriteRune('\n')
	}

	if positionFlag {
		tw.TextEnd = func(_ int) {
			writer.Flush()

			writer.WriteString(strconv.Itoa(pos[0]))
			for _, x := range pos[1:] {
				writer.WriteByte(' ')
				writer.WriteString(strconv.Itoa(x))
			}
			writer.WriteRune('\n')

			posC = 0
			pos = pos[:0]
		}
	} else {
		tw.TextEnd = func(_ int) {
			writer.WriteRune('\n')
			writer.Flush()
		}

	}

	tw.Flush = func() error {
		return writer.Flush()
	}

	return tw
}

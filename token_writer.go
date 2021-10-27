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
func NewTokenWriterFromOptions(w io.Writer, positionFlag bool, tokenFlag bool, sentenceFlag bool, sentencePositionFlag bool, newlineAfterEot bool) *TokenWriter {
	writer := bufio.NewWriter(w)
	posC := 0
	pos := make([]int, 0, 1024)
	sentB := true
	sent := make([]int, 0, 1024)

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

			// Token is the start of a sentence
			if sentB {
				sentB = false
				sent = append(sent, posC)
			}
			posC += len(buf) - offset
			pos = append(pos, posC)

			if tokenFlag {
				writer.WriteString(string(buf[offset:]))
				writer.WriteRune('\n')
			}
		}

		// Only print one token per line
	} else {
		tw.Token = func(offset int, buf []rune) {
			writer.WriteString(string(buf[offset:]))
			writer.WriteRune('\n')
		}
	}

	// Print sentence boundaries
	if sentenceFlag || sentencePositionFlag {
		tw.SentenceEnd = func(offset int) {

			// Add end position of last token to sentence boundary
			sent = append(sent, pos[len(pos)-1])
			sentB = true

			if sentenceFlag {
				writer.WriteRune('\n')
			}
		}

		// Print sentence boundaries as newlines
	} else if sentenceFlag {
		tw.SentenceEnd = func(_ int) {
			writer.WriteRune('\n')
		}

		// Ignore sentence boundaries
	} else {
		tw.SentenceEnd = func(_ int) {}
	}

	if positionFlag || sentencePositionFlag {
		tw.TextEnd = func(_ int) {
			writer.Flush()

			if positionFlag {
				writer.WriteString(strconv.Itoa(pos[0]))
				for _, x := range pos[1:] {
					writer.WriteByte(' ')
					writer.WriteString(strconv.Itoa(x))
				}
				writer.WriteRune('\n')
			}

			if sentencePositionFlag {
				writer.WriteString(strconv.Itoa(sent[0]))
				for _, x := range sent[1:] {
					writer.WriteByte(' ')
					writer.WriteString(strconv.Itoa(x))
				}
				writer.WriteRune('\n')
				sent = sent[:0]
				sentB = true
			}

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

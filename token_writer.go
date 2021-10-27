package datok

import (
	"bufio"
	"io"
	"strconv"
)

type Bits uint8

const (
	TOKENS Bits = 1 << iota
	SENTENCES
	TOKEN_POS
	SENTENCE_POS
	NEWLINE_AFTER_EOT

	SIMPLE = TOKENS | SENTENCES
)

type TokenWriter struct {
	SentenceEnd func(int)
	TextEnd     func(int)
	Flush       func() error
	Token       func(int, []rune)
}

// Create a new token writer based on the options
func NewTokenWriter(w io.Writer, flags Bits) *TokenWriter {
	writer := bufio.NewWriter(w)
	posC := 0
	pos := make([]int, 0, 1024)
	sentB := true
	sent := make([]int, 0, 1024)

	tw := &TokenWriter{}

	if flags&TOKEN_POS != 0 {
		tw.Token = func(offset int, buf []rune) {

			// TODO:
			//   Store in []uint16
			//   and write to string

			// Accept newline after EOT
			if flags&NEWLINE_AFTER_EOT != 0 && posC == 0 && buf[0] == '\n' && writer.Buffered() != 0 {
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

			if flags&TOKENS != 0 {
				writer.WriteString(string(buf[offset:]))
				writer.WriteRune('\n')
			}
		}

		// Only print one token per line
	} else if flags&TOKENS != 0 {
		tw.Token = func(offset int, buf []rune) {
			writer.WriteString(string(buf[offset:]))
			writer.WriteRune('\n')
		}
	} else {
		tw.Token = func(_ int, _ []rune) {}
	}

	// Print sentence boundaries
	if flags&SENTENCE_POS != 0 {
		tw.SentenceEnd = func(offset int) {

			// Add end position of last token to sentence boundary
			// TODO: This only works if token positions are taking into account
			sent = append(sent, pos[len(pos)-1])
			sentB = true

			if flags&SENTENCES != 0 {
				writer.WriteRune('\n')
			}
		}

		// Print sentence boundaries as newlines
	} else if flags&SENTENCES != 0 {
		tw.SentenceEnd = func(_ int) {
			writer.WriteRune('\n')
		}

		// Ignore sentence boundaries
	} else {
		tw.SentenceEnd = func(_ int) {}
	}

	if flags&(TOKEN_POS|SENTENCE_POS) != 0 {
		tw.TextEnd = func(_ int) {
			writer.Flush()

			if flags&TOKEN_POS != 0 {
				writer.WriteString(strconv.Itoa(pos[0]))
				for _, x := range pos[1:] {
					writer.WriteByte(' ')
					writer.WriteString(strconv.Itoa(x))
				}
				writer.WriteRune('\n')
			}

			if flags&SENTENCE_POS != 0 {
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

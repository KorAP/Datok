package datok

import (
	"bufio"
	"io"
	"strconv"
)

type Bits uint8

// TODO-Perf:
// - TokenWriter may support AvailableBuffer(), so tokens can be written
//   directly without a separate buffer. copying from the same underlying
//   byte array is a nop thren (Go 1.18).


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
	// Fail        func(int)
}

// Create a new token writer based on the options
func NewTokenWriter(w io.Writer, flags Bits) *TokenWriter {
	writer := bufio.NewWriter(w)
	posC := 0
	pos := make([]int, 0, 1024)
	sentB := true
	sent := make([]int, 0, 1024)
	init := true

	tw := &TokenWriter{}

	// tw.Fail = func(_ int) {}

	// Collect token positions and maybe tokens
	if flags&(TOKEN_POS|SENTENCE_POS) != 0 {

		// TODO:
		//   Split to
		//   - Token_pos+Tokens+Newline
		//   - Token_pos+Newline
		//   - Token_pos|Sentence_pos
		//   - Sentence_pos
		//   - Tokens

		tw.Token = func(offset int, buf []rune) {

			// TODO:
			//   Store in []uint16
			//   and write to string

			// Accept newline after EOT
			if posC == 0 && flags&NEWLINE_AFTER_EOT != 0 && buf[0] == '\n' && !init {
				posC--
			}

			init = false

			posC += offset
			pos = append(pos, posC)

			// Token is the start of a sentence
			if sentB {
				sentB = false
				sent = append(sent, posC)
			}
			posC += len(buf) - offset
			pos = append(pos, posC)

			// Collect tokens also
			if flags&TOKENS != 0 {
				writer.WriteString(string(buf[offset:]))
				writer.WriteByte('\n')
			}
		}

		// Collect tokens
	} else if flags&TOKENS != 0 {
		tw.Token = func(offset int, buf []rune) {
			writer.WriteString(string(buf[offset:]))
			writer.WriteByte('\n')
		}

		// Ignore tokens
	} else {
		tw.Token = func(_ int, _ []rune) {}
	}

	// Collect sentence positions and maybe sentence boundaries
	if flags&SENTENCE_POS != 0 {
		tw.SentenceEnd = func(_ int) {

			// Add end position of last token to sentence boundary
			// TODO: This only works if token positions are taking into account
			sent = append(sent, pos[len(pos)-1])
			sentB = true

			// Collect sentences also
			if flags&SENTENCES != 0 {
				writer.WriteByte('\n')
			}
		}

		// Collect sentence boundaries
	} else if flags&SENTENCES != 0 {
		tw.SentenceEnd = func(_ int) {
			writer.WriteByte('\n')
			writer.Flush()
		}

		// Ignore sentence boundaries
	} else {
		tw.SentenceEnd = func(_ int) {}
	}

	// Write token or sentence positions
	if flags&(TOKEN_POS|SENTENCE_POS) != 0 {
		tw.TextEnd = func(_ int) {

			// Write token positions
			if flags&TOKEN_POS != 0 {
				writer.WriteString(strconv.Itoa(pos[0]))
				for _, x := range pos[1:] {
					writer.WriteByte(' ')
					writer.WriteString(strconv.Itoa(x))
				}
				writer.WriteByte('\n')
			}

			// Write sentence positions
			if flags&SENTENCE_POS != 0 {
				writer.WriteString(strconv.Itoa(sent[0]))
				for _, x := range sent[1:] {
					writer.WriteByte(' ')
					writer.WriteString(strconv.Itoa(x))
				}
				writer.WriteByte('\n')
				sent = sent[:0]
				sentB = true
			}

			writer.Flush()

			posC = 0
			pos = pos[:0]
		}

		// Collect text ends
	} else {
		tw.TextEnd = func(_ int) {
			writer.WriteByte('\n')
			writer.Flush()
		}
	}

	// Flush the writer
	tw.Flush = func() error {
		return writer.Flush()
	}

	return tw
}

package datok

import (
	"bufio"
	"io"
)

type TokenWriterI interface {
	SentenceEnd(int)
	TextEnd(int)
	Token(int, []rune)
	Flush() error
}

var _ TokenWriterI = &TokenWriterSimple{}

type TokenWriterSimple struct {
	writer *bufio.Writer
}

func NewTokenWriterSimple(w io.Writer) *TokenWriterSimple {
	return &TokenWriterSimple{bufio.NewWriter(w)}
}

func (tw *TokenWriterSimple) SentenceEnd(_ int) {
	tw.writer.WriteRune('\n')
}

func (tw *TokenWriterSimple) TextEnd(_ int) {
	tw.writer.WriteRune('\n')
	tw.writer.Flush()
}

func (tw *TokenWriterSimple) Token(offset int, buf []rune) {
	tw.writer.WriteString(string(buf[offset:]))
	tw.writer.WriteRune('\n')
}

func (tw *TokenWriterSimple) Flush() error {
	return tw.writer.Flush()
}

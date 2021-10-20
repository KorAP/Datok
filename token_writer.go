package datok

import (
	"bufio"
	"io"
)

type TokenWriterI interface {
	SentenceEnd()
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

func (tw *TokenWriterSimple) SentenceEnd() {
	tw.writer.WriteRune('\n')
}

func (tw *TokenWriterSimple) Token(_ int, buf []rune) {
	tw.writer.WriteString(string(buf))
	tw.writer.WriteRune('\n')
}

func (tw *TokenWriterSimple) Flush() error {
	return tw.writer.Flush()
}

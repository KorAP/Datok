package datok

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenWriterSimple(t *testing.T) {
	assert := assert.New(t)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)

	tws := NewTokenWriter(w)

	assert.NotNil(tws)

	tws.Token(0, []rune{'a', 'b', 'c'})

	tws.Token(1, []rune{'d', 'e', 'f'})

	tws.SentenceEnd(0)

	tws.TextEnd(0)

	tws.Flush()

	assert.Equal("abc\nef\n\n\n", w.String())
}

func TestTokenWriterFromOptions(t *testing.T) {
	assert := assert.New(t)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)

	tws := NewTokenWriterFromOptions(w, true, true, false)

	mat := LoadMatrixFile("testdata/tokenizer.matok")

	assert.NotNil(mat)

	assert.True(mat.TransduceTokenWriter(
		strings.NewReader("This.\x0a\x04And.\n\x04\n"), tws),
	)

	matStr := w.String()
	assert.Equal("This\n.\n\n0 4 4 5\nAnd\n.\n\n0 3 3 4\n", matStr)

	w.Reset()
	mat.TransduceTokenWriter(strings.NewReader("\nThis.\x0a\x04\nAnd.\n\x04\n"), tws)

	matStr = w.String()
	assert.Equal("This\n.\n\n1 5 5 6\nAnd\n.\n\n1 4 4 5\n", matStr)

	//
	// Accept newline after EOT
	tws = NewTokenWriterFromOptions(w, true, true, true)

	w.Reset()
	mat.TransduceTokenWriter(strings.NewReader("\nThis.\x0a\x04\nAnd.\n\x04\n"), tws)

	matStr = w.String()
	assert.Equal("This\n.\n\n1 5 5 6\nAnd\n.\n\n0 3 3 4\n", matStr)

	//
	// Write no tokens
	tws = NewTokenWriterFromOptions(w, true, false, true)

	w.Reset()
	mat.TransduceTokenWriter(strings.NewReader("\nThis.\x0a\x04\nAnd.\n\x04\n"), tws)

	matStr = w.String()
	assert.Equal("\n1 5 5 6\n\n0 3 3 4\n", matStr)
}

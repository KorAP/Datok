package datok

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenWriterSimple(t *testing.T) {
	assert := assert.New(t)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)

	tws := NewTokenWriterSimple(w)

	assert.NotNil(tws)

	tws.Token(0, []rune{'a', 'b', 'c'})

	tws.Token(0, []rune{'d', 'e', 'f'})

	tws.SentenceEnd(0)

	tws.TextEnd(0)

	tws.Flush()

	assert.Equal("abc\ndef\n\n\n", w.String())
}

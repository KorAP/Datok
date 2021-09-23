package datok

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFullTokenizerMatrix(t *testing.T) {
	assert := assert.New(t)
	foma := LoadFomaFile("testdata/simpletok.fst")
	assert.NotNil(foma)

	mat := foma.ToMatrix()

	r := strings.NewReader("  wald   gehen Da kann\t man was \"erleben\"!")
	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string
	mat.Transduce(r, w)
	tokens = strings.Split(w.String(), "\n")
	assert.Equal(len(tokens), 9)
	assert.Equal("wald", tokens[0])
	assert.Equal("gehen", tokens[1])
	assert.Equal("Da", tokens[2])
	assert.Equal("kann", tokens[3])
	assert.Equal("man", tokens[4])
	assert.Equal("was", tokens[5])
	assert.Equal("\"erleben\"", tokens[6])
	assert.Equal("!", tokens[7])
}

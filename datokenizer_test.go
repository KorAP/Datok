package datokenizer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleString(t *testing.T) {
	assert := assert.New(t)

	// bau | bauamt
	tok := ParseFile("testdata/bauamt.fst")
	tok.ToDoubleArray()
	assert.True(tok.Match("bau"))
	assert.True(tok.Match("bauamt"))
	assert.False(tok.Match("baum"))
}

func TestSimpleBranches(t *testing.T) {
	assert := assert.New(t)

	// (bau | wahl) (amt | en)
	tok := ParseFile("testdata/wahlamt.fst")
	tok.ToDoubleArray()
	assert.False(tok.Match("bau"))
	assert.True(tok.Match("bauamt"))
	assert.True(tok.Match("wahlamt"))
	assert.True(tok.Match("bauen"))
	assert.True(tok.Match("wahlen"))
	assert.False(tok.Match("baum"))
}

func TestSimpleTokenizer(t *testing.T) {
	assert := assert.New(t)
	tok := ParseFile("testdata/simpletok.fst")
	tok.ToDoubleArray()
	assert.True(tok.Match("bau"))
	assert.True(tok.Match("bad"))
	assert.True(tok.Match("wald gehen"))
}

func TestFullTokenizer(t *testing.T) {
	assert := assert.New(t)
	tok := ParseFile("testdata/tokenizer.fst")
	tok.ToDoubleArray()
	fmt.Println("Size:", tok.maxSize)
	assert.True(tok.Match("bau"))
	assert.True(tok.Match("bad"))
	assert.True(tok.Match("wald gehen"))
}

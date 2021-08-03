package datokenizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleString(t *testing.T) {
	assert := assert.New(t)

	// bau | bauamt
	tok := parse_file("testdata/bauamt.fst")
	tok.buildDA()
	assert.True(tok.match("bau"))
	assert.True(tok.match("bauamt"))
	assert.False(tok.match("baum"))
}

func TestSimpleBranches(t *testing.T) {
	assert := assert.New(t)

	// (bau | wahl) (amt | en)
	tok := parse_file("testdata/wahlamt.fst")
	tok.buildDA()
	assert.False(tok.match("bau"))
	assert.True(tok.match("bauamt"))
	assert.True(tok.match("wahlamt"))
	assert.True(tok.match("bauen"))
	assert.True(tok.match("wahlen"))
	assert.False(tok.match("baum"))
}

func TestSimpleTokenizer(t *testing.T) {
	assert := assert.New(t)
	tok := parse_file("testdata/simpletok.fst")
	tok.buildDA()
	assert.True(tok.match("bau"))
	assert.True(tok.match("bad"))
	assert.True(tok.match("wald gehen"))
}

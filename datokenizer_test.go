package datokenizer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleString(t *testing.T) {
	assert := assert.New(t)

	// bau | bauamt
	tok := LoadFomaFile("testdata/bauamt.fst")
	dat := tok.ToDoubleArray()
	assert.True(dat.Match("bau"))
	assert.True(dat.Match("bauamt"))
	assert.False(dat.Match("baum"))
}

func TestSimpleBranches(t *testing.T) {
	assert := assert.New(t)

	// (bau | wahl) (amt | en)
	tok := LoadFomaFile("testdata/wahlamt.fst")
	dat := tok.ToDoubleArray()
	assert.False(dat.Match("bau"))
	assert.True(dat.Match("bauamt"))
	assert.True(dat.Match("wahlamt"))
	assert.True(dat.Match("bauen"))
	assert.True(dat.Match("wahlen"))
	assert.False(dat.Match("baum"))
}

func TestSimpleTokenizer(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/simpletok.fst")
	dat := tok.ToDoubleArray()
	assert.True(dat.Match("bau"))
	assert.True(dat.Match("bad"))
	assert.True(dat.Match("wald gehen"))
}

func TestWriteTokenizer(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/simpletok.fst")
	dat := tok.ToDoubleArray()
	assert.True(dat.Match("bau"))
	assert.True(dat.Match("bad"))
	assert.True(dat.Match("wald gehen"))

	assert.True(dat.LoadLevel() >= 70)

	b := make([]byte, 1024)
	buf := bytes.NewBuffer(b)
	n, err := dat.WriteTo(buf)
	assert.Nil(err)
	assert.Equal(n, int64(186))
}

func TestFullTokenizer(t *testing.T) {
	/*
		assert := assert.New(t)
		tok := ParseFile("testdata/tokenizer.fst")
		dat := tok.ToDoubleArray()
		assert.True(dat.LoadLevel() >= 70)
		assert.True(dat.Match("bau"))
		assert.True(dat.Match("bad"))
		assert.True(dat.Match("wald gehen"))
	*/
}

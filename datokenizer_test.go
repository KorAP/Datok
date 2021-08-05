package datokenizer

import (
	"bytes"
	"strings"
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

func TestSimpleTokenizerTransduce(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/simpletok.fst")
	dat := tok.ToDoubleArray()

	r := strings.NewReader("  wald   gehen Da kann\t man was \"erleben\"!")
	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	dat.Transduce(r, w)
	tokens = strings.Split(w.String(), "\n")
	assert.Equal("wald", tokens[0])
	assert.Equal("gehen", tokens[1])
	assert.Equal("Da", tokens[2])
	assert.Equal("kann", tokens[3])
	assert.Equal("man", tokens[4])
	assert.Equal("was", tokens[5])
	assert.Equal("\"erleben\"", tokens[6])

	r = strings.NewReader(" In den Wald gehen? -- Da kann\t man was \"erleben\"!")
	w.Reset()
	dat.Transduce(r, w)
	tokens = strings.Split(w.String(), "\n")
	assert.Equal("In", tokens[0])
	assert.Equal("den", tokens[1])
	assert.Equal("Wald", tokens[2])
	assert.Equal("gehen", tokens[3])
	assert.Equal("?", tokens[4])
	assert.Equal("--", tokens[5])

	r = strings.NewReader(" g? -- D")
	w.Reset()
	dat.Transduce(r, w)
	tokens = strings.Split(w.String(), "\n")
	assert.Equal("g", tokens[0])
	assert.Equal("?", tokens[1])
	assert.Equal("--", tokens[2])
	assert.Equal("D", tokens[3])
	assert.Equal("", tokens[4])
	assert.Equal("", tokens[5])
	assert.Equal(6, len(tokens))
}

func TestReadWriteTokenizer(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/simpletok.fst")
	dat := tok.ToDoubleArray()
	assert.True(dat.Match("bau"))
	assert.True(dat.Match("bad"))
	assert.True(dat.Match("wald gehen"))

	assert.True(dat.LoadFactor() >= 70)

	b := make([]byte, 0, 1024)
	buf := bytes.NewBuffer(b)
	n, err := dat.WriteTo(buf)
	assert.Nil(err)
	assert.Equal(int64(218), n)

	dat2 := ParseDatok(buf)
	assert.NotNil(dat2)
	assert.Equal(dat.array, dat2.array)
	assert.Equal(dat.sigma, dat2.sigma)
	assert.Equal(dat.epsilon, dat2.epsilon)
	assert.Equal(dat.unknown, dat2.unknown)
	assert.Equal(dat.identity, dat2.identity)
	assert.Equal(dat.final, dat2.final)
	assert.Equal(dat.LoadFactor(), dat2.LoadFactor())
	assert.True(dat2.Match("bau"))
	assert.True(dat2.Match("bad"))
	assert.True(dat2.Match("wald gehen"))
}

func TestFullTokenizer(t *testing.T) {
	/*
		assert := assert.New(t)
		tok := LoadFomaFile("testdata/tokenizer.fst")
		dat := tok.ToDoubleArray()

		f, _ := os.Create("testdata/tokenizer.datok")
		gz := gzip.NewWriter(f)
		defer f.Close()
		dat.WriteTo(gz)
		assert.NotNil(gz)

		assert.True(dat.LoadFactor() >= 70)
		assert.True(dat.Match("bau"))
		assert.True(dat.Match("bad"))
		assert.True(dat.Match("wald gehen"))
	*/
}

func TestFullTokenizerTransduce(t *testing.T) {
	/*
		assert := assert.New(t)
		// tok := LoadFomaFile("testdata/tokenizer.fst")
		tok := LoadFomaFile("testdata/simpletok.fst")
		dat := tok.ToDoubleArray()

		dat := LoadDatokFile("testdata/tokenizer.datok")
		r := strings.NewReader("wald   gehen! Da kann\t man was \"erleben\"!")
		assert.True(dat.Transduce(r, os.Stdout))

		assert.Fail("!")
	*/
}

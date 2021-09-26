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
	assert.Equal(len(tokens), 10)
	assert.Equal("wald", tokens[0])
	assert.Equal("gehen", tokens[1])
	assert.Equal("Da", tokens[2])
	assert.Equal("kann", tokens[3])
	assert.Equal("man", tokens[4])
	assert.Equal("was", tokens[5])
	assert.Equal("\"erleben\"", tokens[6])
	assert.Equal("!", tokens[7])

	r = strings.NewReader(" In den Wald gehen? -- Da kann\t man was \"erleben\"!")
	w.Reset()
	mat.Transduce(r, w)
	tokens = strings.Split(w.String(), "\n")
	assert.Equal("In", tokens[0])
	assert.Equal("den", tokens[1])
	assert.Equal("Wald", tokens[2])
	assert.Equal("gehen", tokens[3])
	assert.Equal("?", tokens[4])
	assert.Equal("--", tokens[5])

	r = strings.NewReader(" g? -- D")
	w.Reset()
	mat.Transduce(r, w)
	tokens = strings.Split(w.String(), "\n")
	assert.Equal("g", tokens[0])
	assert.Equal("?", tokens[1])
	assert.Equal("--", tokens[2])
	assert.Equal("D", tokens[3])
	assert.Equal("", tokens[4])
	assert.Equal("", tokens[5])
	assert.Equal(6, len(tokens))
}

func TestReadWriteMatrixTokenizer(t *testing.T) {
	assert := assert.New(t)
	foma := LoadFomaFile("testdata/simpletok.fst")
	assert.NotNil(foma)

	mat := foma.ToMatrix()
	assert.NotNil(foma)

	assert.True(tmatch(mat, "bau"))
	assert.True(tmatch(mat, "bad"))
	assert.True(tmatch(mat, "wald gehen"))
	b := make([]byte, 0, 1024)
	buf := bytes.NewBuffer(b)
	n, err := mat.WriteTo(buf)
	assert.Nil(err)
	assert.Equal(int64(248), n)
	mat2 := ParseMatrix(buf)
	assert.NotNil(mat2)
	assert.Equal(mat.sigma, mat2.sigma)
	assert.Equal(mat.epsilon, mat2.epsilon)
	assert.Equal(mat.unknown, mat2.unknown)
	assert.Equal(mat.identity, mat2.identity)
	assert.Equal(mat.stateCount, mat2.stateCount)
	assert.Equal(len(mat.array), len(mat2.array))
	assert.Equal(mat.array, mat2.array)
	assert.True(tmatch(mat2, "bau"))
	assert.True(tmatch(mat2, "bad"))
	assert.True(tmatch(mat2, "wald gehen"))
}

func TestFullTokenizerMatrixSentenceSplitter(t *testing.T) {
	assert := assert.New(t)
	foma := LoadFomaFile("testdata/tokenizer.fst")
	assert.NotNil(foma)

	mat := foma.ToMatrix()

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var sentences []string

	// testSentSplitterSimple
	assert.True(mat.Transduce(strings.NewReader("Der alte Mann."), w))
	sentences = strings.Split(w.String(), "\n\n")

	assert.Equal("Der\nalte\nMann\n.\n\n", w.String())
	assert.Equal("Der\nalte\nMann\n.", sentences[0])
	assert.Equal("", sentences[1])
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat.Transduce(strings.NewReader("Der Vorsitzende der Abk. hat gewählt."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)
	assert.Equal("Der\nVorsitzende\nder\nAbk.\nhat\ngewählt\n.", sentences[0])
	assert.Equal("", sentences[1])

	/*

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader(""), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal(len(sentences), 1)
		assert.Equal("\n", sentences[0])

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Gefunden auf wikipedia.org."), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal(len(sentences), 2)

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Ich bin unter korap@ids-mannheim.de erreichbar."), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal(len(sentences), 2)

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Unsere Website ist https://korap.ids-mannheim.de/?q=Baum"), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal("Unsere\nWebsite\nist\nhttps://korap.ids-mannheim.de/?q=Baum", sentences[0])
		assert.Equal("", sentences[1])
		assert.Equal(len(sentences), 2)

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Unser Server ist 10.0.10.51."), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal("", sentences[1])
		assert.Equal(len(sentences), 2)

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Zu 50.4% ist es sicher"), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal(len(sentences), 2)

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Der Termin ist am 5.9.2018"), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal(len(sentences), 2)

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Ich habe die readme.txt heruntergeladen"), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal(len(sentences), 2)
		assert.Equal("Ich\nhabe\ndie\nreadme.txt\nheruntergeladen", sentences[0])
		assert.Equal("", sentences[1])

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Ausschalten!!! Hast Du nicht gehört???"), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal(len(sentences), 3)
		assert.Equal("Ausschalten\n!!!", sentences[0])
		assert.Equal("Hast\nDu\nnicht\ngehört\n???", sentences[1])
		assert.Equal("", sentences[2])

		w.Reset()
		assert.True(mat.Transduce(strings.NewReader("Ich wohne in der Weststr. und Du?"), w))
		sentences = strings.Split(w.String(), "\n\n")
		assert.Equal(len(sentences), 2)
	*/
	/*
		Test:
		"\"Ausschalten!!!\", sagte er. \"Hast Du nicht gehört???\""), w))
	*/
}

package datok

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var dat *DaTokenizer

func tmatch(tok Tokenizer, s string) bool {
	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	return tok.Transduce(strings.NewReader(s), w)
}

func ttokenize(tok Tokenizer, w *bytes.Buffer, str string) []string {
	w.Reset()
	ok := tok.Transduce(strings.NewReader(str), w)
	if !ok {
		return []string{}
	}
	obj := regexp.MustCompile("\n+")

	tokens := obj.Split(w.String(), -1)
	return tokens[:len(tokens)-1]
}

func TestDoubleArraySimpleString(t *testing.T) {
	assert := assert.New(t)

	// bau | bauamt
	tok := LoadFomaFile("testdata/bauamt.fst")
	dat := tok.ToDoubleArray()
	assert.True(tmatch(dat, "bau"))
	assert.True(tmatch(dat, "bauamt"))
	assert.False(tmatch(dat, "baum"))
	assert.True(tmatch(dat, "baua"))
}

func TestDoubleArraySimpleBranches(t *testing.T) {
	assert := assert.New(t)

	// (bau | wahl) (amt | en)
	tok := LoadFomaFile("testdata/wahlamt.fst")
	dat := tok.ToDoubleArray()
	assert.True(tmatch(dat, "bau"))
	assert.True(tmatch(dat, "bauamt"))
	assert.True(tmatch(dat, "wahlamt"))
	assert.True(tmatch(dat, "bauen"))
	assert.True(tmatch(dat, "wahlen"))
	assert.False(tmatch(dat, "baum"))
}

func TestSimpleTokenizer(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/simpletok.fst")
	dat := tok.ToDoubleArray()
	assert.True(tmatch(dat, "bau"))
	assert.True(tmatch(dat, "bad"))
	assert.True(tmatch(dat, "wald gehen"))
}

func TestDoubleArraySimpleTokenizerTransduce(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/simpletok.fst")
	dat := tok.ToDoubleArray()

	r := strings.NewReader("  wald   gehen Da kann\t man was \"erleben\"!")
	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string
	dat.Transduce(r, w)
	tokens = strings.Split(w.String(), "\n")
	assert.Equal(len(tokens), 11)
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
	assert.Equal(7, len(tokens))
}

func TestDoubleArrayReadWriteTokenizer(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/simpletok.fst")
	dat := tok.ToDoubleArray()
	assert.True(tmatch(dat, "bau"))
	assert.True(tmatch(dat, "bad"))
	assert.True(tmatch(dat, "wald gehen"))

	b := make([]byte, 0, 1024)
	buf := bytes.NewBuffer(b)
	n, err := dat.WriteTo(buf)
	assert.Nil(err)
	assert.Equal(int64(296), n)

	dat2 := ParseDatok(buf)
	assert.NotNil(dat2)
	assert.Equal(dat.array, dat2.array)
	assert.Equal(dat.sigma, dat2.sigma)
	assert.Equal(dat.epsilon, dat2.epsilon)
	assert.Equal(dat.unknown, dat2.unknown)
	assert.Equal(dat.identity, dat2.identity)
	assert.Equal(dat.final, dat2.final)
	assert.Equal(dat.LoadFactor(), dat2.LoadFactor())
	assert.True(tmatch(dat2, "bau"))
	assert.True(tmatch(dat2, "bad"))
	assert.True(tmatch(dat2, "wald gehen"))

	assert.Equal(dat.TransCount(), 17)
	assert.Equal(dat2.TransCount(), 17)
}

func TestDoubleArrayIgnorableMCS(t *testing.T) {

	// This test relies on final states. That's why it is
	// not working correctly anymore.

	assert := assert.New(t)
	// File has MCS in sigma but not in net
	tok := LoadFomaFile("testdata/ignorable_mcs.fst")
	assert.NotNil(tok)
	dat := tok.ToDoubleArray()
	assert.NotNil(dat)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	// Is only unambigous when transducing strictly greedy!
	assert.True(dat.Transduce(strings.NewReader("ab<ab>a"), w))
	tokens = strings.Split(w.String(), "\n")
	assert.Equal("a\nb\n<ab>a\n\n\n", w.String())
	assert.Equal("a", tokens[0])
	assert.Equal("b", tokens[1])
	assert.Equal("<ab>a", tokens[2])
	assert.Equal(6, len(tokens))
	assert.Equal(dat.TransCount(), 15)
}

func TestDoubleArrayFullTokenizer(t *testing.T) {
	assert := assert.New(t)

	if dat == nil {
		dat = LoadDatokFile("testdata/tokenizer.datok")
	}
	assert.NotNil(dat)
	assert.True(dat.LoadFactor() >= 70)
	assert.Equal(dat.epsilon, 1)
	assert.Equal(dat.unknown, 2)
	assert.Equal(dat.identity, 3)
	assert.Equal(dat.final, 142)
	assert.Equal(len(dat.sigma), 137)
	// assert.True(len(dat.array) > 3000000)
	// assert.True(dat.maxSize > 3000000)
	assert.True(tmatch(dat, "bau"))
	assert.True(tmatch(dat, "bad"))
	assert.True(tmatch(dat, "wald gehen"))
}

func TestDoubleArrayTokenizerBranch(t *testing.T) {
	assert := assert.New(t)
	tok := LoadTokenizerFile("testdata/simpletok.datok")
	assert.NotNil(tok)
	assert.Equal(tok.Type(), "DATOK")

	tok = LoadTokenizerFile("testdata/simpletok.matok")
	assert.NotNil(tok)
	assert.Equal(tok.Type(), "MATOK")
}

func XTestDoubleArrayFullTokenizerBuild(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/tokenizer.fst")
	dat := tok.ToDoubleArray()
	assert.NotNil(dat)
	// n, err := dat.Save("testdata/tokenizer.datok")
	// assert.Nil(err)
	// assert.True(n > 500)
}

func TestDoubleArrayFullTokenizerTransduce(t *testing.T) {
	assert := assert.New(t)

	if dat == nil {
		dat = LoadDatokFile("testdata/tokenizer.datok")
	}

	assert.NotNil(dat)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	assert.True(dat.Transduce(strings.NewReader("tra. u Du?"), w))

	tokens = strings.Split(w.String(), "\n")
	assert.Equal("tra\n.\n\nu\nDu\n?\n\n\n", w.String())
	assert.Equal("tra", tokens[0])
	assert.Equal(".", tokens[1])
	assert.Equal("", tokens[2])
	assert.Equal("u", tokens[3])
	assert.Equal("Du", tokens[4])
	assert.Equal("?", tokens[5])
	assert.Equal("", tokens[6])
	assert.Equal("", tokens[7])
	assert.Equal("", tokens[8])
	assert.Equal(9, len(tokens))

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("\"John Doe\"@xx.com"), w))
	assert.Equal("\"\nJohn\nDoe\n\"\n@xx\n.\n\ncom\n\n\n", w.String())
}

func TestDoubleArrayFullTokenizerSentenceSplitter(t *testing.T) {
	assert := assert.New(t)

	if dat == nil {
		dat = LoadDatokFile("testdata/tokenizer.datok")
	}

	assert.NotNil(dat)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var sentences []string

	// testSentSplitterSimple
	assert.True(dat.Transduce(strings.NewReader("Der alte Mann."), w))
	sentences = strings.Split(w.String(), "\n\n")

	assert.Equal("Der\nalte\nMann\n.\n\n\n", w.String())
	assert.Equal("Der\nalte\nMann\n.", sentences[0])
	assert.Equal("\n", sentences[1])
	assert.Equal(2, len(sentences))

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Der Vorsitzende der Abk. hat gewählt."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(2, len(sentences))
	assert.Equal("Der\nVorsitzende\nder\nAbk.\nhat\ngewählt\n.", sentences[0])
	assert.Equal("\n", sentences[1])

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader(""), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(2, len(sentences))
	assert.Equal("", sentences[0])

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Gefunden auf wikipedia.org."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Ich bin unter korap@ids-mannheim.de erreichbar."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Unsere Website ist https://korap.ids-mannheim.de/?q=Baum"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal("Unsere\nWebsite\nist\nhttps://korap.ids-mannheim.de/?q=Baum", sentences[0])
	assert.Equal("\n", sentences[1])
	assert.Equal(2, len(sentences))

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Unser Server ist 10.0.10.51."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal("\n", sentences[1])
	assert.Equal(2, len(sentences))

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Zu 50.4% ist es sicher"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Der Termin ist am 5.9.2018"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Ich habe die readme.txt heruntergeladen"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(2, len(sentences))
	assert.Equal("Ich\nhabe\ndie\nreadme.txt\nheruntergeladen", sentences[0])
	assert.Equal("\n", sentences[1])

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Ausschalten!!! Hast Du nicht gehört???"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(3, len(sentences))
	assert.Equal("Ausschalten\n!!!", sentences[0])
	assert.Equal("Hast\nDu\nnicht\ngehört\n???", sentences[1])
	assert.Equal("\n", sentences[2])

	w.Reset()
	assert.True(dat.Transduce(strings.NewReader("Ich wohne in der Weststr. und Du?"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	/*
		Test:
		"\"Ausschalten!!!\", sagte er. \"Hast Du nicht gehört???\""), w))
	*/
}

func TestDoubleArrayFullTokenizerTokenSplitter(t *testing.T) {
	assert := assert.New(t)

	if dat == nil {
		dat = LoadDatokFile("testdata/tokenizer.datok")
	}

	assert.NotNil(dat)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	// testTokenizerSimple
	tokens = ttokenize(dat, w, "Der alte Mann")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "alte")
	assert.Equal(tokens[2], "Mann")
	assert.Equal(len(tokens), 3)

	tokens = ttokenize(dat, w, "Der alte Mann.")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "alte")
	assert.Equal(tokens[2], "Mann")
	assert.Equal(tokens[3], ".")
	assert.Equal(len(tokens), 4)

	// testTokenizerAbbr
	tokens = ttokenize(dat, w, "Der Vorsitzende der F.D.P. hat gewählt")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "Vorsitzende")
	assert.Equal(tokens[2], "der")
	assert.Equal(tokens[3], "F.D.P.")
	assert.Equal(tokens[4], "hat")
	assert.Equal(tokens[5], "gewählt")
	assert.Equal(len(tokens), 6)
	// Ignored in KorAP-Tokenizer

	// testTokenizerHost1
	tokens = ttokenize(dat, w, "Gefunden auf wikipedia.org")
	assert.Equal(tokens[0], "Gefunden")
	assert.Equal(tokens[1], "auf")
	assert.Equal(tokens[2], "wikipedia.org")
	assert.Equal(len(tokens), 3)

	// testTokenizerWwwHost
	tokens = ttokenize(dat, w, "Gefunden auf www.wikipedia.org")
	assert.Equal("Gefunden", tokens[0])
	assert.Equal("auf", tokens[1])
	assert.Equal("www.wikipedia.org", tokens[2])
	assert.Equal(3, len(tokens))

	// testTokenizerWwwUrl
	tokens = ttokenize(dat, w, "Weitere Infos unter www.info.biz/info")
	assert.Equal("www.info.biz/info", tokens[3])

	// testTokenizerFtpHost
	/*
		tokens = tokenize(dat, w, "Kann von ftp.download.org heruntergeladen werden")
		assert.Equal("Kann", tokens[0])
		assert.Equal("von", tokens[1])
		assert.Equal("ftp.download.org", tokens[2])
		assert.Equal(5, len(tokens))
		// Ignored in KorAP-Tokenizer
	*/

	// testTokenizerDash
	tokens = ttokenize(dat, w, "Das war -- spitze")
	assert.Equal(tokens[0], "Das")
	assert.Equal(tokens[1], "war")
	assert.Equal(tokens[2], "--")
	assert.Equal(tokens[3], "spitze")
	assert.Equal(len(tokens), 4)

	// testTokenizerEmail1
	tokens = ttokenize(dat, w, "Ich bin unter korap@ids-mannheim.de erreichbar.")
	assert.Equal(tokens[0], "Ich")
	assert.Equal(tokens[1], "bin")
	assert.Equal(tokens[2], "unter")
	assert.Equal(tokens[3], "korap@ids-mannheim.de")
	assert.Equal(tokens[4], "erreichbar")
	assert.Equal(tokens[5], ".")
	assert.Equal(len(tokens), 6)

	// testTokenizerEmail2
	tokens = ttokenize(dat, w, "Oder unter korap[at]ids-mannheim[dot]de.")
	assert.Equal(tokens[0], "Oder")
	assert.Equal(tokens[1], "unter")
	assert.Equal(tokens[2], "korap[at]ids-mannheim[dot]de")
	assert.Equal(tokens[3], ".")
	assert.Equal(len(tokens), 4)

	// testTokenizerEmail3
	tokens = ttokenize(dat, w, "Oder unter korap(at)ids-mannheim(dot)de.")
	assert.Equal(tokens[0], "Oder")
	assert.Equal(tokens[1], "unter")
	assert.Equal(tokens[2], "korap(at)ids-mannheim(dot)de")
	assert.Equal(tokens[3], ".")
	assert.Equal(len(tokens), 4)
	// Ignored in KorAP-Tokenizer

	// testTokenizerDoNotAcceptQuotedEmailNames
	tokens = ttokenize(dat, w, "\"John Doe\"@xx.com")
	assert.Equal("\"", tokens[0])
	assert.Equal("John", tokens[1])
	assert.Equal("Doe", tokens[2])
	assert.Equal("\"", tokens[3])
	assert.Equal("@xx", tokens[4])
	assert.Equal(".", tokens[5]) // Differs - as the sentence splitter splits here!
	assert.Equal("com", tokens[6])
	assert.Equal(7, len(tokens))

	// testTokenizerTwitter
	tokens = ttokenize(dat, w, "Folgt @korap und #korap")
	assert.Equal(tokens[0], "Folgt")
	assert.Equal(tokens[1], "@korap")
	assert.Equal(tokens[2], "und")
	assert.Equal(tokens[3], "#korap")
	assert.Equal(len(tokens), 4)

	// testTokenizerWeb1
	tokens = ttokenize(dat, w, "Unsere Website ist https://korap.ids-mannheim.de/?q=Baum")
	assert.Equal(tokens[0], "Unsere")
	assert.Equal(tokens[1], "Website")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "https://korap.ids-mannheim.de/?q=Baum")
	assert.Equal(len(tokens), 4)

	// testTokenizerWeb2
	tokens = ttokenize(dat, w, "Wir sind auch im Internet (https://korap.ids-mannheim.de/?q=Baum)")
	assert.Equal(tokens[0], "Wir")
	assert.Equal(tokens[1], "sind")
	assert.Equal(tokens[2], "auch")
	assert.Equal(tokens[3], "im")
	assert.Equal(tokens[4], "Internet")
	assert.Equal(tokens[5], "(")
	assert.Equal(tokens[6], "https://korap.ids-mannheim.de/?q=Baum")
	assert.Equal(tokens[7], ")")
	assert.Equal(len(tokens), 8)
	// Ignored in KorAP-Tokenizer

	// testTokenizerWeb3
	tokens = ttokenize(dat, w, "Die Adresse ist https://korap.ids-mannheim.de/?q=Baum.")
	assert.Equal(tokens[0], "Die")
	assert.Equal(tokens[1], "Adresse")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "https://korap.ids-mannheim.de/?q=Baum")
	assert.Equal(tokens[4], ".")
	assert.Equal(len(tokens), 5)
	// Ignored in KorAP-Tokenizer

	// testTokenizerServer
	tokens = ttokenize(dat, w, "Unser Server ist 10.0.10.51.")
	assert.Equal(tokens[0], "Unser")
	assert.Equal(tokens[1], "Server")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "10.0.10.51")
	assert.Equal(tokens[4], ".")
	assert.Equal(len(tokens), 5)

	// testTokenizerNum
	tokens = ttokenize(dat, w, "Zu 50,4% ist es sicher")
	assert.Equal(tokens[0], "Zu")
	assert.Equal(tokens[1], "50,4%")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "es")
	assert.Equal(tokens[4], "sicher")
	assert.Equal(len(tokens), 5)
	// Differs from KorAP-Tokenizer

	// testTokenizerDate
	tokens = ttokenize(dat, w, "Der Termin ist am 5.9.2018")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "Termin")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "am")
	assert.Equal(tokens[4], "5.9.2018")
	assert.Equal(len(tokens), 5)

	tokens = ttokenize(dat, w, "Der Termin ist am 5/9/2018")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "Termin")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "am")
	assert.Equal(tokens[4], "5/9/2018")
	assert.Equal(len(tokens), 5)

	// testTokenizerDateRange
	/*
		tokens = tokenize(dat, w, "Der Termin war vom 4.-5.9.2018")
		assert.Equal(tokens[0], "Der")
		assert.Equal(tokens[1], "Termin")
		assert.Equal(tokens[2], "war")
		assert.Equal(tokens[3], "vom")
		assert.Equal(tokens[4], "4.")
		assert.Equal(tokens[5], "-")
		assert.Equal(tokens[6], "5.9.2018")
		assert.Equal(len(tokens), 7)
		// Ignored in KorAP-Tokenizer
	*/

	// testTokenizerEmoji1
	tokens = ttokenize(dat, w, "Das ist toll! ;)")
	assert.Equal(tokens[0], "Das")
	assert.Equal(tokens[1], "ist")
	assert.Equal(tokens[2], "toll")
	assert.Equal(tokens[3], "!")
	assert.Equal(tokens[4], ";)")
	assert.Equal(len(tokens), 5)

	// testTokenizerRef1
	tokens = ttokenize(dat, w, "Kupietz und Schmidt (2018): Korpuslinguistik")
	assert.Equal(tokens[0], "Kupietz")
	assert.Equal(tokens[1], "und")
	assert.Equal(tokens[2], "Schmidt")
	assert.Equal(tokens[3], "(2018)")
	assert.Equal(tokens[4], ":")
	assert.Equal(tokens[5], "Korpuslinguistik")
	assert.Equal(len(tokens), 6)
	// Differs from KorAP-Tokenizer!

	// testTokenizerRef2 () {
	tokens = ttokenize(dat, w, "Kupietz und Schmidt [2018]: Korpuslinguistik")
	assert.Equal(tokens[0], "Kupietz")
	assert.Equal(tokens[1], "und")
	assert.Equal(tokens[2], "Schmidt")
	assert.Equal(tokens[3], "[2018]")
	assert.Equal(tokens[4], ":")
	assert.Equal(tokens[5], "Korpuslinguistik")
	assert.Equal(len(tokens), 6)
	// Differs from KorAP-Tokenizer!

	// testTokenizerOmission1 () {
	tokens = ttokenize(dat, w, "Er ist ein A****loch!")
	assert.Equal(tokens[0], "Er")
	assert.Equal(tokens[1], "ist")
	assert.Equal(tokens[2], "ein")
	assert.Equal(tokens[3], "A****loch")
	assert.Equal(tokens[4], "!")
	assert.Equal(len(tokens), 5)

	// testTokenizerOmission2
	tokens = ttokenize(dat, w, "F*ck!")
	assert.Equal(tokens[0], "F*ck")
	assert.Equal(tokens[1], "!")
	assert.Equal(len(tokens), 2)

	// testTokenizerOmission3 () {
	tokens = ttokenize(dat, w, "Dieses verf***** Kleid!")
	assert.Equal(tokens[0], "Dieses")
	assert.Equal(tokens[1], "verf*****")
	assert.Equal(tokens[2], "Kleid")
	assert.Equal(tokens[3], "!")
	assert.Equal(len(tokens), 4)

	// Probably interpreted as HOST
	// testTokenizerFileExtension1
	tokens = ttokenize(dat, w, "Ich habe die readme.txt heruntergeladen")
	assert.Equal(tokens[0], "Ich")
	assert.Equal(tokens[1], "habe")
	assert.Equal(tokens[2], "die")
	assert.Equal(tokens[3], "readme.txt")
	assert.Equal(tokens[4], "heruntergeladen")
	assert.Equal(len(tokens), 5)

	// Probably interpreted as HOST
	// testTokenizerFileExtension2
	tokens = ttokenize(dat, w, "Nimm die README.TXT!")
	assert.Equal(tokens[0], "Nimm")
	assert.Equal(tokens[1], "die")
	assert.Equal(tokens[2], "README.TXT")
	assert.Equal(tokens[3], "!")
	assert.Equal(len(tokens), 4)

	// Probably interpreted as HOST
	// testTokenizerFileExtension3
	tokens = ttokenize(dat, w, "Zeig mir profile.jpeg")
	assert.Equal(tokens[0], "Zeig")
	assert.Equal(tokens[1], "mir")
	assert.Equal(tokens[2], "profile.jpeg")
	assert.Equal(len(tokens), 3)

	// testTokenizerFile1

	tokens = ttokenize(dat, w, "Zeig mir c:\\Dokumente\\profile.docx")
	assert.Equal(tokens[0], "Zeig")
	assert.Equal(tokens[1], "mir")
	assert.Equal(tokens[2], "c:\\Dokumente\\profile.docx")
	assert.Equal(len(tokens), 3)

	// testTokenizerFile2
	tokens = ttokenize(dat, w, "Gehe zu /Dokumente/profile.docx")
	assert.Equal(tokens[0], "Gehe")
	assert.Equal(tokens[1], "zu")
	assert.Equal(tokens[2], "/Dokumente/profile.docx")
	assert.Equal(len(tokens), 3)

	// testTokenizerFile3
	tokens = ttokenize(dat, w, "Zeig mir c:\\Dokumente\\profile.jpeg")
	assert.Equal(tokens[0], "Zeig")
	assert.Equal(tokens[1], "mir")
	assert.Equal(tokens[2], "c:\\Dokumente\\profile.jpeg")
	assert.Equal(len(tokens), 3)
	// Ignored in KorAP-Tokenizer

	// testTokenizerPunct
	tokens = ttokenize(dat, w, "Er sagte: \"Es geht mir gut!\", daraufhin ging er.")
	assert.Equal(tokens[0], "Er")
	assert.Equal(tokens[1], "sagte")
	assert.Equal(tokens[2], ":")
	assert.Equal(tokens[3], "\"")
	assert.Equal(tokens[4], "Es")
	assert.Equal(tokens[5], "geht")
	assert.Equal(tokens[6], "mir")
	assert.Equal(tokens[7], "gut")
	assert.Equal(tokens[8], "!")
	assert.Equal(tokens[9], "\"")
	assert.Equal(tokens[10], ",")
	assert.Equal(tokens[11], "daraufhin")
	assert.Equal(tokens[12], "ging")
	assert.Equal(tokens[13], "er")
	assert.Equal(tokens[14], ".")
	assert.Equal(len(tokens), 15)

	// testTokenizerPlusAmpersand
	tokens = ttokenize(dat, w, "&quot;Das ist von C&A!&quot;")
	assert.Equal(tokens[0], "&quot;")
	assert.Equal(tokens[1], "Das")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "von")
	assert.Equal(tokens[4], "C&A")
	assert.Equal(tokens[5], "!")
	assert.Equal(tokens[6], "&quot;")
	assert.Equal(len(tokens), 7)

	// testTokenizerLongEnd
	tokens = ttokenize(dat, w, "Siehst Du?!!?")
	assert.Equal(tokens[0], "Siehst")
	assert.Equal(tokens[1], "Du")
	assert.Equal(tokens[2], "?!!?")
	assert.Equal(len(tokens), 3)

	// testTokenizerIrishO
	tokens = ttokenize(dat, w, "Peter O'Toole")
	assert.Equal(tokens[0], "Peter")
	assert.Equal(tokens[1], "O'Toole")
	assert.Equal(len(tokens), 2)

	// testTokenizerAbr
	tokens = ttokenize(dat, w, "Früher bzw. später ...")
	assert.Equal(tokens[0], "Früher")
	assert.Equal(tokens[1], "bzw.")
	assert.Equal(tokens[2], "später")
	assert.Equal(tokens[3], "...")
	assert.Equal(len(tokens), 4)

	// testTokenizerUppercaseRule
	tokens = ttokenize(dat, w, "Es war spät.Morgen ist es früh.")
	assert.Equal(tokens[0], "Es")
	assert.Equal(tokens[1], "war")
	assert.Equal(tokens[2], "spät")
	assert.Equal(tokens[3], ".")
	assert.Equal(tokens[4], "Morgen")
	assert.Equal(tokens[5], "ist")
	assert.Equal(tokens[6], "es")
	assert.Equal(tokens[7], "früh")
	assert.Equal(tokens[8], ".")
	assert.Equal(len(tokens), 9)
	// Ignored in KorAP-Tokenizer

	// testTokenizerOrd
	tokens = ttokenize(dat, w, "Sie erreichte den 1. Platz!")
	assert.Equal(tokens[0], "Sie")
	assert.Equal(tokens[1], "erreichte")
	assert.Equal(tokens[2], "den")
	assert.Equal(tokens[3], "1.")
	assert.Equal(tokens[4], "Platz")
	assert.Equal(tokens[5], "!")
	assert.Equal(len(tokens), 6)

	// testNoZipOuputArchive
	tokens = ttokenize(dat, w, "Archive:  Ich bin kein zip\n")
	assert.Equal(tokens[0], "Archive")
	assert.Equal(tokens[1], ":")
	assert.Equal(tokens[2], "Ich")
	assert.Equal(tokens[3], "bin")
	assert.Equal(tokens[4], "kein")
	assert.Equal(tokens[5], "zip")
	assert.Equal(6, len(tokens))

	// testTokenizerStrasse
	tokens = ttokenize(dat, w, "Ich wohne in der Weststr. und Du?")
	assert.Equal(tokens[4], "Weststr.")
	assert.Equal(8, len(tokens))

	// germanTokenizerKnowsGermanOmissionWords
	tokens = ttokenize(dat, w, "D'dorf Ku'damm Lu'hafen M'gladbach W'schaft")
	assert.Equal("D'dorf", tokens[0])
	assert.Equal("Ku'damm", tokens[1])
	assert.Equal("Lu'hafen", tokens[2])
	assert.Equal("M'gladbach", tokens[3])
	assert.Equal("W'schaft", tokens[4])
	assert.Equal(5, len(tokens))

	// germanTokenizerDoesNOTSeparateGermanContractions
	tokens = ttokenize(dat, w, "mach's macht's was'n ist's haste willste kannste biste kriegste")
	assert.Equal("mach's", tokens[0])
	assert.Equal("macht's", tokens[1])
	assert.Equal("was'n", tokens[2])
	assert.Equal("ist's", tokens[3])
	assert.Equal("haste", tokens[4])
	assert.Equal("willste", tokens[5])
	assert.Equal("kannste", tokens[6])
	assert.Equal("biste", tokens[7])
	assert.Equal("kriegste", tokens[8])
	assert.Equal(9, len(tokens))

	/*
		@Test
		public void englishTokenizerSeparatesEnglishContractionsAndClitics () {
				DerekoDfaTokenizer_en tok = new DerekoDfaTokenizer_en();
				tokens = tokenize(dat, w, "I've we'll you'd I'm we're Peter's isn't")
				assert.Equal("'ve", tokens[1]);
				assert.Equal("'ll", tokens[3]);
				assert.Equal("'d", tokens[5]);
				assert.Equal("'m", tokens[7]);
				assert.Equal("'re", tokens[9]);
				assert.Equal("'s", tokens[11]);
				assert.Equal("is", tokens[12]);
				assert.Equal("n't", tokens[13]);
				assert.Equal(14, len(tokens));
		}

		@Test
		public void frenchTokenizerKnowsFrenchAbbreviations () {
				DerekoDfaTokenizer_fr tok = new DerekoDfaTokenizer_fr();
				tokens = tokenize(dat, w, "Approx. en juill. 2004 mon prof. M. Foux m'a dit qu'il faut faire exerc. no. 4, et lire pp. 27-30.")
				assert.Equal("Approx.", tokens[0]);
				assert.Equal("juill.", tokens[2]);
				assert.Equal("prof.", tokens[5]);
				assert.Equal("exerc.", tokens[15]);
				assert.Equal("no.", tokens[16]);
				assert.Equal("pp.", tokens[21]);
		}

		@Test
		public void frenchTokenizerKnowsFrenchContractions () {
				DerekoDfaTokenizer_fr tok = new DerekoDfaTokenizer_fr();
				tokens = tokenize(dat, w, "J'ai j'habite qu'il d'un jusqu'à Aujourd'hui D'accord Quelqu'un Presqu'île")
				assert.Equal("J'", tokens[0]);
				assert.Equal("j'", tokens[2]);
				assert.Equal("qu'", tokens[4]);
				assert.Equal("d'", tokens[6]);
				assert.Equal("jusqu'", tokens[8]);
				assert.Equal("Aujourd'hui", tokens[10]);
				assert.Equal("D'", tokens[11]); // ’
				assert.Equal("Quelqu'un", tokens[13]); // ’
				assert.Equal("Presqu'île", tokens[14]); // ’
		}

		@Test
		public void frenchTokenizerKnowsFrenchClitics () {
				DerekoDfaTokenizer_fr tok = new DerekoDfaTokenizer_fr();
				tokens = tokenize(dat, w, "suis-je sont-elles ")
				assert.Equal("suis", tokens[0]);
				assert.Equal("-je", tokens[1]);
				assert.Equal("sont", tokens[2]);
				assert.Equal("-elles", tokens[3]);
		}

		@Test
		public void testEnglishTokenizerScienceAbbreviations () {
				DerekoDfaTokenizer_en tok = new DerekoDfaTokenizer_en();
				tokens = tokenize(dat, w, "Approx. in Sept. 1954, Assoc. Prof. Dr. R. J. Ewing reviewed articles on Enzymol. Bacteriol. effects later published in Nutr. Rheumatol. No. 12 and Nº. 13., pp. 17-18.")
				assert.Equal("Approx.", tokens[0]);
				assert.Equal("in", tokens[1]);
				assert.Equal("Sept.", tokens[2]);
				assert.Equal("1954", tokens[3]);
				assert.Equal(",", tokens[4]);
				assert.Equal("Assoc.", tokens[5]);
				assert.Equal("Prof.", tokens[6]);
				assert.Equal("Dr.", tokens[7]);
				assert.Equal("R.", tokens[8]);
				assert.Equal("J.", tokens[9]);
				assert.Equal("Ewing", tokens[10]);
				assert.Equal("reviewed", tokens[11]);
				assert.Equal("articles", tokens[12]);
				assert.Equal("on", tokens[13]);
				assert.Equal("Enzymol.", tokens[14]);
				assert.Equal("Bacteriol.", tokens[15]);
				assert.Equal("effects", tokens[16]);
				assert.Equal("later", tokens[17]);
				assert.Equal("published", tokens[18]);
				assert.Equal("in", tokens[19]);
				assert.Equal("Nutr.", tokens[20]);
				assert.Equal("Rheumatol.", tokens[21]);
				assert.Equal("No.", tokens[22]);
				assert.Equal("12", tokens[23]);
				assert.Equal("and", tokens[24]);
				assert.Equal("Nº.", tokens[25]);
				assert.Equal("13.", tokens[26]);
				assert.Equal(",", tokens[27]);
				assert.Equal("pp.", tokens[28]);
				assert.Equal("17-18", tokens[29]);
				assert.Equal(".", tokens[30]);
		}

		@Test
		public void englishTokenizerCanGuessWhetherIIsAbbrev () {
				DerekoDfaTokenizer_en tok = new DerekoDfaTokenizer_en();
				tokens = tokenize(dat, w, "M. I. Baxter was born during World War I. So was I. He went to the Peter I. Hardy school. So did I.")
				assert.Equal("I.", tokens[1]);
				assert.Equal("I", tokens[8]);
				assert.Equal(".", tokens[9]);
				assert.Equal("I", tokens[12]);
				assert.Equal(".", tokens[13]);
		}

		@Test
		public void testZipOuputArchive () {

				final ByteArrayOutputStream clearOut = new ByteArrayOutputStream();
				System.setOut(new PrintStream(clearOut));
				tokens = tokenize(dat, w, "Archive:  ich/bin/ein.zip\n")
				assert.Equal(0, len(tokens));
		}
	*/
	/*

		@Test
		public void testTextBreakOutputArchive () throws InstantiationException, IllegalAccessException, ClassNotFoundException {
				DerekoDfaTokenizer_de tok = (DerekoDfaTokenizer_de) new KorapTokenizer.Builder()
								.tokenizerClassName(DerekoDfaTokenizer_de.class.getName())
								.printOffsets(true)
								.build();
				Span[] tokens = tok.tokenizePos("Text1\004\nText2 Hallo\004Rumsdibums\004Das freut mich sehr.\n");
				assert.Equal("Text1", tokens[0].getType());
				assert.Equal(len(tokens), 9 );
		}
	*/
}

func TestDoubleArrayLoadFactor1(t *testing.T) {
	assert := assert.New(t)
	tok := LoadFomaFile("testdata/abbr_bench.fst")
	dat := tok.ToDoubleArray()
	assert.True(dat.LoadFactor() > 88)
}

func TestDoubleArrayFullTokenizerXML(t *testing.T) {
	assert := assert.New(t)

	if dat == nil {
		dat = LoadDatokFile("testdata/tokenizer.datok")
	}

	assert.NotNil(dat)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	tokens = ttokenize(dat, w, "Das <b>beste</b> Fußballspiel")
	assert.Equal("Das", tokens[0])
	assert.Equal("<b>", tokens[1])
	assert.Equal("beste", tokens[2])
	assert.Equal("</b>", tokens[3])
	assert.Equal("Fußballspiel", tokens[4])
	assert.Equal(5, len(tokens))

	tokens = ttokenize(dat, w, "Das <b class=\"c\">beste</b> Fußballspiel")
	assert.Equal("Das", tokens[0])
	assert.Equal("<b class=\"c\">", tokens[1])
	assert.Equal("beste", tokens[2])
	assert.Equal("</b>", tokens[3])
	assert.Equal("Fußballspiel", tokens[4])
	assert.Equal(5, len(tokens))

	tokens = ttokenize(dat, w, "der<x  y=\"alte \"> <x x> alte</x> etc. et. Mann.")
	assert.Equal("der", tokens[0])
	assert.Equal("<x  y=\"alte \">", tokens[1])
	assert.Equal("<x x>", tokens[2])
	assert.Equal("alte", tokens[3])
	assert.Equal("</x>", tokens[4])
	assert.Equal("etc.", tokens[5])
	assert.Equal("et", tokens[6])
	assert.Equal(".", tokens[7])
	assert.Equal("Mann", tokens[8])
	assert.Equal(".", tokens[9])
	assert.Equal(10, len(tokens))
}

func BenchmarkDoubleArrayTransduce(b *testing.B) {
	bu := make([]byte, 0, 2048)
	w := bytes.NewBuffer(bu)

	s := `Der Vorsitzende der Abk. hat gewählt. Gefunden auf wikipedia.org. Ich bin unter korap@ids-mannheim.de erreichbar.
	Unsere Website ist https://korap.ids-mannheim.de/?q=Baum. Unser Server ist 10.0.10.51. Zu 50.4% ist es sicher.
	Der Termin ist am 5.9.2018.
	Ich habe die readme.txt heruntergeladen.
	Ausschalten!!! Hast Du nicht gehört???
	Ich wohne in der Weststr. und Du? Kupietz und Schmidt [2018]: Korpuslinguistik. Dieses verf***** Kleid! Ich habe die readme.txt heruntergeladen.
	Er sagte: \"Es geht mir gut!\", daraufhin ging er. &quot;Das ist von C&A!&quot; Früher bzw. später ... Sie erreichte den 1. Platz!
	Archive:  Ich bin kein zip. D'dorf Ku'damm Lu'hafen M'gladbach W'schaft.
	Mach's macht's was'n ist's haste willste kannste biste kriegste.`
	r := strings.NewReader(s)

	dat := LoadDatokFile("testdata/tokenizer.datok")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Reset()
		r.Reset(s)
		ok := dat.Transduce(r, w)
		if !ok {
			fmt.Println("Fail!")
			fmt.Println(w.String())
			os.Exit(1)
		}
	}
}

// This test is deprecated as the datok file changes over time
func XBenchmarkLoadDatokFile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dat := LoadDatokFile("testdata/tokenizer.datok")
		if dat == nil {
			fmt.Println("Fail!")
			os.Exit(1)
		}
	}
}

func BenchmarkDoubleArrayConstruction(b *testing.B) {
	tok := LoadFomaFile("testdata/simple_bench.fst")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dat := tok.ToDoubleArray()
		if dat == nil {
			fmt.Println("Fail!")
			os.Exit(1)
		}
	}
}

func BenchmarkDoubleArrayLarger(b *testing.B) {
	tok := LoadFomaFile("testdata/abbr_bench.fst")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dat := tok.ToDoubleArray()
		if dat == nil {
			fmt.Println("Fail!")
			os.Exit(1)
		}
	}
}

// 2021-08-11 (go 1.16)
// go test -bench=. -test.benchmem
//   BenchmarkTransduce-4         19069             60609 ns/op           11048 B/op        137 allocs/op
// 2021-08-12 (go 1.16)
//   BenchmarkTransduce-4         20833             55241 ns/op            9676 B/op          3 allocs/op
//   BenchmarkLoadDatokFile-4         4         258418169 ns/op        29916470 B/op       5697 allocs/op
//   BenchmarkTransduce-4         19430             58133 ns/op           18696 B/op          3 allocs/op
//   BenchmarkLoadDatokFile-4         8         139071939 ns/op       203158377 B/op       5742 allocs/op
// 2021-08-16
//   BenchmarkTransduce-4               22251             49989 ns/op           17370 B/op          3 allocs/op
//   BenchmarkLoadDatokFile-4               8         138937532 ns/op        203158327 B/op      5742 allocs/op
//   BenchmarkTransduce-4               22005             48665 ns/op           17472 B/op          3 allocs/op
//   BenchmarkLoadDatokFile-4               7         143143934 ns/op        203158450 B/op      5743 allocs/op
//   BenchmarkTransduce-4               34939             34363 ns/op           14056 B/op          3 allocs/op
//   BenchmarkLoadDatokFile-4               7         149511609 ns/op        203217193 B/op      5915 allocs/op
// 2021-08-17
//   BenchmarkTransduce-4               31204             32678 ns/op           14752 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4           44138             26850 ns/op           10704 B/op         29 allocs/op
//   BenchmarkTransduce-4               29376             34562 ns/op           15157 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4           54441             21355 ns/op           10704 B/op         29 allocs/op
// 2021-09-02 - New tokenizer - fixed loading
//   BenchmarkTransduce-4                       40149             31515 ns/op            8240 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4                   51043             22586 ns/op           10702 B/op         29 allocs/op
//   BenchmarkToDoubleArrayLarger-4                 3         396009639 ns/op         6352293 B/op       2575 allocs/op
//   BenchmarkTransduce-4                       38698             31900 ns/op            8240 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4                   50644             21569 ns/op           11151 B/op         14 allocs/op
//   BenchmarkToDoubleArrayLarger-4                 3         441260766 ns/op         6942336 B/op         30 allocs/op
//   BenchmarkTransduce-4                       39966             30835 ns/op            8240 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4                   50720             24863 ns/op           11091 B/op         46 allocs/op
//   BenchmarkToDoubleArrayLarger-4                 3         432523828 ns/op         6413381 B/op       5122 allocs/op
// 2021-09-02 - xCheckSkip() with .9
//   BenchmarkTransduce-4                       36325             38501 ns/op            8240 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4                   66858             19286 ns/op           10607 B/op         29 allocs/op
//   BenchmarkToDoubleArrayLarger-4                18          67428011 ns/op         6360604 B/op       2578 allocs/op
// 2021-09-02 - xCheckSkipNiu() with .9 and >= 3
//   BenchmarkTransduce-4                       37105             27714 ns/op            8240 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4                   76600             15973 ns/op           10703 B/op         29 allocs/op
//   BenchmarkToDoubleArrayLarger-4                21          55161934 ns/op         6357889 B/op       2578 allocs/op
// 2021-09-30 - Go 1.17.1
//   BenchmarkTransduce-4                       47222             25962 ns/op            8240 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4                   69192             17355 ns/op           10704 B/op         29 allocs/op
//   BenchmarkToDoubleArrayLarger-4                16          65042885 ns/op         6357794 B/op       2576 allocs/op
//   BenchmarkTransduceMatrix-4                 45404             25156 ns/op            8240 B/op          3 allocs/op
// 2021-10-02
//   BenchmarkTransduce-4                       47676             25398 ns/op            8240 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4                   71919             16083 ns/op           10702 B/op         29 allocs/op
//   BenchmarkToDoubleArrayLarger-4                16          68012819 ns/op         6357920 B/op       2578 allocs/op
//   BenchmarkTransduceMatrix-4                 51529             23678 ns/op            8240 B/op          3 allocs/op
// 2021-10-12 - Introduction of Callbacks in Matrix
//   BenchmarkTransduce-4                       46947             26043 ns/op            8240 B/op          3 allocs/op
//   BenchmarkToDoubleArray-4                   65192             16501 ns/op           10703 B/op         29 allocs/op
//   BenchmarkToDoubleArrayLarger-4                15          69263576 ns/op         6357859 B/op       2577 allocs/op
//   BenchmarkTransduceMatrix-4                 49928             26313 ns/op           12408 B/op          6 allocs/op
// 2021-10-18 - Introduction of Callbacks in DA
//   BenchmarkTransduce-4                       41055             30058 ns/op           12408 B/op          6 allocs/op
//   BenchmarkToDoubleArray-4                   64672             17659 ns/op           10703 B/op         29 allocs/op
//   BenchmarkToDoubleArrayLarger-4                15          71640553 ns/op         6357865 B/op       2577 allocs/op
//   BenchmarkTransduceMatrix-4                 47036             26009 ns/op           12408 B/op          6 allocs/op
// 2021-10-21 - Simplify DA code to ignore final states
//   BenchmarkTransduce-4                       41365             33766 ns/op           12408 B/op          6 allocs/op
//   BenchmarkToDoubleArray-4                   63663             17675 ns/op           10703 B/op         29 allocs/op
//   BenchmarkToDoubleArrayLarger-4                16          83535733 ns/op         6357874 B/op       2577 allocs/op
//   BenchmarkTransduceMatrix-4                 45362             25258 ns/op           12408 B/op          6 allocs/op
// 2021-10-22 - Introduxe EOT
//   BenchmarkDoubleArrayTransduce-4            43820             27661 ns/op           12408 B/op          6 allocs/op
//   BenchmarkDoubleArrayConstruction-4         68259             16608 ns/op           10703 B/op         29 allocs/op
//   BenchmarkDoubleArrayLarger-4                  16          69889532 ns/op         6357901 B/op       2578 allocs/op
//   BenchmarkMatrixTransduce-4                 49426             25105 ns/op           12408 B/op          6 allocs/op
// 2021-10-23 - Improve offset handling
//   BenchmarkDoubleArrayTransduce-4            41890             29729 ns/op           12408 B/op          6 allocs/op
//   BenchmarkDoubleArrayConstruction-4         74510             15879 ns/op           10703 B/op         29 allocs/op
//   BenchmarkDoubleArrayLarger-4                  18          73752383 ns/op         6357956 B/op       2579 allocs/op
//   BenchmarkMatrixTransduce-4                 46870             27140 ns/op           12408 B/op          6 allocs/op
// 2021-10-28 - Finalize feature compatibility with KorAP-Tokenizer
//   BenchmarkDoubleArrayTransduce-4            39130             31612 ns/op           28944 B/op         16 allocs/op
//   BenchmarkDoubleArrayConstruction-4         79302             14994 ns/op           10703 B/op         29 allocs/op
//   BenchmarkDoubleArrayLarger-4                  18          67942077 ns/op         6357870 B/op       2577 allocs/op
//   BenchmarkMatrixTransduce-4                 39536             30510 ns/op           28944 B/op         16 allocs/op
// 2021-11-09 - go 1.17.3
//   BenchmarkDoubleArrayTransduce-4            35067             34192 ns/op           28944 B/op         17 allocs/op
//   BenchmarkDoubleArrayConstruction-4         72446             15614 ns/op           10703 B/op         29 allocs/op
//   BenchmarkDoubleArrayLarger-4                  16          71058822 ns/op         6357860 B/op       2577 allocs/op
//   BenchmarkMatrixTransduce-4                 36703             31891 ns/op           28944 B/op         17 allocs/op
// 2021-11-10 - rearranged longest match operator
//   BenchmarkDoubleArrayTransduce-4    	   34522	     33210 ns/op	   28944 B/op	      17 allocs/op
//   BenchmarkDoubleArrayConstruction-4   	   66990	     16012 ns/op	   10703 B/op	      29 allocs/op
//   BenchmarkDoubleArrayLarger-4         	      16	  62829878 ns/op	 6357823 B/op	    2576 allocs/op
//   BenchmarkMatrixTransduce-4           	   36154	     32702 ns/op	   28944 B/op	      17 allocs/op

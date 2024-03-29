package datok

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var s string = `Der Vorsitzende der Abk. hat gewählt. Gefunden auf wikipedia.org. Ich bin unter korap@ids-mannheim.de erreichbar.
Unsere Website ist https://korap.ids-mannheim.de/?q=Baum. Unser Server ist 10.0.10.51. Zu 50.4% ist es sicher.
Der Termin ist am 5.9.2018.
Ich habe die readme.txt heruntergeladen.
Ausschalten!!! Hast Du nicht gehört???
Ich wohne in der Weststr. und Du? Kupietz und Schmidt [2018]: Korpuslinguistik. Dieses verf***** Kleid! Ich habe die readme.txt heruntergeladen.
Er sagte: \"Es geht mir gut!\", daraufhin ging er. &quot;Das ist von C&A!&quot; Früher bzw. später ... Sie erreichte den 1. Platz!
Archive:  Ich bin kein zip. D'dorf Ku'damm Lu'hafen M'gladbach W'schaft.
Mach's macht's was'n ist's haste willste kannste biste kriegste.`

var mat_de, mat_en *MatrixTokenizer

func TestMatrixFullTokenizer(t *testing.T) {
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
	assert.Equal(len(tokens), 11)
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
	assert.Equal("", tokens[6])
	assert.Equal(7, len(tokens))
}

func TestMatrixSimpleString(t *testing.T) {
	assert := assert.New(t)
	// bau | bauamt
	tok := LoadFomaFile("testdata/bauamt.fst")
	mat := tok.ToMatrix()

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	tokens = ttokenize(mat, w, "ibauamt")
	assert.Equal("i", tokens[0])
	assert.Equal("bauamt", tokens[1])

	tokens = ttokenize(mat, w, "ibbauamt")
	assert.Equal("i", tokens[0])

	assert.Equal("b", tokens[1])
	assert.Equal("bauamt", tokens[2])

	tokens = ttokenize(mat, w, "bau")
	assert.Equal("bau", tokens[0])

	tokens = ttokenize(mat, w, "baum")
	assert.Equal("bau", tokens[0])
	assert.Equal("m", tokens[1])

	tokens = ttokenize(mat, w, "baudibauamt")
	assert.Equal("bau", tokens[0])
	assert.Equal("d", tokens[1])
	assert.Equal("i", tokens[2])
	assert.Equal("bauamt", tokens[3])
}

func TestMatrixCliticRule(t *testing.T) {
	assert := assert.New(t)
	mat := LoadMatrixFile("testdata/clitic_test.matok")

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	tokens = ttokenize(mat, w, "ibauamt")
	assert.Equal("ibauamt", tokens[0])

	exstring := "dead. "

	tokens = ttokenize(mat, w, exstring)
	assert.Equal("dead", tokens[0])
	assert.Equal(".", tokens[1])

	w.Reset()
	tws := NewTokenWriter(w, TOKENS|SENTENCES)

	assert.True(mat.TransduceTokenWriter(

		strings.NewReader(exstring), tws),
	)
	tws.Flush()

	matStr := w.String()
	assert.Equal("dead\n.\n\n\n", matStr)

	tokens = ttokenize(mat, w, "they're")
	assert.Equal("they", tokens[0])
	assert.Equal("'re", tokens[1])

	tokens = ttokenize(mat, w, "they're They're their don't wouldn't")
	assert.Equal("they", tokens[0])
	assert.Equal("'re", tokens[1])
	assert.Equal("They", tokens[2])
	assert.Equal("'re", tokens[3])
	assert.Equal("their", tokens[4])
	assert.Equal("do", tokens[5])
	assert.Equal("n't", tokens[6])
	assert.Equal("would", tokens[7])
	assert.Equal("n't", tokens[8])
}

func TestMatrixReadWriteTokenizer(t *testing.T) {
	assert := assert.New(t)
	foma := LoadFomaFile("testdata/simpletok.fst")
	assert.NotNil(foma)

	mat := foma.ToMatrix()
	assert.NotNil(mat)

	assert.Equal(ttokenizeStr(mat, "bau"), "bau")
	assert.Equal(ttokenizeStr(mat, "bad"), "bad")
	assert.Equal(ttokenizeStr(mat, "wald gehen"), "wald\ngehen")
	b := make([]byte, 0, 1024)
	buf := bytes.NewBuffer(b)
	n, err := mat.WriteTo(buf)
	assert.Nil(err)
	assert.Equal(int64(230), n)
	mat2 := ParseMatrix(buf)
	assert.NotNil(mat2)
	assert.Equal(mat.sigma, mat2.sigma)
	assert.Equal(mat.epsilon, mat2.epsilon)
	assert.Equal(mat.unknown, mat2.unknown)
	assert.Equal(mat.identity, mat2.identity)
	assert.Equal(mat.stateCount, mat2.stateCount)
	assert.Equal(len(mat.array), len(mat2.array))
	assert.Equal(mat.array, mat2.array)
	assert.Equal(ttokenizeStr(mat2, "bau"), "bau")
	assert.Equal(ttokenizeStr(mat2, "bad"), "bad")
	assert.Equal(ttokenizeStr(mat2, "wald gehen"), "wald\ngehen")
}

func TestMatrixIgnorableMCS(t *testing.T) {
	assert := assert.New(t)

	// This test relies on final states. That's why it is
	// not working correctly anymore.

	// File has MCS in sigma but not in net
	tok := LoadFomaFile("testdata/ignorable_mcs.fst")
	assert.NotNil(tok)
	mat := tok.ToMatrix()
	assert.NotNil(mat)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	// Is only unambigous when transducing strictly greedy!
	assert.True(mat.Transduce(strings.NewReader("ab<ab>a"), w))
	tokens = strings.Split(w.String(), "\n")
	assert.Equal("a\nb\n<ab>a\n\n\n", w.String())
	assert.Equal("a", tokens[0])
	assert.Equal("b", tokens[1])
	assert.Equal("<ab>a", tokens[2])
	assert.Equal(6, len(tokens))
}

func xTestMatrixReadWriteFullTokenizer(t *testing.T) {
	assert := assert.New(t)
	foma := LoadFomaFile("testdata/tokenizer_de.fst")
	assert.NotNil(foma)

	mat := foma.ToMatrix()
	assert.NotNil(foma)

	tb := make([]byte, 0, 2048)
	w := bytes.NewBuffer(tb)

	assert.True(mat.Transduce(strings.NewReader("der alte baum"), w))
	assert.Equal("der\nalte\nbaum\n\n\n", w.String())

	b := make([]byte, 0, 1024)
	buf := bytes.NewBuffer(b)
	_, err := mat.WriteTo(buf)
	assert.Nil(err)
	w.Reset()
	// assert.Equal(int64(248), n)

	mat2 := ParseMatrix(buf)
	assert.NotNil(mat2)
	assert.Equal(mat.sigma, mat2.sigma)
	assert.Equal(mat.epsilon, mat2.epsilon)
	assert.Equal(mat.unknown, mat2.unknown)
	assert.Equal(mat.identity, mat2.identity)
	assert.Equal(mat.stateCount, mat2.stateCount)
	assert.Equal(len(mat.array), len(mat2.array))
	// assert.Equal(mat.array, mat2.array)

	assert.True(mat2.Transduce(strings.NewReader("der alte baum"), w))
	assert.Equal("der\nalte\nbaum\n\n\n", w.String())
}

func TestMatrixFullTokenizerTransduce(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	assert.NotNil(mat_de)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	assert.True(mat_de.Transduce(strings.NewReader("tra. u Du?"), w))

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
	assert.Equal(9, len(tokens))

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("\"John Doe\"@xx.com"), w))
	assert.Equal("\"\nJohn\nDoe\n\"\n@xx\n.\n\ncom\n\n\n", w.String())
}

func TestMatrixFullTokenizerMatrixSentenceSplitter(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var sentences []string

	// testSentSplitterSimple
	assert.True(mat_de.Transduce(strings.NewReader("Der alte Mann."), w))
	sentences = strings.Split(w.String(), "\n\n")

	assert.Equal("Der\nalte\nMann\n.\n\n\n", w.String())
	assert.Equal("Der\nalte\nMann\n.", sentences[0])
	assert.Equal("\n", sentences[1])
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Der Vorsitzende der F.D.P. hat gewählt."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)
	assert.Equal("Der\nVorsitzende\nder\nF.D.P.\nhat\ngewählt\n.", sentences[0])
	assert.Equal("\n", sentences[1])

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Der Vorsitzende der Abk. hat gewählt."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)
	assert.Equal("Der\nVorsitzende\nder\nAbk.\nhat\ngewählt\n.", sentences[0])
	assert.Equal("\n", sentences[1])

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader(""), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)
	assert.Equal("", sentences[0])
	assert.Equal("", sentences[1])

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Gefunden auf wikipedia.org."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Ich bin unter korap@ids-mannheim.de erreichbar."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Unsere Website ist https://korap.ids-mannheim.de/?q=Baum"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal("Unsere\nWebsite\nist\nhttps://korap.ids-mannheim.de/?q=Baum", sentences[0])
	assert.Equal("\n", sentences[1])
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Unser Server ist 10.0.10.51."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal("\n", sentences[1])
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Zu 50.4% ist es sicher"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Der Termin ist am 5.9.2018"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Ich habe die readme.txt heruntergeladen"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)
	assert.Equal("Ich\nhabe\ndie\nreadme.txt\nheruntergeladen", sentences[0])
	assert.Equal("\n", sentences[1])

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Ausschalten!!! Hast Du nicht gehört???"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 3)
	assert.Equal("Ausschalten\n!!!", sentences[0])
	assert.Equal("Hast\nDu\nnicht\ngehört\n???", sentences[1])
	assert.Equal("\n", sentences[2])

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("Ich wohne in der Weststr. und Du?"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 2)

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("\"Alter!\", sagte er: \"Komm nicht wieder!\" Geh!!! \"Lass!\" Dann ging er."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 5)
	assert.Equal("\"\nAlter\n!\n\"\n,\nsagte\ner\n:\n\"\nKomm\nnicht\nwieder\n!\n\"", sentences[0])
	assert.Equal("Geh\n!!!", sentences[1])
	assert.Equal("\"\nLass\n!\n\"", sentences[2])
	assert.Equal("Dann\nging\ner\n.", sentences[3])

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("\"Ausschalten!!!\", sagte er. \"Hast Du nicht gehört???\""), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 3)
	assert.Equal("\"\nAusschalten\n!!!\n\"\n,\nsagte\ner\n.", sentences[0])
	assert.Equal("\"\nHast\nDu\nnicht\ngehört\n???\n\"", sentences[1])

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("“Ausschalten!!!”, sagte er. «Hast Du nicht gehört???»"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 3)
	assert.Equal("“\nAusschalten\n!!!\n”\n,\nsagte\ner\n.", sentences[0])
	assert.Equal("«\nHast\nDu\nnicht\ngehört\n???\n»", sentences[1])

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("“Ausschalten!!!”, sagte er. «Hast Du nicht gehört???»"), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 3)
	assert.Equal("“\nAusschalten\n!!!\n”\n,\nsagte\ner\n.", sentences[0])
	assert.Equal("«\nHast\nDu\nnicht\ngehört\n???\n»", sentences[1])

	text := `»Meinetwegen. Denkst du, daß ich darauf warte? Das fehlte noch.
Übrigens, ich kriege schon einen und vielleicht bald. Da ist mir nicht
bange. Neulich erst hat mir der kleine Ventivegni von drüben gesagt:
'Fräulein Effi, was gilt die Wette, wir sind hier noch in diesem Jahre
zu Polterabend und Hochzeit.'«

»Und was sagtest du da?«`

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader(text), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 8)
	assert.Equal("Neulich\nerst\nhat\nmir\nder\nkleine\nVentivegni\nvon\ndrüben\ngesagt\n:\n'\nFräulein\nEffi\n,\nwas\ngilt\ndie\nWette\n,\nwir\nsind\nhier\nnoch\nin\ndiesem\nJahre\nzu\nPolterabend\nund\nHochzeit\n.\n'\n«", sentences[5])
	assert.Equal("»\nUnd\nwas\nsagtest\ndu\nda\n?\n«", sentences[6])

	text = `»Nun, gib dich zufrieden, ich fange schon an ... Also Baron
Innstetten!`

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader(text), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 3)
	assert.Equal("»\nNun\n,\ngib\ndich\nzufrieden\n,\nich\nfange\nschon\nan\n...", sentences[0])
	assert.Equal("Also\nBaron\nInnstetten\n!", sentences[1])

	// Check parantheses at the end of the sentence
	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("(Er ging.) Und kam (später)."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 3)
	assert.Equal("(\nEr\nging\n.\n)", sentences[0])
	assert.Equal("Und\nkam\n(\nspäter\n)\n.", sentences[1])

	// Check parantheses and quotes at the end of the sentence
	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader("(Er sagte: \"Hallo!\") Dann ging er."), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 3)
	assert.Equal("(\nEr\nsagte\n:\n\"\nHallo\n!\n\"\n)", sentences[0])
	assert.Equal("Dann\nging\ner\n.", sentences[1])

}

func TestMatrixFullTokenizerMatrixSentenceSplitterBug1(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var sentences []string

	text := `Wüllersdorf war aufgestanden. »Ich finde es furchtbar, daß Sie recht haben, aber Sie haben recht. Ich quäle Sie nicht länger mit meinem 'Muß es sein?'. Die Welt ist einmal, wie sie ist, und die Dinge verlaufen nicht, wie wir wollen, sondern wie die andern wollen. Das mit dem 'Gottesgericht', wie manche hochtrabend versichern, ist freilich ein Unsinn, nichts davon, umgekehrt, unser Ehrenkultus ist ein Götzendienst, aber wir müssen uns ihm unterwerfen, solange der Götze gilt.«`

	w.Reset()
	assert.True(mat_de.Transduce(strings.NewReader(text), w))
	sentences = strings.Split(w.String(), "\n\n")
	assert.Equal(len(sentences), 6)
	assert.Equal("Wüllersdorf\nwar\naufgestanden\n.", sentences[0])
	assert.Equal("»\nIch\nfinde\nes\nfurchtbar\n,\ndaß\nSie\nrecht\nhaben\n,\naber\nSie\nhaben\nrecht\n.", sentences[1])
	assert.Equal("Ich\nquäle\nSie\nnicht\nlänger\nmit\nmeinem\n'\nMuß\nes\nsein\n?\n'\n.", sentences[2])
	assert.Equal("Die\nWelt\nist\neinmal\n,\nwie\nsie\nist\n,\nund\ndie\nDinge\nverlaufen\nnicht\n,\nwie\nwir\nwollen\n,\nsondern\nwie\ndie\nandern\nwollen\n.", sentences[3])
	assert.Equal("Das\nmit\ndem\n'\nGottesgericht\n'\n,\nwie\nmanche\nhochtrabend\nversichern\n,\nist\nfreilich\nein\nUnsinn\n,\nnichts\ndavon\n,\numgekehrt\n,\nunser\nEhrenkultus\nist\nein\nGötzendienst\n,\naber\nwir\nmüssen\nuns\nihm\nunterwerfen\n,\nsolange\nder\nGötze\ngilt\n.\n«", sentences[4])
}

func TestMatrixFullTokenizerTokenSplitter(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	// testTokenizerSimple
	tokens = ttokenize(mat_de, w, "Der alte Mann")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "alte")
	assert.Equal(tokens[2], "Mann")
	assert.Equal(len(tokens), 3)

	tokens = ttokenize(mat_de, w, "Der alte Mann.")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "alte")
	assert.Equal(tokens[2], "Mann")
	assert.Equal(tokens[3], ".")
	assert.Equal(len(tokens), 4)

	// testTokenizerAbbr
	tokens = ttokenize(mat_de, w, "Der Vorsitzende der F.D.P. hat gewählt")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "Vorsitzende")
	assert.Equal(tokens[2], "der")
	assert.Equal(tokens[3], "F.D.P.")
	assert.Equal(tokens[4], "hat")
	assert.Equal(tokens[5], "gewählt")
	assert.Equal(len(tokens), 6)
	// Ignored in KorAP-Tokenizer

	// testTokenizerHost1
	tokens = ttokenize(mat_de, w, "Gefunden auf wikipedia.org")
	assert.Equal(tokens[0], "Gefunden")
	assert.Equal(tokens[1], "auf")
	assert.Equal(tokens[2], "wikipedia.org")
	assert.Equal(len(tokens), 3)

	// testTokenizerWwwHost
	tokens = ttokenize(mat_de, w, "Gefunden auf www.wikipedia.org")
	assert.Equal("Gefunden", tokens[0])
	assert.Equal("auf", tokens[1])
	assert.Equal("www.wikipedia.org", tokens[2])
	assert.Equal(3, len(tokens))

	// testTokenizerWwwUrl
	tokens = ttokenize(mat_de, w, "Weitere Infos unter www.info.biz/info")
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
	tokens = ttokenize(mat_de, w, "Das war -- spitze")
	assert.Equal(tokens[0], "Das")
	assert.Equal(tokens[1], "war")
	assert.Equal(tokens[2], "--")
	assert.Equal(tokens[3], "spitze")
	assert.Equal(len(tokens), 4)

	// testTokenizerEmail1
	tokens = ttokenize(mat_de, w, "Ich bin unter korap@ids-mannheim.de erreichbar.")
	assert.Equal(tokens[0], "Ich")
	assert.Equal(tokens[1], "bin")
	assert.Equal(tokens[2], "unter")
	assert.Equal(tokens[3], "korap@ids-mannheim.de")
	assert.Equal(tokens[4], "erreichbar")
	assert.Equal(tokens[5], ".")
	assert.Equal(len(tokens), 6)

	// testTokenizerEmail2
	tokens = ttokenize(mat_de, w, "Oder unter korap[at]ids-mannheim[dot]de.")
	assert.Equal(tokens[0], "Oder")
	assert.Equal(tokens[1], "unter")
	assert.Equal(tokens[2], "korap[at]ids-mannheim[dot]de")
	assert.Equal(tokens[3], ".")
	assert.Equal(len(tokens), 4)

	// testTokenizerEmail3
	tokens = ttokenize(mat_de, w, "Oder unter korap(at)ids-mannheim(dot)de.")
	assert.Equal(tokens[0], "Oder")
	assert.Equal(tokens[1], "unter")
	assert.Equal(tokens[2], "korap(at)ids-mannheim(dot)de")
	assert.Equal(tokens[3], ".")
	assert.Equal(len(tokens), 4)
	// Ignored in KorAP-Tokenizer

	// testTokenizerDoNotAcceptQuotedEmailNames
	tokens = ttokenize(mat_de, w, "\"John Doe\"@xx.com")
	assert.Equal("\"", tokens[0])
	assert.Equal("John", tokens[1])
	assert.Equal("Doe", tokens[2])
	assert.Equal("\"", tokens[3])
	assert.Equal("@xx", tokens[4])
	assert.Equal(".", tokens[5]) // Differs - as the sentence splitter splits here!
	assert.Equal("com", tokens[6])
	assert.Equal(7, len(tokens))

	// testTokenizerTwitter
	tokens = ttokenize(mat_de, w, "Folgt @korap und #korap")
	assert.Equal(tokens[0], "Folgt")
	assert.Equal(tokens[1], "@korap")
	assert.Equal(tokens[2], "und")
	assert.Equal(tokens[3], "#korap")
	assert.Equal(len(tokens), 4)

	// testTokenizerWeb1
	tokens = ttokenize(mat_de, w, "Unsere Website ist https://korap.ids-mannheim.de/?q=Baum")
	assert.Equal(tokens[0], "Unsere")
	assert.Equal(tokens[1], "Website")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "https://korap.ids-mannheim.de/?q=Baum")
	assert.Equal(len(tokens), 4)

	// testTokenizerWeb2
	tokens = ttokenize(mat_de, w, "Wir sind auch im Internet (https://korap.ids-mannheim.de/?q=Baum)")
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
	tokens = ttokenize(mat_de, w, "Die Adresse ist https://korap.ids-mannheim.de/?q=Baum.")
	assert.Equal(tokens[0], "Die")
	assert.Equal(tokens[1], "Adresse")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "https://korap.ids-mannheim.de/?q=Baum")
	assert.Equal(tokens[4], ".")
	assert.Equal(len(tokens), 5)
	// Ignored in KorAP-Tokenizer

	// testTokenizerServer
	tokens = ttokenize(mat_de, w, "Unser Server ist 10.0.10.51.")
	assert.Equal(tokens[0], "Unser")
	assert.Equal(tokens[1], "Server")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "10.0.10.51")
	assert.Equal(tokens[4], ".")
	assert.Equal(len(tokens), 5)

	// testTokenizerNum
	tokens = ttokenize(mat_de, w, "Zu 50,4% ist es sicher")
	assert.Equal(tokens[0], "Zu")
	assert.Equal(tokens[1], "50,4%")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "es")
	assert.Equal(tokens[4], "sicher")
	assert.Equal(len(tokens), 5)
	// Differs from KorAP-Tokenizer

	// testTokenizerDate
	tokens = ttokenize(mat_de, w, "Der Termin ist am 5.9.2018")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "Termin")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "am")
	assert.Equal(tokens[4], "5.9.2018")
	assert.Equal(len(tokens), 5)

	tokens = ttokenize(mat_de, w, "Der Termin ist am 5/9/2018")
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
	tokens = ttokenize(mat_de, w, "Das ist toll! ;)")
	assert.Equal(tokens[0], "Das")
	assert.Equal(tokens[1], "ist")
	assert.Equal(tokens[2], "toll")
	assert.Equal(tokens[3], "!")
	assert.Equal(tokens[4], ";)")
	assert.Equal(len(tokens), 5)

	// testTokenizerRef1
	tokens = ttokenize(mat_de, w, "Kupietz und Schmidt (2018): Korpuslinguistik")
	assert.Equal(tokens[0], "Kupietz")
	assert.Equal(tokens[1], "und")
	assert.Equal(tokens[2], "Schmidt")
	assert.Equal(tokens[3], "(2018)")
	assert.Equal(tokens[4], ":")
	assert.Equal(tokens[5], "Korpuslinguistik")
	assert.Equal(len(tokens), 6)
	// Differs from KorAP-Tokenizer!

	// testTokenizerRef2 () {
	tokens = ttokenize(mat_de, w, "Kupietz und Schmidt [2018]: Korpuslinguistik")
	assert.Equal(tokens[0], "Kupietz")
	assert.Equal(tokens[1], "und")
	assert.Equal(tokens[2], "Schmidt")
	assert.Equal(tokens[3], "[2018]")
	assert.Equal(tokens[4], ":")
	assert.Equal(tokens[5], "Korpuslinguistik")
	assert.Equal(len(tokens), 6)
	// Differs from KorAP-Tokenizer!

	// testTokenizerOmission1 () {
	tokens = ttokenize(mat_de, w, "Er ist ein A****loch!")
	assert.Equal(tokens[0], "Er")
	assert.Equal(tokens[1], "ist")
	assert.Equal(tokens[2], "ein")
	assert.Equal(tokens[3], "A****loch")
	assert.Equal(tokens[4], "!")
	assert.Equal(len(tokens), 5)

	// testTokenizerOmission2
	tokens = ttokenize(mat_de, w, "F*ck!")
	assert.Equal(tokens[0], "F*ck")
	assert.Equal(tokens[1], "!")
	assert.Equal(len(tokens), 2)

	// testTokenizerOmission3 () {
	tokens = ttokenize(mat_de, w, "Dieses verf***** Kleid!")
	assert.Equal(tokens[0], "Dieses")
	assert.Equal(tokens[1], "verf*****")
	assert.Equal(tokens[2], "Kleid")
	assert.Equal(tokens[3], "!")
	assert.Equal(len(tokens), 4)

	// Probably interpreted as HOST
	// testTokenizerFileExtension1
	tokens = ttokenize(mat_de, w, "Ich habe die readme.txt heruntergeladen")
	assert.Equal(tokens[0], "Ich")
	assert.Equal(tokens[1], "habe")
	assert.Equal(tokens[2], "die")
	assert.Equal(tokens[3], "readme.txt")
	assert.Equal(tokens[4], "heruntergeladen")
	assert.Equal(len(tokens), 5)

	// Probably interpreted as HOST
	// testTokenizerFileExtension2
	tokens = ttokenize(mat_de, w, "Nimm die README.TXT!")
	assert.Equal(tokens[0], "Nimm")
	assert.Equal(tokens[1], "die")
	assert.Equal(tokens[2], "README.TXT")
	assert.Equal(tokens[3], "!")
	assert.Equal(len(tokens), 4)

	// Probably interpreted as HOST
	// testTokenizerFileExtension3
	tokens = ttokenize(mat_de, w, "Zeig mir profile.jpeg")
	assert.Equal(tokens[0], "Zeig")
	assert.Equal(tokens[1], "mir")
	assert.Equal(tokens[2], "profile.jpeg")
	assert.Equal(len(tokens), 3)

	// testTokenizerFile1

	tokens = ttokenize(mat_de, w, "Zeig mir c:\\Dokumente\\profile.docx")
	assert.Equal(tokens[0], "Zeig")
	assert.Equal(tokens[1], "mir")
	assert.Equal(tokens[2], "c:\\Dokumente\\profile.docx")
	assert.Equal(len(tokens), 3)

	// testTokenizerFile2
	tokens = ttokenize(mat_de, w, "Gehe zu /Dokumente/profile.docx")
	assert.Equal(tokens[0], "Gehe")
	assert.Equal(tokens[1], "zu")
	assert.Equal(tokens[2], "/Dokumente/profile.docx")
	assert.Equal(len(tokens), 3)

	// testTokenizerFile3
	tokens = ttokenize(mat_de, w, "Zeig mir c:\\Dokumente\\profile.jpeg")
	assert.Equal(tokens[0], "Zeig")
	assert.Equal(tokens[1], "mir")
	assert.Equal(tokens[2], "c:\\Dokumente\\profile.jpeg")
	assert.Equal(len(tokens), 3)
	// Ignored in KorAP-Tokenizer

	// testTokenizerPunct
	tokens = ttokenize(mat_de, w, "Er sagte: \"Es geht mir gut!\", daraufhin ging er.")
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
	tokens = ttokenize(mat_de, w, "&quot;Das ist von C&A!&quot;")
	assert.Equal(tokens[0], "&quot;")
	assert.Equal(tokens[1], "Das")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "von")
	assert.Equal(tokens[4], "C&A")
	assert.Equal(tokens[5], "!")
	assert.Equal(tokens[6], "&quot;")
	assert.Equal(len(tokens), 7)

	// testTokenizerLongEnd
	tokens = ttokenize(mat_de, w, "Siehst Du?!!?")
	assert.Equal(tokens[0], "Siehst")
	assert.Equal(tokens[1], "Du")
	assert.Equal(tokens[2], "?!!?")
	assert.Equal(len(tokens), 3)

	// testTokenizerIrishO
	tokens = ttokenize(mat_de, w, "Peter O'Toole")
	assert.Equal(tokens[0], "Peter")
	assert.Equal(tokens[1], "O'Toole")
	assert.Equal(len(tokens), 2)

	// testTokenizerAbr
	tokens = ttokenize(mat_de, w, "Früher bzw. später ...")
	assert.Equal(tokens[0], "Früher")
	assert.Equal(tokens[1], "bzw.")
	assert.Equal(tokens[2], "später")
	assert.Equal(tokens[3], "...")
	assert.Equal(len(tokens), 4)

	// testTokenizerUppercaseRule
	tokens = ttokenize(mat_de, w, "Es war spät.Morgen ist es früh.")
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
	tokens = ttokenize(mat_de, w, "Sie erreichte den 1. Platz!")
	assert.Equal(tokens[0], "Sie")
	assert.Equal(tokens[1], "erreichte")
	assert.Equal(tokens[2], "den")
	assert.Equal(tokens[3], "1.")
	assert.Equal(tokens[4], "Platz")
	assert.Equal(tokens[5], "!")
	assert.Equal(len(tokens), 6)

	// testNoZipOuputArchive
	tokens = ttokenize(mat_de, w, "Archive:  Ich bin kein zip\n")
	assert.Equal(tokens[0], "Archive")
	assert.Equal(tokens[1], ":")
	assert.Equal(tokens[2], "Ich")
	assert.Equal(tokens[3], "bin")
	assert.Equal(tokens[4], "kein")
	assert.Equal(tokens[5], "zip")
	assert.Equal(6, len(tokens))

	// testTokenizerStrasse
	tokens = ttokenize(mat_de, w, "Ich wohne in der Weststr. und Du?")
	assert.Equal(tokens[4], "Weststr.")
	assert.Equal(8, len(tokens))

	// germanTokenizerKnowsGermanOmissionWords
	tokens = ttokenize(mat_de, w, "D'dorf Ku'damm Lu'hafen M'gladbach W'schaft")
	assert.Equal("D'dorf", tokens[0])
	assert.Equal("Ku'damm", tokens[1])
	assert.Equal("Lu'hafen", tokens[2])
	assert.Equal("M'gladbach", tokens[3])
	assert.Equal("W'schaft", tokens[4])
	assert.Equal(5, len(tokens))

	// germanTokenizerDoesNOTSeparateGermanContractions
	tokens = ttokenize(mat_de, w, "mach's macht's was'n ist's haste willste kannste biste kriegste")
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

	tokens = ttokenize(mat_de, w, "Es ist gleich 2:30 Uhr.")
	assert.Equal("Es", tokens[0])
	assert.Equal("ist", tokens[1])
	assert.Equal("gleich", tokens[2])
	assert.Equal("2:30", tokens[3])
	assert.Equal("Uhr", tokens[4])
	assert.Equal(".", tokens[5])
	assert.Equal(6, len(tokens))

	tokens = ttokenize(mat_de, w, "Sie schwamm die Strecke in 00:00:57,34 00:57,341 0:57 Stunden.")
	assert.Equal("Sie", tokens[0])
	assert.Equal("schwamm", tokens[1])
	assert.Equal("die", tokens[2])
	assert.Equal("Strecke", tokens[3])
	assert.Equal("in", tokens[4])
	assert.Equal("00:00:57,34", tokens[5])
	assert.Equal("00:57,341", tokens[6])
	assert.Equal("0:57", tokens[7])
	assert.Equal("Stunden", tokens[8])
	assert.Equal(".", tokens[9])
	assert.Equal(10, len(tokens))

	// waste example
	tokens = ttokenize(mat_de, w, "Am 24.1.1806 feierte E. T. A. Hoffmann seinen 30. Geburtstag.")
	assert.Equal(tokens[0], "Am")
	assert.Equal(tokens[1], "24.1.1806")
	assert.Equal(tokens[2], "feierte")
	assert.Equal(tokens[3], "E.")
	assert.Equal(tokens[4], "T.")
	assert.Equal(tokens[5], "A.")
	assert.Equal(tokens[6], "Hoffmann")
	assert.Equal(tokens[7], "seinen")
	assert.Equal(tokens[8], "30.")
	assert.Equal(tokens[9], "Geburtstag")
	assert.Equal(tokens[10], ".")
	assert.Equal(11, len(tokens))

	// IPtest
	tokens = ttokenize(mat_de, w, "Meine IP ist 192.178.168.55.")
	assert.Equal(tokens[0], "Meine")
	assert.Equal(tokens[1], "IP")
	assert.Equal(tokens[2], "ist")
	assert.Equal(tokens[3], "192.178.168.55")
	assert.Equal(tokens[4], ".")
	assert.Equal(5, len(tokens))

	// XML entities
	tokens = ttokenize(mat_de, w, "Das ist&nbsp;1:30 Stunden&20 Minuten zu spät &GT;.")
	assert.Equal(tokens[0], "Das")
	assert.Equal(tokens[1], "ist")
	assert.Equal(tokens[2], "&nbsp;")
	assert.Equal(tokens[3], "1:30")
	assert.Equal(tokens[4], "Stunden")
	assert.Equal(tokens[5], "&")
	assert.Equal(tokens[6], "20")
	assert.Equal(tokens[7], "Minuten")
	assert.Equal(tokens[8], "zu")
	assert.Equal(tokens[9], "spät")
	assert.Equal(tokens[10], "&GT;")
	assert.Equal(tokens[11], ".")
	assert.Equal(12, len(tokens))

	// Plusampersand compounds (1)
	tokens = ttokenize(mat_de, w, "Die 2G+-Regel soll weitere Covid-19-Erkrankungen reduzieren.")
	assert.Equal(tokens[0], "Die")
	assert.Equal(tokens[1], "2G+-Regel")
	assert.Equal(tokens[2], "soll")
	assert.Equal(tokens[3], "weitere")
	assert.Equal(tokens[4], "Covid-19-Erkrankungen")
	assert.Equal(tokens[5], "reduzieren")
	assert.Equal(tokens[6], ".")
	assert.Equal(7, len(tokens))

	// Plusampersand compounds (2)
	tokens = ttokenize(mat_de, w, "Der Neu-C++-Programmierer.")
	assert.Equal(tokens[0], "Der")
	assert.Equal(tokens[1], "Neu-C++-Programmierer")
	assert.Equal(tokens[2], ".")
	assert.Equal(3, len(tokens))

	// z.B.
	tokens = ttokenize(mat_de, w, "Dies sind z.B. zwei Wörter - z. B. auch.")
	assert.Equal(tokens[0], "Dies")
	assert.Equal(tokens[1], "sind")
	assert.Equal(tokens[2], "z.")
	assert.Equal(tokens[3], "B.")
	assert.Equal(tokens[4], "zwei")
	assert.Equal(tokens[5], "Wörter")
	assert.Equal(tokens[6], "-")
	assert.Equal(tokens[7], "z.")
	assert.Equal(tokens[8], "B.")
	assert.Equal(tokens[9], "auch")
	assert.Equal(tokens[10], ".")
	assert.Equal(11, len(tokens))

	// z.B.
	tokens = ttokenize(mat_de, w, "Dies sind z.B. zwei Wörter - z. B. auch.")
	assert.Equal(tokens[0], "Dies")
	assert.Equal(tokens[1], "sind")
	assert.Equal(tokens[2], "z.")
	assert.Equal(tokens[3], "B.")
	assert.Equal(tokens[4], "zwei")
	assert.Equal(tokens[5], "Wörter")
	assert.Equal(tokens[6], "-")
	assert.Equal(tokens[7], "z.")
	assert.Equal(tokens[8], "B.")
	assert.Equal(tokens[9], "auch")
	assert.Equal(tokens[10], ".")
	assert.Equal(11, len(tokens))

	// Single quote handling
	tokens = ttokenize(mat_de, w, "Es heißt 'Leitungssportteams' und nicht anders.")
	assert.Equal(tokens[0], "Es")
	assert.Equal(tokens[1], "heißt")
	assert.Equal(tokens[2], "'")
	assert.Equal(tokens[3], "Leitungssportteams")
	assert.Equal(tokens[4], "'")
	assert.Equal(tokens[5], "und")
	assert.Equal(tokens[6], "nicht")
	assert.Equal(tokens[7], "anders")
	assert.Equal(tokens[8], ".")
	assert.Equal(9, len(tokens))

	// Apostrophe handling
	tokens = ttokenize(mat_de, w, "Das ist Nils’ Einkaufskorb bei McDonald's.")
	assert.Equal(tokens[0], "Das")
	assert.Equal(tokens[1], "ist")
	assert.Equal(tokens[2], "Nils’")
	assert.Equal(tokens[3], "Einkaufskorb")
	assert.Equal(tokens[4], "bei")
	assert.Equal(tokens[5], "McDonald's")
	assert.Equal(tokens[6], ".")
	assert.Equal(7, len(tokens))

}

func TestMatrixFullTokenizerTokenSplitterEN(t *testing.T) {
	assert := assert.New(t)

	if mat_en == nil {
		mat_en = LoadMatrixFile("testdata/tokenizer_en.matok")
	}

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	// testEnglishTokenizerScienceAbbreviations
	tokens = ttokenize(mat_en, w, "Approx. in Sept. 1954, Assoc. Prof. Dr. R. J. Ewing reviewed articles on Enzymol. Bacteriol. effects later published in Nutr. Rheumatol. No. 12 and Nº. 13., pp. 17-18.")
	assert.Equal("Approx.", tokens[0])
	assert.Equal("in", tokens[1])
	assert.Equal("Sept.", tokens[2])
	assert.Equal("1954", tokens[3])
	assert.Equal(",", tokens[4])
	assert.Equal("Assoc.", tokens[5])
	assert.Equal("Prof.", tokens[6])
	assert.Equal("Dr.", tokens[7])
	assert.Equal("R.", tokens[8])
	assert.Equal("J.", tokens[9])
	assert.Equal("Ewing", tokens[10])
	assert.Equal("reviewed", tokens[11])
	assert.Equal("articles", tokens[12])
	assert.Equal("on", tokens[13])
	assert.Equal("Enzymol.", tokens[14])
	assert.Equal("Bacteriol.", tokens[15])
	assert.Equal("effects", tokens[16])
	assert.Equal("later", tokens[17])
	assert.Equal("published", tokens[18])
	assert.Equal("in", tokens[19])
	assert.Equal("Nutr.", tokens[20])
	assert.Equal("Rheumatol.", tokens[21])
	assert.Equal("No.", tokens[22])
	assert.Equal("12", tokens[23])
	assert.Equal("and", tokens[24])
	assert.Equal("Nº.", tokens[25])
	assert.Equal("13.", tokens[26])
	assert.Equal(",", tokens[27])
	assert.Equal("pp.", tokens[28])
	assert.Equal("17-18", tokens[29])
	assert.Equal(".", tokens[30])

	// englishTokenizerCanGuessWhetherIIsAbbrev
	tokens = ttokenize(mat_en, w, "M. I. Baxter was born during World War I. So was I. He went to the Peter I. Hardy school. So did I.")
	assert.Equal("I.", tokens[1])
	assert.Equal("I", tokens[8])
	assert.Equal(".", tokens[9])
	assert.Equal("I", tokens[12])
	assert.Equal(".", tokens[13])

	// englishTokenizerSeparatesEnglishContractionsAndClitics
	tokens = ttokenize(mat_en, w, "I've we'll you'd I'm we're Peter's isn't who'll've")
	assert.Equal("I", tokens[0])
	assert.Equal("'ve", tokens[1])
	assert.Equal("'ll", tokens[3])
	assert.Equal("'d", tokens[5])
	assert.Equal("'m", tokens[7])
	assert.Equal("'re", tokens[9])
	assert.Equal("'s", tokens[11])
	assert.Equal("is", tokens[12])
	assert.Equal("n't", tokens[13])
	assert.Equal("who", tokens[14])
	assert.Equal("'ll", tokens[15])
	assert.Equal("'ve", tokens[16])
	assert.Equal(17, len(tokens))
	/*
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

func TestMatrixEmoticons(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	assert.NotNil(mat_de)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	tokens = ttokenize(mat_de, w, ":-* ;) :)) :*( ^___^ T__T ^^; -_-;;; -_-^")
	assert.Equal(tokens[0], ":-*")
	assert.Equal(tokens[1], ";)")
	assert.Equal(tokens[2], ":))")
	assert.Equal(tokens[3], ":*(")
	assert.Equal(tokens[4], "^___^")
	assert.Equal(tokens[5], "T__T")
	assert.Equal(tokens[6], "^^;")
	assert.Equal(tokens[7], "-_-;;;")
	assert.Equal(tokens[8], "-_-^")
	assert.Equal(len(tokens), 9)

	tokens = ttokenize(mat_de, w, "das -> Lustig<-!")
	assert.Equal("das", tokens[0])
	assert.Equal("->", tokens[1])
	assert.Equal("Lustig", tokens[2])
	assert.Equal("<-", tokens[3])
	assert.Equal("!", tokens[4])
	assert.Equal(5, len(tokens))
}

func TestMatrixFullTokenizerXML(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	assert.NotNil(mat_de)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)
	var tokens []string

	tokens = ttokenize(mat_de, w, "Das <b>beste</b> Fußballspiel")
	assert.Equal("Das", tokens[0])
	assert.Equal("<b>", tokens[1])
	assert.Equal("beste", tokens[2])
	assert.Equal("</b>", tokens[3])
	assert.Equal("Fußballspiel", tokens[4])
	assert.Equal(5, len(tokens))

	tokens = ttokenize(mat_de, w, "Das <b class=\"c\">beste</b> Fußballspiel")
	assert.Equal("Das", tokens[0])
	assert.Equal("<b class=\"c\">", tokens[1])
	assert.Equal("beste", tokens[2])
	assert.Equal("</b>", tokens[3])
	assert.Equal("Fußballspiel", tokens[4])
	assert.Equal(5, len(tokens))

	tokens = ttokenize(mat_de, w, "der<x  y=\"alte \"> <x x> alte</x> etc. et. Mann.")
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

	tokens = ttokenize(mat_de, w, "das<br   class=\"br\" />ging.")
	assert.Equal("das", tokens[0])
	assert.Equal("<br   class=\"br\" />", tokens[1])
	assert.Equal("ging", tokens[2])
	assert.Equal(".", tokens[3])
	assert.Equal(4, len(tokens))

	tokens = ttokenize(mat_de, w, "das  <?robot xgh ?>  <!-- hm hm -->   <![CDATA[ cdata ]]>  <br />")
	assert.Equal("das", tokens[0])
	assert.Equal("<?robot", tokens[1])
	assert.Equal("xgh", tokens[2])
	assert.Equal("?>", tokens[3])
	assert.Equal("<!--", tokens[4])
	assert.Equal("hm", tokens[5])
	assert.Equal("hm", tokens[6])
	assert.Equal("-->", tokens[7])
	assert.Equal("<![CDATA[", tokens[8])
	assert.Equal("cdata", tokens[9])
	assert.Equal("]]>", tokens[10])
	assert.Equal("<br />", tokens[11])
	assert.Equal(12, len(tokens))

}

func TestMatokDatokEquivalence(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}
	dat := LoadDatokFile("testdata/tokenizer_de.datok")

	r := strings.NewReader(s)

	tb := make([]byte, 0, 2048)
	w := bytes.NewBuffer(tb)

	// Transduce with double array representation
	dat.Transduce(r, w)

	datStr := w.String()

	r.Reset(s)
	w.Reset()

	// Transduce with matrix representation
	mat_de.Transduce(r, w)

	matStr := w.String()

	assert.Equal(datStr, matStr)
}

func TestMatrixFullTokenizerCallbackTransduce(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	assert.NotNil(mat_de)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)

	assert.True(mat_de.Transduce(strings.NewReader("Der alte Baum. Er war schon alt."), w))

	matStr := w.String()

	assert.Equal("Der\nalte\nBaum\n.\n\nEr\nwar\nschon\nalt\n.\n\n\n", matStr)
}

func TestMatrixFullTokenizerTextTreatment(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	assert.NotNil(mat_de)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)

	assert.True(mat_de.Transduce(strings.NewReader("Erste.\n\n\n\n\x04\x0aNächst.\x04"), w))
	matStr := w.String()
	assert.Equal("Erste\n.\n\n\nNächst\n.\n\n\n", matStr)
}

func TestMatrixFullTokenizerLongText(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	assert.NotNil(mat_de)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)

	text := `The Project Gutenberg EBook of Effi Briest, by Theodor Fontane

Copyright laws are changing all over the world. Be sure to check the
copyright laws for your country before downloading or redistributing
this or any other Project Gutenberg eBook.

This header should be the first thing seen when viewing this Project
Gutenberg file.  Please do not remove it.  Do not change or edit the
header without written permission.

Please read the "legal small print," and other information about the
eBook and Project Gutenberg at the bottom of this file.  Included is
important information about your specific rights and restrictions in
how the file may be used.  You can also find out about how to make a
donation to Project Gutenberg, and how to get involved.


**Welcome To The World of Free Plain Vanilla Electronic Texts**

**eBooks Readable By Both Humans and By Computers, Since 1971**

*****These eBooks Were Prepared By Thousands of Volunteers!*****


Title: Effi Briest

Author: Theodor Fontane

Release Date: March, 2004  [EBook #5323]
`

	assert.True(mat_de.Transduce(strings.NewReader(text), w))

	assert.True(strings.Contains(w.String(), "Release"))
}

func TestMatrixTrimming(t *testing.T) {
	assert := assert.New(t)

	if mat_de == nil {
		mat_de = LoadMatrixFile("testdata/tokenizer_de.matok")
	}

	assert.NotNil(mat_de)

	b := make([]byte, 0, 2048)
	w := bytes.NewBuffer(b)

	assert.True(mat_de.Transduce(strings.NewReader("  Erste."), w))
	matStr := w.String()
	assert.Equal("Erste\n.\n\n\n", matStr)
}

func BenchmarkMatrixTransduce(b *testing.B) {
	bu := make([]byte, 0, 2048)
	w := bytes.NewBuffer(bu)

	r := strings.NewReader(s)

	mat := LoadMatrixFile("testdata/tokenizer_de.matok")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Reset()
		r.Reset(s)
		ok := mat.Transduce(r, w)
		if !ok {
			fmt.Println("Fail!")
			fmt.Println(w.String())
			os.Exit(1)
		}
	}
}

package main

import (
	"fmt"
	"io"
	"os"

	"log"

	datok "github.com/KorAP/datok"
	"github.com/alecthomas/kong"
)

// TODO:
// - Support version information based on
//   https://blog.carlmjohnson.net/post/2021/golang-118-minor-features/

var cli struct {
	Convert struct {
		Foma        string `kong:"required,short='i',help='The Foma FST file'"`
		Tokenizer   string `kong:"required,short='o',help='The Tokenizer file'"`
		DoubleArray bool   `kong:"optional,short='d',help='Convert to Double Array instead of Matrix representation'"`
	} `kong:"cmd, help='Convert a compiled foma FST file to a Matrix or Double Array tokenizer'"`
	Tokenize struct {
		Tokenizer         string `kong:"required,short='t',help='The Matrix or Double Array Tokenizer file'"`
		Input             string `kong:"required,arg='',type='existingfile',help='Input file to tokenize (use - for STDIN)'"`
		Tokens            bool   `kong:"optional,negatable,default=true,help='Print token surfaces (defaults to ${default})'"`
		Sentences         bool   `kong:"optional,negatable,default=true,help='Print sentence boundaries (defaults to ${default})'"`
		TokenPositions    bool   `kong:"optional,default=false,short='p',help='Print token offsets (defaults to ${default})'"`
		SentencePositions bool   `kong:"optional,default=false,help='Print sentence offsets (defaults to ${default})'"`
		NewlineAfterEOT   bool   `kong:"optional,default=false,help='Ignore newline after EOT (defaults to ${default})'"`
	} `kong:"cmd, help='Tokenize a text'"`
}

// Main method for command line handling
func main() {

	// Parse command line parameters
	parser := kong.Must(
		&cli,
		kong.Name("datok"),
		kong.Description("FSA based tokenizer"),
		kong.UsageOnError(),
	)

	ctx, err := parser.Parse(os.Args[1:])

	parser.FatalIfErrorf(err)

	if ctx.Command() == "convert" {
		tok := datok.LoadFomaFile(cli.Convert.Foma)
		if tok == nil {
			log.Fatalln("Unable to load foma file")
		}
		if cli.Convert.DoubleArray {
			dat := tok.ToDoubleArray()
			fmt.Println("Load factor", dat.LoadFactor())
			_, err := dat.Save(cli.Convert.Tokenizer)
			if err != nil {
				log.Fatalln(err)
			}
		} else {
			mat := tok.ToMatrix()
			_, err := mat.Save(cli.Convert.Tokenizer)
			if err != nil {
				log.Fatalln(err)
			}
		}
		fmt.Println("File successfully converted.")
		os.Exit(0)
	}

	// Load the Datok or Matrix file
	dat := datok.LoadTokenizerFile(cli.Tokenize.Tokenizer)

	// Unable to load the datok file
	if dat == nil {
		log.Fatalln("Unable to load file")
		os.Exit(1)
	}

	// Create flags parameter based on command line parameters
	var flags datok.Bits
	if cli.Tokenize.Tokens {
		flags |= datok.TOKENS
	}

	if cli.Tokenize.TokenPositions {
		flags |= datok.TOKEN_POS
	}

	if cli.Tokenize.Sentences {
		flags |= datok.SENTENCES
	}

	if cli.Tokenize.SentencePositions {
		flags |= datok.SENTENCE_POS
	}

	if cli.Tokenize.NewlineAfterEOT {
		flags |= datok.NEWLINE_AFTER_EOT
	}

	// Create token writer based on the options defined
	tw := datok.NewTokenWriter(os.Stdout, flags)
	defer os.Stdout.Close()

	var r io.Reader

	// Program is running in a pipe
	if cli.Tokenize.Input == "-" {
		fileInfo, _ := os.Stdin.Stat()
		if fileInfo.Mode()&os.ModeCharDevice == 0 {
			r = os.Stdin
			defer os.Stdin.Close()
		} else {
			log.Fatalln("Unable to read from STDIN")
			os.Exit(1)
			return
		}
	} else {
		f, err := os.Open(cli.Tokenize.Input)
		if err != nil {
			log.Fatalln(err)
			os.Exit(1)
			return
		}
		defer f.Close()
		r = f
	}

	dat.TransduceTokenWriter(r, tw)
	tw.Flush()
}

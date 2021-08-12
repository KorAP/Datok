package main

import (
	"fmt"
	"os"

	"log"

	datok "github.com/KorAP/datokenizer"
	"github.com/alecthomas/kong"
)

var cli struct {
	Convert struct {
		Foma      string `kong:"required,short='i',help='The Foma file'"`
		Tokenizer string `kong:"required,short='o',help='The Double Array Tokenizer file'"`
	} `kong:"cmd, help='Convert a foma file to a double array tokenizer'"`
	Tokenize struct {
		Tokenizer string `kong:"required,short='t',help='The Double Array Tokenizer file'"`
	} `kong:"cmd, help='Tokenize a text'"`
}

// Main method for command line handling
func main() {

	// Parse command line parameters
	parser := kong.Must(
		&cli,
		kong.Name("datok"),
		kong.Description("Double Array based tokenizer"),
		kong.UsageOnError(),
	)

	ctx, err := parser.Parse(os.Args[1:])

	parser.FatalIfErrorf(err)

	if ctx.Command() == "convert" {
		tok := datok.LoadFomaFile(cli.Convert.Foma)
		if tok == nil {
			log.Fatalln("Unable to load foma file")
		}
		dat := tok.ToDoubleArray()
		_, err := dat.Save(cli.Convert.Tokenizer)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("File successfully converted.")
		os.Exit(0)
	}

	// Load the Datok file
	dat := datok.LoadDatokFile(cli.Tokenize.Tokenizer)

	// Unable to load the datok file
	if dat == nil {
		os.Exit(1)
	}

	// Program is running in a pipe
	fileInfo, _ := os.Stdin.Stat()
	if fileInfo.Mode()&os.ModeCharDevice == 0 {

		// Transduce from STDIN and write to STDOUT
		dat.Transduce(os.Stdin, os.Stdout)
	}
}

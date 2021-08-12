package main

import (
	"os"

	datok "github.com/KorAP/datokenizer"
	"github.com/alecthomas/kong"
)

var cli struct {
	Tokenizer string `kong:"required,short='t',help='The Double Array Tokenizer file'"`
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

	_, err := parser.Parse(os.Args[1:])

	parser.FatalIfErrorf(err)

	// Load the Datok file
	dat := datok.LoadDatokFile(cli.Tokenizer)

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

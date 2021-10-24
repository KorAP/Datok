package main

import (
	"fmt"
	"os"

	"log"

	datok "github.com/KorAP/datok"
	"github.com/alecthomas/kong"
)

var cli struct {
	Convert struct {
		Foma        string `kong:"required,short='i',help='The Foma file'"`
		Tokenizer   string `kong:"required,short='o',help='The Tokenizer file'"`
		DoubleArray bool   `kong:"optional,short='d',help='Convert to Double Array instead of Matrix representation'"`
	} `kong:"cmd, help='Convert a foma file to a Matrix or Double Array tokenizer'"`
	Tokenize struct {
		Tokenizer string `kong:"required,short='t',help='The Matrix or Double Array Tokenizer file'"`
		Positions bool   `kong:"optional,negatable,default=false,short='p',help='Print token offsets'"`
		Tokens    bool   `kong:"optional,negatable,default=true,help="Print token surfaces""`
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

	// Create token writer based on the options defined
	tw := datok.NewTokenWriterFromOptions(os.Stdout, cli.Tokenize.Positions)

	// Program is running in a pipe
	fileInfo, _ := os.Stdin.Stat()
	if fileInfo.Mode()&os.ModeCharDevice == 0 {
		dat.TransduceTokenWriter(os.Stdin, tw)
		tw.Flush()
	}
}

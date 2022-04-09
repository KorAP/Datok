# Datok - Finite State Tokenizer

![Introduction to Datok](https://raw.githubusercontent.com/KorAP/Datok/master/misc/introducing-datok.gif)

Implementation of a finite state automaton for
fast natural language tokenization, based on a finite state
transducer generated with [Foma](https://fomafst.github.io/).

The library contains sources for a german tokenizer
based on [KorAP-Tokenizer](https://github.com/KorAP/KorAP-Tokenizer).


## Tokenization

```
Usage: datok tokenize --tokenizer=STRING <input>

Arguments:
  <input>    Input file to tokenize (use - for STDIN)

Flags:
  -h, --help                  Show context-sensitive help.

  -t, --tokenizer=STRING      The Matrix or Double Array Tokenizer file
      --[no-]tokens           Print token surfaces (defaults to true)
      --[no-]sentences        Print sentence boundaries (defaults to true)
  -p, --token-positions       Print token offsets (defaults to false)
      --sentence-positions    Print sentence offsets (defaults to false)
      --newline-after-eot     Ignore newline after EOT (defaults to false)
```

The special `END OF TRANSMISSION` character (`\x04`) can be used to mark the end of a text.

> *Caution*: When experimenting with STDIN and echo,
> you may need to disable history expansion.


## Conversion

```
Usage: datok convert --foma=STRING --tokenizer=STRING

Flags:
  -h, --help                Show context-sensitive help.

  -i, --foma=STRING         The Foma FST file
  -o, --tokenizer=STRING    The Tokenizer file
  -d, --double-array        Convert to Double Array instead of Matrix
                            representation
```

## Conventions

The FST generated by Foma must adhere to the following rules,
to be converted by Datok:

- Character accepting arcs need to be translated
  *only* to themselves or to ε (the empty symbol).
- Multi-character symbols are not allowed,
  except for the `@_TOKEN_SYMBOL_@`,
  that denotes the end of a token.
- ε accepting arcs (transitions not consuming
  any character) need to be translated to
  the `@_TOKEN_SYMBOL_@`.
- Two consecutive `@_TOKEN_SYMBOL_@`s mark a sentence end.
- Flag diacritics are not supported.
- Final states are ignored. The `@_TOKEN_SYMBOL_@` marks
  the end of a token instead.

A minimal usable tokenizer written in XFST and following
the guidelines to tokenizers in Beesley and Karttunen (2003)
and Beesley (2004) would look like this:

```xfst
define TE "@_TOKEN_SYMBOL_@";

define WS [" "|"\u000a"|"\u0009"];

define PUNCT ["."|"?"|"!"];

define Char \[WS|PUNCT];

define Word Char+;

! Compose token ends
define Tokenizer [[Word|PUNCT] @-> ... TE] .o.
! Compose Whitespace ignorance
       [WS+ @-> 0] .o.
! Compose sentence ends
       [[PUNCT+] @-> ... TE \/ TE _ ];

read regex Tokenizer;
```

> *Hint*: For development it's easier to replace `@_TOKEN_SYMBOL_@`
with a newline.

## Building

To build the tokenizer tool, run

```shell
$ go build ./cmd/datok.go
```

To create a foma file from the example sources, first install
[Foma](https://fomafst.github.io/), then run in
the root directory of this repository

```shell
$ cd src && \
  foma -e "source de/tokenizer.xfst" \
  -e "save stack ../mytokenizer.fst" -q -s && \
  cd ..
```

This will load and compile the german `tokenizer.xfst`
and will save the compiled FST as `mytokenizer.fst`
in the root directory.

To generate a Datok FSA (matrix representation) based on
this FST, run

```shell
$ datok convert -i mytokenizer.fst -o mytokenizer.datok
```

To generate a Datok FSA (double array representation*) based
on this FST, run

```shell
$ datok convert -i mytokenizer.fst -o mytokenizer.datok -d
```

The final datok file can then be used as a model for the tokenizer.

* This may take quite some time depending on the number
of arcs in the FST and is therefore now deprecated.


## Technology

Internally the FSA is represented
either as a matrix or as a double array.

Both representations mark all non-word-character targets with a
leading bit. All ε (aka *tokenend*) transitions mark the end of a
token or the end of a sentence (2 subsequential ε).
The transduction is greedy with a single backtracking
option to the last ε transition.

The double array representation (Aoe 1989) of all transitions
in the FST is implemented as an extended DFA following Mizobuchi
et al. (2000) and implementation details following Kanda et al. (2018).


## License

Datok is published under the [Apache 2.0 License](LICENSE).

The german tokenizer shipped is based on work done by the
[Lucene project](https://github.com/apache/lucene-solr)
(published under the Apache License),
[David Hall](https://github.com/dlwh/epic)
(published under the Apache License),
[Çağrı Çöltekin](https://github.com/coltekin/TRmorph/)
(published under the MIT License),
and [Marc Kupietz](https://github.com/KorAP/KorAP-Tokenizer)
(published under the Apache License).

The foma parser is based on
[*foma2js*](https://github.com/mhulden/foma),
written by Mans Hulden (published under the Apache License).


## Bibliography

Aoe, Jun-ichi (1989): *An Efficient Digital Search Algorithm by Using a Double-Array Structure*.
IEEE Transactions on Software Engineering, 15 (9), pp. 1066-1077.

Beesley, Kenneth R. & Lauri Karttunen (2003): *Finite State Morphology*. Stanford, CA: CSLI Publications.

Beesley, Kenneth R. (2004): *Tokenizing Transducers*.
[https://web.stanford.edu/~laurik/fsmbook/clarifications/tokfst.html](https://web.stanford.edu/~laurik/fsmbook/clarifications/tokfst.html)

Hulden, Mans (2009): *Foma: a finite-state compiler and library*. In: Proceedings of the
12th Conference of the European Chapter of the Association for Computational Linguistics,
Association for Computational Linguistics, pp. 29-32.

Mizobuchi, Shoji, Toru Sumitomo, Masao Fuketa & Jun-ichi Aoe (2000):
*An efficient representation for implementing finite state machines based on the double-array*.
Information Sciences 129, pp. 119-139.

Kanda, Shunsuke, Yuma Fujita, Kazuhiro Morita & Masao Fuketa (2018):
*Practical rearrangement methods for dynamic double-array dictionaries*.
Software: Practice and Experience (SPE), 48(1), pp. 65–83.

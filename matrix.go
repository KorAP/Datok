package datok

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	MAMAGIC       = "MATOK"
	SYMBOLSUPPORT = true
	EOT           = 4
)

type MatrixTokenizer struct {
	sigma      map[rune]int
	sigmaASCII [256]int
	array      []uint32
	stateCount int

	// Special symbols in sigma
	epsilon  int
	unknown  int
	identity int
}

// ToMatrix turns the intermediate tokenizer into a
// matrix representation.
func (auto *Automaton) ToMatrix() *MatrixTokenizer {

	mat := &MatrixTokenizer{
		sigma:      make(map[rune]int),
		unknown:    auto.unknown,
		identity:   auto.identity,
		epsilon:    auto.epsilon,
		stateCount: auto.stateCount,
	}

	max := 0

	// Init with identity
	if mat.identity != -1 {
		for i := 0; i < 256; i++ {
			mat.sigmaASCII[i] = mat.identity
		}
		max = mat.identity
	}

	for num, sym := range auto.sigmaRev {
		if int(sym) < 256 {
			mat.sigmaASCII[int(sym)] = num
		}
		mat.sigma[sym] = num
		if num > auto.sigmaCount {
			panic("sigmaCount is smaller")
		}

		// Find max
		// see https://dev.to/jobinrjohnson/branchless-programming-does-it-really-matter-20j4
		max -= ((max - num) & ((max - num) >> 31))
		// if num > max {
		// 	 max = num
		// }
	}
	// Add final entry to the list (maybe not necessary actually)

	remember := make([]bool, auto.stateCount+2)

	// lower sigmaCount, as no final value exists
	mat.array = make([]uint32, (auto.stateCount+1)*(max+1))

	// Store all transitions in matrix
	var toMatrix func([]uint32, int)

	toMatrix = func(matrix []uint32, start int) {
		if start > auto.stateCount {
			panic("stateCount is smaller")
		}
		if remember[start] {
			return
		}
		remember[start] = true
		for alpha, t := range auto.transitions[start] {
			matrix[(alpha-1)*auto.stateCount+start] = uint32(t.end)

			// Mark nontoken transitions
			if t.nontoken {
				matrix[(alpha-1)*auto.stateCount+start] |= FIRSTBIT
			}

			toMatrix(matrix, t.end)
		}
	}

	toMatrix(mat.array, 1)

	return mat
}

// Type of tokenizer
func (MatrixTokenizer) Type() string {
	return MAMAGIC
}

// Save stores the matrix data in a file
func (mat *MatrixTokenizer) Save(file string) (n int64, err error) {
	f, err := os.Create(file)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	n, err = mat.WriteTo(gz)
	if err != nil {
		log.Println(err)
		return n, err
	}
	gz.Flush()
	return n, nil
}

// WriteTo stores the matrix data in an io.Writer.
func (mat *MatrixTokenizer) WriteTo(w io.Writer) (n int64, err error) {

	wb := bufio.NewWriter(w)
	defer wb.Flush()

	// Store magical header
	all, err := wb.Write([]byte(MAMAGIC))
	if err != nil {
		log.Println(err)
		return int64(all), err
	}

	// Get sigma as a list
	// In datok it's 16 - 4*4
	sigmalist := make([]rune, len(mat.sigma)+16)
	max := 0
	for sym, num := range mat.sigma {
		sigmalist[num] = sym

		// Find max
		// see https://dev.to/jobinrjohnson/branchless-programming-does-it-really-matter-20j4
		max -= ((max - num) & ((max - num) >> 31))
		// if num > max {
		// 	max = num
		// }
	}

	// Add final entry to the list (maybe not necessary actually)
	sigmalist = sigmalist[:max+1]

	buf := make([]byte, 0, 14)
	bo.PutUint16(buf[0:2], VERSION)
	bo.PutUint16(buf[2:4], uint16(mat.epsilon))
	bo.PutUint16(buf[4:6], uint16(mat.unknown))
	bo.PutUint16(buf[6:8], uint16(mat.identity))
	bo.PutUint32(buf[8:12], uint32(mat.stateCount))
	bo.PutUint16(buf[12:14], uint16(len(sigmalist)))
	more, err := wb.Write(buf[0:14])
	if err != nil {
		log.Println(err)
		return int64(all), err
	}

	all += more

	// Write sigma
	for _, sym := range sigmalist {

		more, err = wb.WriteRune(sym)
		if err != nil {
			log.Println(err)
			return int64(all), err
		}
		all += more
	}

	if err != nil {
		log.Println(err)
		return int64(all), err
	}

	// Test marker - could be checksum
	more, err = wb.Write([]byte("M"))
	if err != nil {
		log.Println(err)
		return int64(all), err
	}
	all += more

	for _, x := range mat.array {
		bo.PutUint32(buf[0:4], uint32(x))
		more, err = wb.Write(buf[0:4])
		if err != nil {
			log.Println(err)
			return int64(all), err
		}
		all += more
		if more != 4 {
			log.Println("Can not write base uint32")
			return int64(all), err
		}
	}

	return int64(all), err
}

// LoadDatokFile reads a double array represented tokenizer
// from a file.
func LoadMatrixFile(file string) *MatrixTokenizer {
	f, err := os.Open(file)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer gz.Close()

	// Todo: Read the whole file!
	return ParseMatrix(gz)
}

// LoadMatrixFile reads a matrix represented tokenizer
// from an io.Reader
func ParseMatrix(ior io.Reader) *MatrixTokenizer {

	// Initialize tokenizer with default values
	mat := &MatrixTokenizer{
		sigma:      make(map[rune]int),
		epsilon:    0,
		unknown:    0,
		identity:   0,
		stateCount: 0,
	}

	r := bufio.NewReader(ior)

	buf := make([]byte, 1024)
	buf = buf[0:len(MAMAGIC)]

	_, err := r.Read(buf)

	if err != nil {
		log.Println(err)
		return nil
	}

	if string(MAMAGIC) != string(buf) {
		log.Println("Not a matok file")
		return nil
	}

	more, err := io.ReadFull(r, buf[0:14])
	if err != nil {
		log.Println(err)
		return nil
	}

	if more != 14 {
		log.Println("Read bytes do not fit")
		return nil
	}

	version := bo.Uint16(buf[0:2])

	if version != VERSION {
		log.Println("Version not compatible")
		return nil
	}

	mat.epsilon = int(bo.Uint16(buf[2:4]))
	mat.unknown = int(bo.Uint16(buf[4:6]))
	mat.identity = int(bo.Uint16(buf[6:8]))
	mat.stateCount = int(bo.Uint32(buf[8:12]))
	sigmaCount := int(bo.Uint16(buf[12:14]))
	arraySize := (mat.stateCount + 1) * sigmaCount

	// Init with identity
	if mat.identity != -1 {
		for i := 0; i < 256; i++ {
			mat.sigmaASCII[i] = mat.identity
		}
	}

	for x := 0; x < sigmaCount; x++ {
		sym, _, err := r.ReadRune()
		if err == nil && sym != 0 {
			if int(sym) < 256 {
				mat.sigmaASCII[int(sym)] = x
			}
			mat.sigma[sym] = x
		}
	}

	_, err = io.ReadFull(r, buf[0:1])

	if err != nil {
		log.Print(err)
		return nil
	}

	if string("M") != string(buf[0:1]) {
		log.Println("Not a matok file")
		return nil
	}

	// Read based on length
	mat.array = make([]uint32, arraySize)

	dataArray, err := io.ReadAll(r)

	if err == io.EOF {
		log.Println(err)
		return nil
	}

	if len(dataArray) < arraySize*4 {
		log.Println("Not enough bytes read", len(dataArray), arraySize*4)
		return nil
	}

	for x := 0; x < arraySize; x++ {
		mat.array[x] = bo.Uint32(dataArray[x*4 : (x*4)+4])
	}

	return mat
}

// Transduce input to ouutput
func (mat *MatrixTokenizer) Transduce(r io.Reader, w io.Writer) bool {
	return mat.TransduceTokenWriter(r, NewTokenWriter(w, SIMPLE))
}

// TransduceTokenWriter transduces an input string against
// the matrix FSA. The rules are always greedy. If the
// automaton fails, it takes the last possible token ending
// branch.
func (mat *MatrixTokenizer) TransduceTokenWriter(r io.Reader, w *TokenWriter) bool {
	var a int
	var t0 uint32
	t := uint32(1) // Initial state
	var ok, rewindBuffer bool

	// Remember the last position of a possible tokenend,
	// in case the automaton fails.
	epsilonState := uint32(0)
	epsilonOffset := 0

	// Remember if the last transition was epsilon
	sentenceEnd := false

	// Remember if a text end was already set
	textEnd := false

	buffer := make([]rune, 1024)
	bufft := 0 // Buffer token offset
	buffc := 0 // Buffer current symbol
	buffi := 0 // Buffer length

	// The buffer is organized as follows:
	// [   t[....c..]..i]

	reader := bufio.NewReader(r)
	defer w.Flush()

	var char rune

	var err error
	eof := false
	eot := false
	newchar := true
	var nextChar []byte

PARSECHARM:
	for {

		if newchar {
			// Get from reader if buffer is empty
			if buffc >= buffi {
				if eof {
					break
				}
				char, _, err = reader.ReadRune()

				// No more runes to read
				if err != nil {
					if err == io.EOF {
						eof = true
						break
					}

					log.Fatalln(err)
					os.Exit(1)
					return false
				}

				buffer[buffi] = char
				buffi++

				// Check if the next character is a Joiner or a Variant Joiner,
				// which makes the preceeding char an emoji
				nextChar, err = reader.Peek(3)

				// check for zero width joiner aka []byte(string(rune(0x200d)))
				if err == nil && nextChar[0] == 226 && nextChar[1] == 128 && nextChar[2] == 141 {
					fmt.Println("Found zero width joiner")
				}

				// If yes, parse all joins in the buffer and increment buffi.
				// If it is a symbol, store length in symbol[buffc], otherwise
				// store 0
			}

			char = buffer[buffc]

			if DEBUG {
				log.Println("Current char", string(char), int(char), showBufferNew(buffer, bufft, buffc, buffi))
			}

			eot = false

			// TODO:
			//   Better not repeatedly check for a!
			//   Possibly keep a buffer with a.
			//   And this could be joined with the
			//   symbol buffer!
			if int(char) < 256 {
				eot = int(char) == EOT

				// mat.SigmaASCII[] is initialized with mat.identity
				a = mat.sigmaASCII[int(char)]
			} else if SYMBOLSUPPORT && char > 0x2190 {
				// Check symbol unicode blocks
				//
				// 0x2190 - 0x21FF - Arrows
				// 0x2200 - 0x22FF - Mathematical Operators
				// 0x2300 - 0x23FF - Miscellaneous Technical
				// 0x2400 - 0x243F - Control Pictures
				// ! 0x2440 - 0x245F - Optical Character Recognition
				// 0x2460 - 0x24FF - Enclosed Alphanumerics
				// 0x2500 - 0x257F - Box Drawing
				// 0x2580 - 0x259F - Block Elements
				// 0x25A0 - 0x25FF - Geometric Shapes
				// 0x2600 - 0x26FF - Miscellaneous Symbols
				// 0x2700 - 0x27BF - Dingbats
				// 0x27C0 - 0x27EF - Miscellaneous Mathematical Symbols-A
				// 0x27F0 - 0x27FF - Supplemental Arrows-A
				// ! 0x2800..0x28FF 	Braille Patterns 	256 	256 	Braille
				// 0x2900 - 0x297F - Supplemental Arrows-B
				// 0x2980 - 0x29FF - Miscellaneous Mathematical Symbols-B
				// 0x2A00 - 0x2AFF - Supplemental Mathematical Operators
				// 0x2B00 - 0x2BFF - Miscellaneous Symbols and Arrows
				// ---
				// 0x1F000 - 0x1F02F - Mahjong Tiles
				// 0x1F030 - 0x1F09F - Domino Tiles
				// 0x1F0A0 - 0x1F0FF - Playing Cards
				// ! 0x1F100 - 0x1F1FF - Enclosed Alphanumeric Supplement
				// ! 0x1F200 - 0x1F2FF - Enclosed Ideographic Supplement
				// 0x1F300 - 0x1F5FF - Miscellaneous Symbols and Pictographs
				// 0x1F600 - 0x1F64F - Emoticons
				// 0x1F650 - 0x1F67F - Ornamental Dingbats
				// 0x1F680 - 0x1F6FF - Transport and Map Symbols
				// 0x1F700 - 0x1F77F - Alchemical Symbols
				// 0x1F780 - 0x1F7FF - Geometric Shapes Extended
				// 0x1F800 - 0x1F8FF - Supplemental Arrows-C
				// 0x1F900 - 0x1F9FF - Supplemental Symbols and Pictographs
				// 0x1FA00 - 0x1FA6F - Chess Symbols
				// 0x1FA70 - 0x1FAFF - Symbols and Pictographs Extended-A
				// 0x1FB00 - 0x1FBFF - Symbols for Legacy Computing
				if char < 0x2BFF || (char > 0x1F000 && char < 0x1FBFF) && !((char > 0x2440 && char < 0x245f) || (char > 0x2800 && char < 0x28FF) || (char > 0x1F100 && char < 0x1F2FF)) {
					// TODO:
					// Symbol! Check if the next character is 0x200d,
					// which is a zero width joiner, that means
					// multiple symbols form a single symbol
					a, ok = mat.sigma[char]

					// Use identity symbol if character is not in sigma
					if !ok && mat.identity != -1 {
						a = mat.identity
					}

				} else {
					a, ok = mat.sigma[char]

					// Use identity symbol if character is not in sigma
					if !ok && mat.identity != -1 {
						a = mat.identity
					}

				}
			} else {
				a, ok = mat.sigma[char]

				// Use identity symbol if character is not in sigma
				if !ok && mat.identity != -1 {

					// TODO: Maybe use unknown?
					a = mat.identity
				}
			}

			t0 = t

			// Check for epsilon transitions and remember

			// TODO: Can t0 be negative here?
			if mat.array[(mat.epsilon-1)*mat.stateCount+int(t0)] != 0 {
				// Remember state for backtracking to last tokenend state

				// Maybe not necessary - and should be simpler!
				// Just Remove
				// t0 &= ^FIRSTBIT
				epsilonState = t0
				epsilonOffset = buffc

				if DEBUG {
					log.Println("epsilonOffset is set to", buffc)
				}
			}
		}

		// can happen when no identity is defined.
		// This shouldn't be tested in every loop
		if a == 0 {
			t = 0
		} else {
			// Checks a transition based on t0, a and buffo
			t = mat.array[(int(a)-1)*mat.stateCount+int(t0)]
		}

		if DEBUG {
			// Char is only relevant if set
			log.Println("Check", t0, "-", a, "(", string(char), ")", "->", t)
		}

		// Check if the transition is invalid according to the matrix
		if t == 0 {

			if DEBUG {
				log.Println("Match is not fine!")
			}

			if !ok && a == mat.identity {

				// Try again with unknown symbol, in case identity failed
				// Char is only relevant when set
				if DEBUG {
					log.Println("UNKNOWN symbol", string(char), "->", mat.unknown)
				}
				a = mat.unknown

			} else if a != mat.epsilon && epsilonState != 0 {

				// Try again with epsilon symbol, in case everything else failed
				t0 = epsilonState
				epsilonState = 0 // reset
				buffc = epsilonOffset
				a = mat.epsilon

				if DEBUG {
					log.Println("Get from epsilon stack and set buffo!", showBufferNew(buffer, bufft, buffc, buffi))
				}

			} else {

				if DEBUG {
					log.Println("Fail!")
				}

				// w.Fail(bufft)

				// The following procedure means the automaton fails to consume a certain character.
				// In the tokenization scenario, this means, the tokenizer will drop the old or current data as a
				// token and start blank at the root node of the automaton for the remaining data.
				// It may be beneficial to have something like a "drop()" event to capture these cases,
				// as they are likely the result of a bad automaton design.

				//			fmt.Println("Problem", len(buffer), buffc, bufft)

				if buffc-bufft <= 0 {
					buffc++
					if buffc == 0 {
						eof = true
						break
					}
				}
				// This will hopefully be branchless by the compiler

				if DEBUG {
					log.Println("-> Flush buffer: [", string(buffer[bufft:buffc]), "]", showBufferNew(buffer, bufft, buffc, buffi))
				}

				w.Token(bufft, buffer[:buffc])

				sentenceEnd = false
				textEnd = false

				if DEBUG {
					log.Println("-> Rewind buffer", bufft, buffc, buffi, epsilonOffset)
				}

				copy(buffer[0:], buffer[buffc:buffi])

				buffi -= buffc
				epsilonState = 0

				buffc = 0
				bufft = 0

				a = mat.epsilon

				// Restart from root state
				t = uint32(1)
				newchar = true
				// goto PARSECHARM
				continue
			}

			newchar = false
			eot = false
			continue
		}

		// Transition was successful
		rewindBuffer = false

		// Transition consumes no character
		if a == mat.epsilon {
			// Transition marks the end of a token - so flush the buffer
			if buffc-bufft > 0 {
				if DEBUG {
					log.Println("-> Flush buffer: [", string(buffer[bufft:buffc]), "]", showBufferNew(buffer, bufft, buffc, buffi))
				}
				w.Token(bufft, buffer[:buffc])
				rewindBuffer = true
				sentenceEnd = false
				textEnd = false
			} else {
				sentenceEnd = true
				w.SentenceEnd(buffc)
			}

			// Transition consumes a character
		} else {
			buffc++

			// Transition does not produce a character
			// Hopefully generated branchless code
			if buffc-bufft == 1 && (t&FIRSTBIT) != 0 {
				if DEBUG {
					log.Println("Nontoken forward", showBufferNew(buffer, bufft, buffc, buffi))
				}
				bufft++
				// rewindBuffer = true
			}
		}

		if eot {
			eot = false
			textEnd = true
			w.TextEnd(buffc)
			rewindBuffer = true
			if DEBUG {
				log.Println("END OF TEXT")
			}
		}

		// Rewind the buffer if necessary
		if rewindBuffer {

			if DEBUG {
				log.Println("-> Rewind buffer", bufft, buffc, buffi, epsilonOffset)
			}

			copy(buffer[0:], buffer[buffc:buffi])

			buffi -= buffc
			// epsilonOffset -= buffo
			epsilonOffset = 0
			epsilonState = 0

			buffc = 0
			bufft = 0

			if DEBUG {
				log.Println("Remaining:", showBufferNew(buffer, bufft, buffc, buffi))
			}
		}

		t &= ^FIRSTBIT

		newchar = true

		// TODO:
		//   Prevent endless epsilon loops!
	}

	// Input reader is not yet finished
	if !eof {
		if DEBUG {
			log.Println("Not at the end")
		}
		// This should never happen
		return false
	}

	if DEBUG {
		log.Println("Entering final check")
	}

	// Check epsilon transitions as long as possible
	t0 = t
	t = mat.array[(int(mat.epsilon)-1)*mat.stateCount+int(t0)]
	a = mat.epsilon
	newchar = false
	// t can't be < 0
	if t != 0 {
		// Remember state for backtracking to last tokenend state
		goto PARSECHARM

	} else if epsilonState != 0 {
		t0 = epsilonState
		epsilonState = 0 // reset
		buffc = epsilonOffset
		if DEBUG {
			log.Println("Get from epsilon stack and set buffo!", showBufferNew(buffer, bufft, buffc, buffi))
		}
		goto PARSECHARM
	}

	// something left in buffer
	if buffc-bufft > 0 {
		if DEBUG {
			log.Println("-> Flush buffer: [", string(buffer[bufft:buffc]), "]", showBufferNew(buffer, bufft, buffc, buffi))
		}
		w.Token(bufft, buffer[:buffc])
		sentenceEnd = false
		textEnd = false
	}

	// Add an additional sentence ending, if the file is over but no explicit
	// sentence split was reached. This may be controversial and therefore
	// optional via parameter.
	if !sentenceEnd {
		w.SentenceEnd(buffc)
		if DEBUG {
			log.Println("Sentence end")
		}
	}

	if !textEnd {
		w.TextEnd(buffc)
		if DEBUG {
			log.Println("Text end")
		}
	}

	return true
}

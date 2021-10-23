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
	MAMAGIC = "MATOK"
	EOT     = 4
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
	for num, sym := range auto.sigmaRev {
		if int(sym) < 256 {
			mat.sigmaASCII[int(sym)] = num
		}
		mat.sigma[sym] = num
		if num > auto.sigmaCount {
			panic("sigmaCount is smaller")
		}
		if num > max {
			max = num
		}
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
		if num > max {
			max = num
		}
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

func (mat *MatrixTokenizer) Transduce(r io.Reader, w io.Writer) bool {
	return mat.TransduceTokenWriter(r, NewTokenWriterSimple(w))
}

func (mat *MatrixTokenizer) TransduceTokenWriter(r io.Reader, w TokenWriterI) bool {
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
	buffo := 0 // Buffer offset
	buffi := 0 // Buffer length

	reader := bufio.NewReader(r)
	defer w.Flush()

	var char rune

	var err error
	eof := false
	eot := false
	newchar := true

PARSECHARM:
	for {

		if newchar {
			// Get from reader if buffer is empty
			if buffo >= buffi {
				if eof {
					break
				}
				char, _, err = reader.ReadRune()

				// No more runes to read
				if err != nil {
					eof = true
					break
				}
				buffer[buffi] = char
				buffi++
			}

			char = buffer[buffo]

			if DEBUG {
				fmt.Println("Current char", string(char), int(char), showBuffer(buffer, buffo, buffi))
			}

			eot = false

			// TODO:
			//   Better not repeatedly check for a!
			//   Possibly keep a buffer with a.
			if int(char) < 256 {
				if int(char) == EOT {
					eot = true
				}
				a = mat.sigmaASCII[int(char)]
			} else {
				a, ok = mat.sigma[char]
				if !ok {
					a = 0
				}
			}

			// Use identity symbol if character is not in sigma
			if a == 0 && mat.identity != -1 {
				a = mat.identity
			}

			t0 = t

			// Check for epsilon transitions and remember

			// TODO: Can t0 be negative here?
			if mat.array[(mat.epsilon-1)*mat.stateCount+int(t0)] != 0 {
				// Remember state for backtracking to last tokenend state

				// Maybe not necessary - and should be simpler!
				// Just Remove
				t0 &= ^FIRSTBIT
				epsilonState = t0
				epsilonOffset = buffo

				if DEBUG {
					fmt.Println("epsilonOffset is set to", buffo)
				}
			}
		}

		// Checks a transition based on t0, a and buffo
		t = mat.array[(int(a)-1)*mat.stateCount+int(t0)]

		if DEBUG {
			// Char is only relevant if set
			fmt.Println("Check", t0, "-", a, "(", string(char), ")", "->", t)
		}

		// Check if the transition is invalid according to the matrix
		if t == 0 {

			if DEBUG {
				fmt.Println("Match is not fine!")
			}

			if !ok && a == mat.identity {

				// Try again with unknown symbol, in case identity failed
				// Char is only relevant when set
				if DEBUG {
					fmt.Println("UNKNOWN symbol", string(char), "->", mat.unknown)
				}
				a = mat.unknown

			} else if a != mat.epsilon {

				// Try again with epsilon symbol, in case everything else failed
				t0 = epsilonState
				epsilonState = 0 // reset
				buffo = epsilonOffset
				a = mat.epsilon

				if DEBUG {
					fmt.Println("Get from epsilon stack and set buffo!", showBuffer(buffer, buffo, buffi))
				}

			} else {
				break
			}

			newchar = false
			eot = false
			continue
		}

		// Transition was successful
		rewindBuffer = false

		// Transition consumes a character
		if a != mat.epsilon {

			buffo++

			// Transition does not produce a character
			if buffo == 1 && (t&FIRSTBIT) != 0 {
				if DEBUG {
					fmt.Println("Nontoken forward", showBuffer(buffer, buffo, buffi))
				}
				rewindBuffer = true
			}

		} else {
			// Transition marks the end of a token - so flush the buffer
			if buffo > 0 {
				if DEBUG {
					fmt.Println("-> Flush buffer: [", string(buffer[:buffo]), "]", showBuffer(buffer, buffo, buffi))
				}
				w.Token(0, buffer[:buffo])
				rewindBuffer = true
				sentenceEnd = false
				textEnd = false
			} else {
				sentenceEnd = true
				w.SentenceEnd(0)
			}
			if DEBUG {
				fmt.Println("-> Newline")
			}
			// writer.WriteRune('\n')
		}

		// Rewind the buffer if necessary
		if rewindBuffer {

			if DEBUG {
				fmt.Println("-> Rewind buffer", buffo, buffi, epsilonOffset)
			}

			// TODO: Better as a ring buffer
			for x, i := range buffer[buffo:buffi] {
				buffer[x] = i
			}

			buffi -= buffo
			// epsilonOffset -= buffo
			epsilonOffset = 0
			epsilonState = 0

			buffo = 0
			if DEBUG {
				fmt.Println("Remaining:", showBuffer(buffer, buffo, buffi))
			}

			if eot {
				eot = false
				textEnd = true
				w.TextEnd(0)
				if DEBUG {
					fmt.Println("END OF TEXT")
				}
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
			fmt.Println("Not at the end")
		}
		return false
	}

	if DEBUG {
		fmt.Println("Entering final check")
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
		buffo = epsilonOffset
		if DEBUG {
			fmt.Println("Get from epsilon stack and set buffo!", showBuffer(buffer, buffo, buffi))
		}
		goto PARSECHARM
	}

	// Add an additional sentence ending, if the file is over but no explicit
	// sentence split was reached. This may be controversial and therefore
	// optional via parameter.
	if !sentenceEnd {
		w.SentenceEnd(0)
		if DEBUG {
			fmt.Println("Sentence end")
		}
	}

	if !textEnd {
		w.TextEnd(0)

		if DEBUG {
			fmt.Println("Text end")
		}
	}

	return true
}

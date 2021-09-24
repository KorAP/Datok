package datok

import (
	"bufio"
	"fmt"
	"io"
)

type MatrixTokenizer struct {
	sigma      map[rune]int
	sigmaASCII [256]int
	array      []int
	stateCount int

	// Special symbols in sigma
	epsilon  int
	unknown  int
	identity int
	final    int
	tokenend int
}

// ToMatrix turns the intermediate tokenizer into a
// matrix representation.
func (auto *Automaton) ToMatrix() *MatrixTokenizer {

	mat := &MatrixTokenizer{
		sigma:      make(map[rune]int),
		final:      auto.final,
		unknown:    auto.unknown,
		identity:   auto.identity,
		epsilon:    auto.epsilon,
		tokenend:   auto.tokenend,
		stateCount: auto.stateCount,
	}

	mat.array = make([]int, (auto.stateCount+1)*(auto.sigmaCount+1))

	for num, sym := range auto.sigmaRev {
		if int(sym) < 256 {
			mat.sigmaASCII[int(sym)] = num
		}
		mat.sigma[sym] = num
		if num > auto.sigmaCount {
			panic("sigmaCount is smaller")
		}
	}
	remember := make([]bool, auto.stateCount+2)

	// Store all transitions in matrix
	var toMatrix func([]int, int)

	toMatrix = func(matrix []int, start int) {
		if start > auto.stateCount {
			panic("stateCount is smaller")
		}
		if remember[start] {
			return
		}
		remember[start] = true
		for alpha, t := range auto.transitions[start] {
			matrix[(alpha-1)*auto.stateCount+start] = t.end

			// Mark nontoken transitions
			if t.nontoken {
				matrix[(alpha-1)*auto.stateCount+start] *= -1
			}

			toMatrix(matrix, t.end)
		}
	}

	toMatrix(mat.array, 1)

	return mat
}

func (mat *MatrixTokenizer) Transduce(r io.Reader, w io.Writer) bool {
	var a int
	var t0 int
	t := int(1) // Initial state
	var ok, rewindBuffer bool

	// Remember the last position of a possible tokenend,
	// in case the automaton fails.
	epsilonState := int(0)
	epsilonOffset := 0

	// Remember if the last transition was epsilon
	sentenceEnd := false

	buffer := make([]rune, 1024)
	buffo := 0 // Buffer offset
	buffi := 0 // Buffer length

	reader := bufio.NewReader(r)
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	var char rune

	var err error
	eof := false
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
				fmt.Println("Current char", string(char), showBuffer(buffer, buffo, buffi))
			}

			// TODO:
			//   Better not repeatedly check for a!
			//   Possibly keep a buffer with a.
			if int(char) < 256 {
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

			if mat.array[(mat.epsilon-1)*mat.stateCount+t0] != 0 {
				// Remember state for backtracking to last tokenend state
				epsilonState = t0
				epsilonOffset = buffo
			}
		}

		// Checks a transition based on t0, a and buffo
		t = mat.array[(int(a)-1)*mat.stateCount+int(t0)]
		// t = mat.array[t0].getBase() + uint32(a)
		// ta := dat.array[t]

		if DEBUG {
			// Char is only relevant if set
			fmt.Println("Check", t0, "-", a, "(", string(char), ")", "->", t)
			/*
				if false {
					fmt.Println(dat.outgoing(t0))
				}
			*/
		}

		// Check if the transition is invalid according to the double array
		// if t > dat.array[1].getCheck() || ta.getCheck() != t0 {
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
			continue
		}

		// Transition was successful
		rewindBuffer = false

		// Transition consumes a character
		if a != mat.epsilon {

			buffo++

			// Transition does not produce a character
			// if buffo == 1 && ta.isNonToken() {
			if buffo == 1 && t < 0 {
				if DEBUG {
					fmt.Println("Nontoken forward", showBuffer(buffer, buffo, buffi))
				}
				rewindBuffer = true
			}

		} else {
			// Transition marks the end of a token - so flush the buffer

			if buffi > 0 {
				if DEBUG {
					fmt.Println("-> Flush buffer: [", string(buffer[:buffo]), "]", showBuffer(buffer, buffo, buffi))
				}
				writer.WriteString(string(buffer[:buffo]))
				rewindBuffer = true
				sentenceEnd = false
			} else {
				sentenceEnd = true
			}
			if DEBUG {
				fmt.Println("-> Newline")
			}
			writer.WriteRune('\n')
		}

		// Rewind the buffer if necessary
		if rewindBuffer {

			// TODO: Better as a ring buffer
			for x, i := range buffer[buffo:buffi] {
				buffer[x] = i
			}

			buffi -= buffo
			epsilonOffset -= buffo
			buffo = 0
			if DEBUG {
				fmt.Println("Remaining:", showBuffer(buffer, buffo, buffi))
			}
		}

		// Move to representative state
		/*
			if ta.isSeparate() {
				t = ta.getBase()
				ta = dat.array[t]

				if DEBUG {
					fmt.Println("Representative pointing to", t)
				}
			}
		*/

		// Ignore nontoken mark
		if t < 0 {
			t *= -1
		}

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
	/*
		// Automaton is in a final state, so flush the buffer and return
		x := dat.array[t].getBase() + uint32(dat.final)

		if x < dat.array[1].getCheck() && dat.array[x].getCheck() == t {

			if buffi > 0 {
				if DEBUG {
					fmt.Println("-> Flush buffer: [", string(buffer[:buffi]), "]")
				}
				writer.WriteString(string(buffer[:buffi]))

				if dat.array[t].isTokenEnd() {
					writer.WriteRune('\n')
					if DEBUG {
						fmt.Println("-> Newline")
					}
				}
			}

			// Add an additional sentence ending, if the file is over but no explicit
			// sentence split was reached. This may be controversial and therefore
			// optional via parameter.
			if !dat.array[t0].isTokenEnd() {
				writer.WriteRune('\n')
				if DEBUG {
					fmt.Println("-> Newline")
				}
			}

			// TODO:
			//   There may be a new line at the end, from an epsilon,
			//   so we may need to go on!
			return true
		}
	*/

	// Check epsilon transitions until a final state is reached
	t0 = t
	// t = dat.array[t0].getBase() + uint32(dat.epsilon)
	t = mat.array[(int(mat.epsilon)-1)*mat.stateCount+int(t0)]
	a = mat.epsilon
	newchar = false
	// if dat.array[t].getCheck() == t0 {
	// t can't be < 0
	if t > 0 {
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
		writer.WriteRune('\n')
		if DEBUG {
			fmt.Println("-> Newline")
		}
	}

	return true
}

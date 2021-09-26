package datok

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	PROPS  = 1
	SIGMA  = 2
	STATES = 3
	NONE   = 4
)

type edge struct {
	inSym    int
	outSym   int
	end      int
	nontoken bool
	tokenend bool
}

type Tokenizer interface {
	Transduce(r io.Reader, w io.Writer) bool
	Type() string
}

// Automaton is the intermediate representation
// of the tokenizer.
type Automaton struct {
	sigmaRev    map[int]rune
	sigmaMCS    map[int]string
	arcCount    int
	sigmaCount  int
	stateCount  int
	transitions []map[int]*edge

	// Special symbols in sigma
	epsilon  int
	unknown  int
	identity int
	final    int
	tokenend int
}

// ParseFoma reads the FST from a foma file
// and creates an internal representation,
// in case it follows the tokenizer's convention.
func LoadFomaFile(file string) *Automaton {
	f, err := os.Open(file)
	if err != nil {
		log.Print(err)
		return nil
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		log.Print(err)
		return nil
	}
	defer gz.Close()

	return ParseFoma(gz)
}

// ParseFoma reads the FST from a foma file reader
// and creates an internal representation,
// in case it follows the tokenizer's convention.
func ParseFoma(ior io.Reader) *Automaton {
	r := bufio.NewReader(ior)

	auto := &Automaton{
		sigmaRev: make(map[int]rune),
		sigmaMCS: make(map[int]string),
		epsilon:  -1,
		unknown:  -1,
		identity: -1,
		final:    -1,
		tokenend: -1,
	}

	var state, inSym, outSym, end, final int

	mode := 0
	var elem []string
	var elemint [5]int

	// Iterate over all lines of the file.
	// This is mainly based on foma2js,
	// licensed under the Apache License, version 2,
	// and written by Mans Hulden.
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Print(err)
			return nil
		}

		// Read parser mode for the following lines
		if strings.HasPrefix(line, "##") {
			if strings.HasPrefix(line, "##props##") {
				mode = PROPS

			} else if strings.HasPrefix(line, "##states##") {
				mode = STATES

				// Adds a final transition symbol to sigma
				// written as '#' in Mizobuchi et al (2000)
				auto.sigmaCount++
				auto.final = auto.sigmaCount

			} else if strings.HasPrefix(line, "##sigma##") {

				mode = SIGMA

			} else if strings.HasPrefix(line, "##end##") {

				mode = NONE

			} else if !strings.HasPrefix(line, "##foma-net") {
				log.Print("Unknown input line")
				break
			}
			continue
		}

		// Based on the current parser mode, interpret the lines
		switch mode {
		case PROPS:
			{
				elem = strings.Split(line, " ")
				/*
					fmt.Println("arity:            " + elem[0])
					fmt.Println("arccount:         " + elem[1])
					fmt.Println("statecount:       " + elem[2])
					fmt.Println("linecount:        " + elem[3])
					fmt.Println("finalcount:       " + elem[4])
					fmt.Println("pathcount:        " + elem[5])
					fmt.Println("is_deterministic: " + elem[6])
					fmt.Println("is_pruned:        " + elem[7])
					fmt.Println("is_minimized:     " + elem[8])
					fmt.Println("is_epsilon_free:  " + elem[9])
					fmt.Println("is_loop_free:     " + elem[10])
					fmt.Println("extras:           " + elem[11])
					fmt.Println("name:             " + elem[12])
				*/
				if elem[6] != "1" {
					log.Print("The FST needs to be deterministic")
					return nil
				}

				if elem[9] != "1" {
					log.Print("The FST needs to be epsilon free")
					return nil
				}

				elemint[0], err = strconv.Atoi(elem[1])
				if err != nil {
					log.Print("Can't read arccount")
					return nil
				}
				auto.arcCount = elemint[0]

				elemint[0], err = strconv.Atoi(elem[2])
				if err != nil {
					log.Print("Can't read statecount")
					return nil
				}

				// States start at 1 in Mizobuchi et al (2000),
				// as the state 0 is associated with a fail.
				// Initialize states and transitions
				auto.stateCount = elemint[0]
				auto.transitions = make([]map[int]*edge, elemint[0]+1)
				continue
			}
		case STATES:
			{
				elem = strings.Split(line[0:len(line)-1], " ")
				if elem[0] == "-1" {
					if DEBUG {
						fmt.Println("Skip", elem)
					}
					continue
				}
				elemint[0], err = strconv.Atoi(elem[0])
				if err != nil {
					fmt.Println("Unable to translate", elem[0])
					break
				}

				if len(elem) > 1 {
					elemint[1], err = strconv.Atoi(elem[1])
					if err != nil {
						fmt.Println("Unable to translate", elem[1])
						break
					}
					if len(elem) > 2 {
						elemint[2], err = strconv.Atoi(elem[2])
						if err != nil {
							fmt.Println("Unable to translate", elem[2])
							break
						}
						if len(elem) > 3 {
							elemint[3], err = strconv.Atoi(elem[3])
							if err != nil {
								fmt.Println("Unable to translate", elem[3])
								break
							}
							if len(elem) > 4 {
								elemint[4], err = strconv.Atoi(elem[4])
								if err != nil {
									fmt.Println("Unable to translate", elem[4])
									break
								}
							}
						}
					}
				}

				switch len(elem) {
				case 5:
					{
						state = elemint[0]
						inSym = elemint[1]
						outSym = elemint[2]
						end = elemint[3]
						final = elemint[4]
					}
				case 4:
					{
						if elemint[1] == -1 {
							state = elemint[0]
							final = elemint[3]

							// Final state that has no outgoing edges
							if final == 1 {

								// Initialize outgoing states
								if auto.transitions[state+1] == nil {
									auto.transitions[state+1] = make(map[int]*edge)
								}

								// TODO:
								//   Maybe this is less relevant for tokenizers
								auto.transitions[state+1][auto.final] = &edge{}
							}
							continue
						} else {
							state = elemint[0]
							inSym = elemint[1]
							end = elemint[2]
							final = elemint[3]
							outSym = inSym
						}
					}
				case 3:
					{
						inSym = elemint[0]
						outSym = elemint[1]
						end = elemint[2]
					}
				case 2:
					{
						inSym = elemint[0]
						end = elemint[1]
						outSym = inSym
					}
				}

				nontoken := false
				tokenend := false

				// While the states in foma start with 0, the states in the
				// Mizobuchi FSA start with one - so we increase every state by 1.
				// We also increase sigma by 1, so there are no 0 transitions.
				inSym++
				outSym++

				// Only a limited list of transitions are allowed
				if inSym != outSym {
					if outSym == auto.tokenend && inSym == auto.epsilon {
						tokenend = true
					} else if outSym == auto.epsilon {
						nontoken = true
					} else {
						log.Println(
							"Unsupported transition: " +
								strconv.Itoa(state) +
								" -> " + strconv.Itoa(end) +
								" (" +
								strconv.Itoa(inSym) +
								":" +
								strconv.Itoa(outSym) +
								") (" +
								string(auto.sigmaRev[inSym]) +
								":" +
								string(auto.sigmaRev[outSym]) +
								")")
						return nil
					}
				} else if inSym == auto.tokenend {
					// Ignore tokenend accepting arcs
					continue
				} else if inSym == auto.epsilon {
					log.Println("General epsilon transitions are not supported")
					return nil
				} else if auto.sigmaMCS[inSym] != "" {
					// log.Fatalln("Non supported character", tok.sigmaMCS[inSym])
					// Ignore MCS transitions
					continue
				}

				// Create an edge based on the collected information
				targetObj := &edge{
					inSym:    inSym,
					outSym:   outSym,
					end:      end + 1,
					tokenend: tokenend,
					nontoken: nontoken,
				}

				// Initialize outgoing states
				if auto.transitions[state+1] == nil {
					auto.transitions[state+1] = make(map[int]*edge)
				}

				// Ignore transitions with invalid symbols
				if inSym >= 0 {
					auto.transitions[state+1][inSym] = targetObj
				}

				// Add final transition
				if final == 1 {
					// TODO:
					//   Maybe this is less relevant for tokenizers
					auto.transitions[state+1][auto.final] = &edge{}
				}

				if DEBUG {
					fmt.Println("Add",
						state+1, "->", end+1,
						"(",
						inSym,
						":",
						outSym,
						") (",
						string(auto.sigmaRev[inSym]),
						":",
						string(auto.sigmaRev[outSym]),
						")",
						";",
						"TE:", tokenend,
						"NT:", nontoken,
						"FIN:", final)
				}

				continue
			}
		case SIGMA:
			{
				elem = strings.SplitN(line[0:len(line)-1], " ", 2)

				// Turn string into sigma id
				number, err := strconv.Atoi(elem[0])

				// ID needs to be > 1
				number++

				if err != nil {
					log.Println(err)
					return nil
				}

				auto.sigmaCount = number

				var symbol rune

				// Read rune
				if utf8.RuneCountInString(elem[1]) == 1 {
					symbol = []rune(elem[1])[0]

				} else if utf8.RuneCountInString(elem[1]) > 1 {

					// Probably a MCS
					switch elem[1] {
					case "@_EPSILON_SYMBOL_@":
						{
							auto.epsilon = number
						}
					case "@_UNKNOWN_SYMBOL_@":
						{
							auto.unknown = number
						}

					case "@_IDENTITY_SYMBOL_@":
						{
							auto.identity = number
						}

					case "@_TOKEN_SYMBOL_@":
						{
							auto.tokenend = number
						}
					default:
						{
							// MCS not supported
							auto.sigmaMCS[number] = line
						}
					}
					continue

				} else { // Probably a new line symbol
					line, err = r.ReadString('\n')
					if err != nil {
						log.Println(err)
						return nil
					}
					if len(line) != 1 {
						// MCS not supported
						auto.sigmaMCS[number] = line
						continue
					}
					symbol = rune('\n')
				}

				auto.sigmaRev[number] = symbol
			}
		}
	}
	auto.sigmaMCS = nil
	return auto
}

func LoadTokenizerFile(file string) Tokenizer {
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

	r := bufio.NewReader(gz)

	mstr, err := r.Peek(len(DAMAGIC))

	if err != nil {
		log.Println(err)
		return nil
	}

	if string(mstr) == MAMAGIC {
		return ParseMatrix(r)
	} else if string(mstr) == DAMAGIC {
		return ParseDatok(r)
	}

	log.Println("Neither a matrix nor a datok file")
	return nil
}

// Set alphabet A to the list of all symbols
// outgoing from s
func (auto *Automaton) getSet(s int, A *[]int) {
	for a := range auto.transitions[s] {
		*A = append(*A, a)
	}

	// Not required, but simplifies bug hunting
	// sort.Ints(*A)
}

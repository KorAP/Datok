package datokenizer

/**
 * The file reader is basically a port of foma2js,
 * licensed under the Apache License, version 2,
 * and written by Mans Hulden.
 */

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	PROPS   = 1
	SIGMA   = 2
	STATES  = 3
	NONE    = 4
	NEWLINE = '\u000a'
)

// Special symbols in sigma
var EPSILON = -1
var UNKNOWN = -1
var IDENTITY = -1
var FINAL = -1

type mapping struct {
	source int
	target int
}

type edge struct {
	in     int
	out    int
	target int
}

type Tokenizer struct {
	sigma       map[rune]int
	sigma_rev   map[int]rune
	arccount    int
	statecount  int
	sigmacount  int
	maxsize     int
	array       []int
	transitions []map[int]*edge
}

func parse_file(file string) *Tokenizer {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	defer gz.Close()

	return parse(gz)
}

func parse(ior io.Reader) *Tokenizer {
	r := bufio.NewReader(ior)

	tok := &Tokenizer{
		sigma:     make(map[rune]int),
		sigma_rev: make(map[int]rune),
	}

	final := false

	var arrstate, arrin, arrout, arrtarget, arrfinal int

	mode := 0
	var elem []string
	var elemint [5]int

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if strings.HasPrefix(line, "##foma-net") {
			continue
		}
		if strings.HasPrefix(line, "##props##") {
			mode = PROPS
			continue
		}
		if strings.HasPrefix(line, "##states##") {
			mode = STATES

			// Adds a final transition symbol to sigma
			// written as '#' in Mizobuchi et al (2000)
			tok.sigmacount++
			FINAL = tok.sigmacount
			continue
		}
		if strings.HasPrefix(line, "##sigma##") {
			mode = SIGMA
			continue
		}
		if strings.HasPrefix(line, "##end##") {
			mode = NONE
			continue
		}

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
					panic("The FST needs to be deterministic")
				}
				if elem[9] != "1" {
					panic("The FST needs to be epsilon free")
				}

				elemint[0], err = strconv.Atoi(elem[1])
				if err != nil {
					panic("Can't read arccount")
				}
				tok.arccount = elemint[0]

				// States start at 1 in Mizobuchi et al (2000),
				// as the state 0 is associated with a fail.
				// Initialize states and transitions
				elemint[0], err = strconv.Atoi(elem[2])
				if err != nil {
					panic("Can't read statecount")
				}
				tok.statecount = elemint[0]
				tok.transitions = make([]map[int]*edge, elemint[0]+1)
				continue
			}
		case STATES:
			{
				elem = strings.Split(line[0:len(line)-1], " ")
				if elem[0] == "-1" {
					continue
				}
				elemint[0], err = strconv.Atoi(elem[0])
				if err != nil {
					break
				}

				if len(elem) > 1 {
					elemint[1], err = strconv.Atoi(elem[1])
					if err != nil {
						break
					}
					if len(elem) > 2 {
						elemint[2], err = strconv.Atoi(elem[2])
						if err != nil {
							break
						}
						if len(elem) > 3 {
							elemint[3], err = strconv.Atoi(elem[3])
							if err != nil {
								break
							}
							if len(elem) > 4 {
								elemint[4], err = strconv.Atoi(elem[4])
								if err != nil {
									break
								}
							}
						}
					}
				}

				switch len(elem) {
				case 5:
					{
						arrstate = elemint[0]
						arrin = elemint[1]
						arrout = elemint[2]
						arrtarget = elemint[3]
						arrfinal = elemint[4]
					}
				case 4:
					{
						if elemint[1] == -1 {
							arrstate = elemint[0]
							arrfinal = elemint[3]
						} else {
							arrstate = elemint[0]
							arrin = elemint[1]
							arrtarget = elemint[2]
							arrfinal = elemint[3]
							arrout = arrin
						}
					}
				case 3:
					{
						arrin = elemint[0]
						arrout = elemint[1]
						arrtarget = elemint[2]
					}
				case 2:
					{
						arrin = elemint[0]
						arrtarget = elemint[1]
						arrout = arrin
					}
				}

				// This collects all edges until arrstate changes
				if arrfinal == 1 {
					final = true
				} else {
					final = false
				}

				// While the states in foma start with 0, the states in the
				// Mizobuchi FSA start with one - so we increase every state by 1.

				/*
					if arrin != arrout && arrin != EPSILON && tok.sigma_rev[arrin] != '\n' {
						panic("Problem: " + strconv.Itoa(arrstate) + " -> " + strconv.Itoa(arrtarget) + " (" + strconv.Itoa(arrin) + ":" + strconv.Itoa(arrout) + ") ")
					}
				*/
				if arrin != arrout {
					if arrin == EPSILON && tok.sigma_rev[arrout] == NEWLINE {
					} else if arrin != EPSILON && arrout == EPSILON {
					} else {
						panic(
							"Problem: " +
								strconv.Itoa(arrstate) +
								" -> " + strconv.Itoa(arrtarget) +
								" (" +
								strconv.Itoa(arrin) +
								":" +
								strconv.Itoa(arrout) +
								") (" +
								string(tok.sigma_rev[arrin]) +
								":" +
								string(tok.sigma_rev[arrout]) +
								")")
					}
				}

				// TODO:
				//   if arrin == EPSILON && arrout == TOKENEND, mark state as newline
				//   if the next transition is the same, remove TOKENEND and add SENTENCEEND
				//   This requires to remove the transition alltogether and marks the state instead.

				// TODO:
				//   if arrout == EPSILON, mark the transition as NOTOKEN

				targetObj := &edge{
					in:     arrin,
					out:    arrout,
					target: arrtarget + 1,
				}

				// Initialize outgoing state
				if tok.transitions[arrstate+1] == nil {
					tok.transitions[arrstate+1] = make(map[int]*edge)
				}

				if arrin >= 0 {
					tok.transitions[arrstate+1][arrin] = targetObj
				}

				if final {
					tok.transitions[arrstate+1][FINAL] = &edge{}
				}

				fmt.Println("Add",
					arrstate+1, "->", arrtarget+1,
					"(",
					arrin,
					":",
					arrout,
					") (",
					string(tok.sigma_rev[arrin]),
					":",
					string(tok.sigma_rev[arrout]),
					")")

				continue
			}
		case SIGMA:
			{
				elem = strings.SplitN(line[0:len(line)-1], " ", 2)

				// Turn string into sigma id
				number, err := strconv.Atoi(elem[0])

				if err != nil {
					panic(err)
				}

				tok.sigmacount = number

				var symbol rune

				// Read rune
				if utf8.RuneCountInString(elem[1]) == 1 {
					symbol = []rune(elem[1])[0]

					// Probably a MCS
				} else if utf8.RuneCountInString(elem[1]) > 1 {
					switch elem[1] {
					case "@_EPSILON_SYMBOL_@":
						{
							EPSILON = number
							continue
						}
					case "@_UNKNOWN_SYMBOL_@":
						{
							UNKNOWN = number
							continue
						}

					case "@_IDENTITY_SYMBOL_@":
						{
							IDENTITY = number
							continue
						}
					default:
						panic("MCS not supported: " + line)
					}

					// Probably a new line symbol
				} else {
					line, err = r.ReadString('\n')
					if err != nil {
						panic(err)
					}
					if len(line) != 1 {
						panic("MCS not supported:" + line)
					}
					symbol = rune('\n')
				}

				tok.sigma[symbol] = number
				tok.sigma_rev[number] = symbol
			}
		}
	}

	return tok
}

// Implementation of Mizobuchi et al (2000), p.128
func (tok *Tokenizer) buildDA() *Tokenizer {

	mark := 0
	size := 0

	// Create a mapping from s to t
	table := make([]*mapping, tok.arccount+1)

	table[size] = &mapping{source: 1, target: 1}
	size++

	A := make([]int, 0, 256)

	for mark < size {
		s := table[mark].source // This is a state in Ms
		t := table[mark].target // This is a state in Mt
		mark++
		//		fmt.Println("Increase mark", mark)
		// St := append(St, t)
		A = A[:0]
		tok.get_set(s, &A)

		// fmt.Println("Outgoing arcs from t", t, A)

		// tok.array[t].base = tok.x_check(A)
		tok.set_base(t, tok.x_check(A))

		for _, a := range A {

			if a != FINAL {
				s1 := tok.transitions[s][a].target // g(s, a)

				// fmt.Println("Found", s, "to", s1, "via", a)

				t1 := tok.get_base(t) + a
				tok.set_check(t1, t)

				r := in_table(s1, table, size)
				if r == 0 {
					// fmt.Println("Increase size", t1)
					table[size] = &mapping{source: s1, target: t1}
					size++
				} else {
					//fmt.Println("Rep is there", t1, r)
					tok.set_base(t1, -1*r)
					// tok.array[t1].base = -1 * r
				}
			} else {
				fmt.Println("I set a final")
				// t1 := tok.array[t].base + FINAL
				t1 := tok.get_base(t) + FINAL
				// tok.array[t1].check = t
				tok.set_check(t1, t)
			}
		}
	}

	// Following Mizobuchi et al (2000) the size of the
	// FSA should be stored in check(1).
	tok.set_check(1, tok.maxsize+1)
	tok.array = tok.array[:tok.maxsize+1]
	return tok
}

func (tok *Tokenizer) resize(l int) {
	if len(tok.array) <= l {
		tok.array = append(tok.array, make([]int, l)...)
	}
}

func (tok *Tokenizer) set_base(p int, v int) {
	l := p*2 + 1
	tok.resize(l)
	if tok.maxsize < l {
		tok.maxsize = l
	}
	tok.array[p*2] = v
}

func (tok *Tokenizer) get_base(p int) int {
	if p*2 >= len(tok.array) {
		return 0
	}
	return tok.array[p*2]
}

func (tok *Tokenizer) set_check(p int, v int) {
	l := p*2 + 1
	tok.resize(l)
	if tok.maxsize < l {
		tok.maxsize = l
	}
	tok.array[(p*2)+1] = v
}

func (tok *Tokenizer) get_check(p int) int {
	if (p*2)+1 >= len(tok.array) {
		return 0
	}
	return tok.array[(p*2)+1]
}

// Check the table if a mapping of s
// exists and return this as a representative
func in_table(s int, table []*mapping, size int) int {
	for x := 0; x < size; x++ {
		if table[x].source == s {
			return table[x].target
		}
	}
	return 0
}

// Set alphabet A to the list of all symbols
// outgoing from s
func (tok *Tokenizer) get_set(s int, A *[]int) {
	for a := range tok.transitions[s] {
		*A = append(*A, a)
	}
}

// Based on Mizobuchi et al (2000), p. 124
// This iterates for every state through the complete double array
// structure until it finds a gap that fits all outgoing transitions
// of the state. This is extremely slow, but is only necessary in the
// construction phase of the tokenizer.
func (tok *Tokenizer) x_check(symbols []int) int {
	// see https://github.com/bramstein/datrie/blob/master/lib/trie.js
	base := 1

	// 	fmt.Println("Resize", len(tok.linarray), "<", ((base + FINAL + 1) * 2))

OVERLAP:
	tok.resize((base + FINAL) * 2)
	for _, a := range symbols {
		// if tok.array[base+a].check != 0 {
		if tok.get_check(base+a) != 0 {
			base++
			goto OVERLAP
		}
	}
	//	fmt.Println("Found a nice place at", base, "for", len(symbols))
	return base
}

// Based on Mizobuchi et al (2000), p. 129
// Added support for IDENTITY, UNKNOWN and EPSILON
func (tok *Tokenizer) match(input string) bool {
	t := 1 // Start position
	chars := []rune(input)
	i := 0
	var a int
	var tu int
	var ok bool

	//	fmt.Println("Length of string is", len(chars))
	for i < len(chars) {
		a, ok = tok.sigma[chars[i]]

		// Support identity symbol if char not in sigma
		if !ok && IDENTITY != -1 {
			fmt.Println("IDENTITY symbol", string(chars[i]), "->", IDENTITY)
			a = IDENTITY
		} else {
			fmt.Println("Sigma transition is okay for [", string(chars[i]), "]")
		}
		tu = t
	CHECK:
		t = tok.get_base(tu) + a
		if t > tok.get_check(1) || tok.get_check(t) != tu {
			fmt.Println("Match is not fine!", t, "and", tok.get_check(t), "vs", tu)

			// Try again with unknown symbol, in case identity failed
			if !ok {
				if a == IDENTITY {
					fmt.Println("UNKNOWN symbol", string(chars[i]), "->", UNKNOWN)
					a = UNKNOWN
					goto CHECK
				} else if a == UNKNOWN {
					fmt.Println("aEPSILON symbol", string(chars[i]), "->", EPSILON)
					a = EPSILON
					// In the worst case, this checks epsilon twice at the same state -
					// here and at the end
					goto CHECK
				}
			} else if a != EPSILON {
				fmt.Println("bEPSILON symbol", string(chars[i]), "->", EPSILON)
				a = EPSILON
				// In the worst case, this checks epsilon twice at the same state -
				// here and at the end
				goto CHECK
			}
			break
		} else if tok.get_base(t) < 0 {
			// Move to representative state
			t = -1 * tok.get_base(t)
		}
		if a != EPSILON {
			i++
		}
	}

	if i == len(chars) {
		fmt.Println("At the end")
	} else {
		fmt.Println("Not at the end")
		return false
	}

	// fmt.Println("Hmm...", tok.get_check(tok.get_base(t)+FINAL), "-", t)

FINALCHECK:
	if tok.get_check(tok.get_base(t)+FINAL) == t {
		return true
	}

	tu = t
	a = EPSILON

	t = tok.get_base(tu) + a
	if t > tok.get_check(1) || tok.get_check(t) != tu {
		fmt.Println("xMatch is not fine!", t, "and", tok.get_check(t), "vs", tu)
		return false
	} else if tok.get_base(t) < 0 {
		// Move to representative state
		t = -1 * tok.get_base(t)
		goto FINALCHECK
	}
	goto FINALCHECK
}

// In the final realization, the states can only have 30 bits:
// base[1] -> is final
// base[2] -> is_separate
// check[1] -> translates to epsilon
// check[2] -> appends newine (Maybe)
// If check[1] && check[2] is set, this translates to a sentence split (Maybe)

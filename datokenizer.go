package datokenizer

/**
 * The file reader is basically a port of foma2js,
 * licensed under the Apache License, version 2,
 * and written by Mans Hulden.
 */

// The maximum number of states is 1.073.741.823 (30bit),
// with a loadfactor of ~70, this means roughly 70 million
// states in the FSA, which is sufficient for the current
// job.

// TODO:
// - replace maxSize with the check value
// - Strip first state and make everything start with 0!
// - Add checksum to serialization.

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
)

const (
	PROPS            = 1
	SIGMA            = 2
	STATES           = 3
	NONE             = 4
	NEWLINE          = '\u000a'
	DEBUG            = false
	MAGIC            = "DATOK"
	VERSION          = uint16(1)
	firstBit  uint32 = 1 << 31
	secondBit uint32 = 1 << 30
	restBit   uint32 = ^uint32(0) &^ (firstBit | secondBit)
)

var bo binary.ByteOrder = binary.LittleEndian

type mapping struct {
	source int
	target uint32
}

type edge struct {
	inSym    int
	outSym   int
	end      int
	nontoken bool
	tokenend bool
}

type Tokenizer struct {
	// sigma       map[rune]int
	sigmaRev    map[int]rune
	arcCount    int
	stateCount  int
	sigmaCount  int
	transitions []map[int]*edge

	// Special symbols in sigma
	epsilon  int
	unknown  int
	identity int
	final    int
}

type DaTokenizer struct {
	// sigmaRev map[int]rune
	sigma      map[rune]int
	maxSize    int
	loadFactor float64
	array      []uint32
	// lastFilledBase uint32

	// Special symbols in sigma
	epsilon  int
	unknown  int
	identity int
	final    int
}

func LoadFomaFile(file string) *Tokenizer {
	f, err := os.Open(file)
	if err != nil {
		log.Error().Err(err)
		os.Exit(0)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		log.Error().Err(err)
		os.Exit(0)
	}
	defer gz.Close()

	return ParseFoma(gz)
}

func ParseFoma(ior io.Reader) *Tokenizer {
	r := bufio.NewReader(ior)

	tok := &Tokenizer{
		sigmaRev: make(map[int]rune),
		epsilon:  -1,
		unknown:  -1,
		identity: -1,
		final:    -1,
	}

	checkmap := make(map[string]bool)

	var state, inSym, outSym, end, final int

	mode := 0
	var elem []string
	var elemint [5]int

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Error().Err(err)
			os.Exit(0)
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
			tok.sigmaCount++
			tok.final = tok.sigmaCount
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
					log.Error().Msg("The FST needs to be deterministic")
					os.Exit(1)
				}
				if elem[9] != "1" {
					log.Error().Msg("The FST needs to be epsilon free")
					os.Exit(1)
				}

				elemint[0], err = strconv.Atoi(elem[1])
				if err != nil {
					log.Error().Msg("Can't read arccount")
					os.Exit(1)
				}
				tok.arcCount = elemint[0]

				// States start at 1 in Mizobuchi et al (2000),
				// as the state 0 is associated with a fail.
				// Initialize states and transitions
				elemint[0], err = strconv.Atoi(elem[2])
				if err != nil {
					log.Error().Msg("Can't read statecount")
					os.Exit(1)
				}
				tok.stateCount = elemint[0]
				tok.transitions = make([]map[int]*edge, elemint[0]+1)
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

				// While the states in foma start with 0, the states in the
				// Mizobuchi FSA start with one - so we increase every state by 1.

				nontoken := false
				tokenend := false

				// ID needs to be > 1
				inSym++
				outSym++

				if inSym != outSym {

					if tok.sigmaRev[outSym] == NEWLINE {
						tokenend = true
					} else if outSym == tok.epsilon {
						nontoken = true
					} else {
						log.Error().Msg(
							"Unsupported transition: " +
								strconv.Itoa(state) +
								" -> " + strconv.Itoa(end) +
								" (" +
								strconv.Itoa(inSym) +
								":" +
								strconv.Itoa(outSym) +
								") (" +
								string(tok.sigmaRev[inSym]) +
								":" +
								string(tok.sigmaRev[outSym]) +
								")")
						os.Exit(1)
					}

				} else if inSym == tok.epsilon {
					log.Error().Msg("Epsilon transitions not supported")
					os.Exit(1)
				}

				// This collects all edges until arrstate changes

				// TODO:
				//   if arrin == EPSILON && arrout == TOKENEND, mark state as newline
				//   if the next transition is the same, remove TOKENEND and add SENTENCEEND
				//   This requires to remove the transition alltogether and marks the state instead.

				// TODO:
				//   if arrout == EPSILON, mark the transition as NOTOKEN

				targetObj := &edge{
					inSym:    inSym,
					outSym:   outSym,
					end:      end + 1,
					tokenend: tokenend,
					nontoken: nontoken,
				}

				// Initialize outgoing states
				if tok.transitions[state+1] == nil {
					tok.transitions[state+1] = make(map[int]*edge)
				}

				// Ignore transitions with invalid symbols
				if inSym >= 0 {
					tok.transitions[state+1][inSym] = targetObj
				}

				// Add final transition
				if final == 1 {
					// TODO: maybe this is irrelevant for tokenizers
					tok.transitions[state+1][tok.final] = &edge{}
				}

				test := fmt.Sprint(state+1) + ":" + fmt.Sprint(inSym)
				if checkmap[test] {
					fmt.Println("Path already defined!", test)
					os.Exit(0)
				} else {
					checkmap[test] = true
				}

				if DEBUG {
					fmt.Println("Add",
						state+1, "->", end+1,
						"(",
						inSym,
						":",
						outSym,
						") (",
						string(tok.sigmaRev[inSym]),
						":",
						string(tok.sigmaRev[outSym]),
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
					log.Error().Err(err)
					os.Exit(0)
				}

				tok.sigmaCount = number

				var symbol rune

				// Read rune
				if utf8.RuneCountInString(elem[1]) == 1 {
					symbol = []rune(elem[1])[0]

					// Probably a MCS
				} else if utf8.RuneCountInString(elem[1]) > 1 {
					switch elem[1] {
					case "@_EPSILON_SYMBOL_@":
						{
							tok.epsilon = number
							continue
						}
					case "@_UNKNOWN_SYMBOL_@":
						{
							tok.unknown = number
							continue
						}

					case "@_IDENTITY_SYMBOL_@":
						{
							tok.identity = number
							continue
						}
					default:
						{
							log.Error().Msg("MCS not supported: " + line)
							os.Exit(1)
						}
					}

				} else { // Probably a new line symbol
					line, err = r.ReadString('\n')
					if err != nil {
						log.Error().Err(err)
						os.Exit(0)
					}
					if len(line) != 1 {
						log.Error().Msg("MCS not supported:" + line)
						os.Exit(0)
					}
					symbol = rune(NEWLINE)
				}

				tok.sigmaRev[number] = symbol
			}
		}
	}

	return tok
}

// Set alphabet A to the list of all symbols
// outgoing from s
func (tok *Tokenizer) get_set(s int, A *[]int) {
	for a := range tok.transitions[s] {
		*A = append(*A, a)
	}

	// Not required, but simplifies bug hunting
	sort.Ints(*A)
}

// Implementation of Mizobuchi et al (2000), p.128
func (tok *Tokenizer) ToDoubleArray() *DaTokenizer {

	dat := &DaTokenizer{
		sigma:      make(map[rune]int),
		loadFactor: -1,
		final:      tok.final,
		unknown:    tok.unknown,
		identity:   tok.identity,
		epsilon:    tok.epsilon,
		// lastFilledBase: 1,
	}

	for num, sym := range tok.sigmaRev {
		dat.sigma[sym] = num
	}

	mark := 0
	size := 0

	// Create a mapping from s to t
	table := make([]*mapping, tok.arcCount+1)

	table[size] = &mapping{source: 1, target: 1}
	size++

	// Allocate space for the outgoing symbol range
	A := make([]int, 0, tok.sigmaCount)

	for mark < size {
		s := table[mark].source // This is a state in Ms
		t := table[mark].target // This is a state in Mt
		mark++

		if t == 6288 {
			fmt.Println("1 State", t, "was", s)
		}

		// Following the paper, here the state t can be remembered
		// in the set of states St
		A = A[:0]
		tok.get_set(s, &A)

		// Set base to the first free slot in the double array
		dat.setBase(t, dat.xCheck(A))

		// TODO:
		//   Sort the outgoing transitions based on the
		//   outdegree of .end

		// Iterate over all outgoing symbols
		for _, a := range A {

			if a != tok.final {

				// Aka g(s, a)
				s1 := tok.transitions[s][a].end

				// Store the transition
				t1 := dat.getBase(t) + uint32(a)
				dat.setCheck(t1, t)

				if DEBUG || t1 == 6288 {
					fmt.Println("Translate transition",
						s, "->", s1, "(", a, ")", "to", t, "->", t1)
				}

				// Mark the state as being the target of a nontoken transition
				if tok.transitions[s][a].nontoken {
					dat.setNonToken(t1, true)
					if DEBUG {
						fmt.Println("Set", t1, "to nontoken")
					}
				}

				// Mark the state as being the target of a tokenend transition
				if tok.transitions[s][a].tokenend {
					dat.setTokenEnd(t1, true)
					if DEBUG {
						fmt.Println("Set", t1, "to tokenend")
					}
				}

				// Check for representative states
				r := in_table(s1, table, size)

				if r == 0 {
					// Remember the mapping
					table[size] = &mapping{source: s1, target: t1}
					size++
				} else {
					// Overwrite with the representative state
					dat.setBase(t1, r)
					dat.setSeparate(t1, true)
				}
			} else {
				// Store a final transition
				dat.setCheck(dat.getBase(t)+uint32(dat.final), t)
			}
		}
	}

	// Following Mizobuchi et al (2000) the size of the
	// FSA should be stored in check(1).
	dat.setSize(dat.maxSize + 1)
	dat.array = dat.array[:dat.maxSize+1]
	return dat
}

// Check the table if a mapping of s
// exists and return this as a representative.
// Currently iterates through the whole table
// in a bruteforce manner.
func in_table(s int, table []*mapping, size int) uint32 {
	for x := 0; x < size; x++ {
		if table[x].source == s {
			return table[x].target
		}
	}
	return 0
}

// Resize double array when necessary
func (dat *DaTokenizer) resize(l int) {
	// TODO:
	//   This is a bit too aggressive atm and should be calmed down.
	if len(dat.array) <= l {
		dat.array = append(dat.array, make([]uint32, l)...)
	}
}

// Set base value in double array
func (dat *DaTokenizer) setBase(p uint32, v uint32) {
	l := int(p*2 + 1)
	dat.resize(l)
	if dat.maxSize < l {
		dat.maxSize = l
	}
	dat.array[p*2] = v
}

// Returns true if a state is separate pointing to a representative
func (dat *DaTokenizer) isSeparate(p uint32) bool {
	return dat.array[p*2]&firstBit != 0
}

// Mark a state as separate pointing to a representative
func (dat *DaTokenizer) setSeparate(p uint32, sep bool) {
	if sep {
		dat.array[p*2] |= firstBit
	} else {
		dat.array[p*2] &= (restBit | secondBit)
	}
}

// Returns true if a state is the target of a nontoken transition
func (dat *DaTokenizer) isNonToken(p uint32) bool {
	return dat.array[p*2+1]&firstBit != 0
}

// Mark a state as being the target of a nontoken transition
func (dat *DaTokenizer) setNonToken(p uint32, sep bool) {
	if sep {
		dat.array[p*2+1] |= firstBit
	} else {
		dat.array[p*2+1] &= (restBit | secondBit)
	}
}

// Returns true if a state is the target of a tokenend transition
func (dat *DaTokenizer) isTokenEnd(p uint32) bool {
	return dat.array[p*2+1]&secondBit != 0
}

// Mark a state as being the target of a tokenend transition
func (dat *DaTokenizer) setTokenEnd(p uint32, sep bool) {
	if sep {
		dat.array[p*2+1] |= secondBit
	} else {
		dat.array[p*2+1] &= (restBit | firstBit)
	}
}

// Get base value in double array
func (dat *DaTokenizer) getBase(p uint32) uint32 {
	if int(p*2) >= len(dat.array) {
		return 0
	}
	return dat.array[p*2] & restBit
}

// Set check value in double array
func (dat *DaTokenizer) setCheck(p uint32, v uint32) {
	l := int(p*2 + 1)
	dat.resize(l)
	if dat.maxSize < l {
		dat.maxSize = l
	}
	dat.array[(p*2)+1] = v
}

// Get check value in double array
func (dat *DaTokenizer) getCheck(p uint32) uint32 {
	if int((p*2)+1) >= len(dat.array) {
		return 0
	}
	return dat.array[(p*2)+1] & restBit
}

// Set size of double array
func (dat *DaTokenizer) setSize(v int) {
	dat.setCheck(1, uint32(v))
}

// Get size of double array
func (dat *DaTokenizer) GetSize() int {
	return int(dat.getCheck(1))
}

// Based on Mizobuchi et al (2000), p. 124
// This iterates for every state through the complete double array
// structure until it finds a gap that fits all outgoing transitions
// of the state. This is extremely slow, but is only necessary in the
// construction phase of the tokenizer.
func (dat *DaTokenizer) xCheck(symbols []int) uint32 {

	// Start at the first entry of the double array list
	base := uint32(1) // dat.lastFilledBase
	// skip := false
OVERLAP:

	/*
		if !skip {
			if dat.getCheck(base) != 0 {
				dat.lastFilledBase = base
			} else {
				skip = true
			}
		}
	*/

	// Resize the array if necessary
	dat.resize((int(base) + dat.final) * 2)
	for _, a := range symbols {
		if dat.getCheck(base+uint32(a)) != 0 {
			base++
			goto OVERLAP
		}
	}
	return base
}

// List all outgoing transitions for a state
// for testing purposes
func (dat *DaTokenizer) outgoing(t uint32) []int {

	valid := make([]int, 0, len(dat.sigma))

	for _, a := range dat.sigma {
		t1 := dat.getBase(t) + uint32(a)
		if t1 <= dat.getCheck(1) && dat.getCheck(t1) == t {
			valid = append(valid, a)
		}
	}

	for _, a := range []int{dat.epsilon, dat.unknown, dat.identity, dat.final} {
		t1 := dat.getBase(t) + uint32(a)
		if t1 <= dat.getCheck(1) && dat.getCheck(t1) == t {
			valid = append(valid, -1*a)
		}
	}

	sort.Ints(valid)

	return valid
}

// LoadFactor as defined in Kanda et al (2018),
// i.e. the proportion of non-empty elements to all elements.
func (dat *DaTokenizer) LoadFactor() float64 {

	// Cache the loadfactor
	if dat.loadFactor > 0 {
		return dat.loadFactor
	}
	nonEmpty := 0
	all := len(dat.array) / 2
	for x := 1; x <= len(dat.array); x = x + 2 {
		if dat.array[x] != 0 {
			nonEmpty++
		}
	}
	dat.loadFactor = float64(nonEmpty) / float64(all) * 100
	return dat.loadFactor
}

// WriteTo stores the double array data in an io.Writer.
func (dat *DaTokenizer) Save(file string) (n int64, err error) {
	f, err := os.Create(file)
	if err != nil {
		log.Error().Err(err)
		return 0, nil
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	n, err = dat.WriteTo(gz)
	if err != nil {
		log.Error().Err(err)
		return n, err
	}
	gz.Flush()
	return n, nil
}

// WriteTo stores the double array data in an io.Writer.
func (dat *DaTokenizer) WriteTo(w io.Writer) (n int64, err error) {

	wb := bufio.NewWriter(w)
	defer wb.Flush()

	// Store magical header
	all, err := wb.Write([]byte(MAGIC))
	if err != nil {
		log.Error().Err(err)
		return int64(all), err
	}

	// Get sigma as a list
	sigmalist := make([]rune, len(dat.sigma)+16)
	max := 0
	for sym, num := range dat.sigma {
		sigmalist[num] = sym
		if num > max {
			max = num
		}
	}

	sigmalist = sigmalist[:max+1]

	buf := make([]byte, 0, 16)
	bo.PutUint16(buf[0:2], VERSION)
	bo.PutUint16(buf[2:4], uint16(dat.epsilon))
	bo.PutUint16(buf[4:6], uint16(dat.unknown))
	bo.PutUint16(buf[6:8], uint16(dat.identity))
	bo.PutUint16(buf[8:10], uint16(dat.final))
	bo.PutUint16(buf[10:12], uint16(len(sigmalist)))
	bo.PutUint32(buf[12:16], uint32(len(dat.array)))
	more, err := wb.Write(buf[0:16])
	if err != nil {
		log.Error().Err(err)
		return int64(all), err
	}

	all += more

	//  wbuf := bytes.NewBuffer(nil)
	// wbufWrap := bufio.NewWriter(wbuf)

	// Write sigma
	for _, sym := range sigmalist {

		more, err = wb.WriteRune(sym)
		if err != nil {
			log.Error().Err(err)
			return int64(all), err
		}
		all += more
	}
	// wbufWrap.Flush()
	// more, err = w.Write(wbuf.Bytes())
	if err != nil {
		log.Error().Err(err)
		return int64(all), err
	}
	// all += more

	// Test marker - could be checksum
	more, err = wb.Write([]byte("T"))
	if err != nil {
		log.Error().Err(err)
		return int64(all), err
	}
	all += more

	// wbuf.Reset()

	for x := 0; x < len(dat.array); x++ {
		//	for _, d := range dat.array {
		bo.PutUint32(buf[0:4], dat.array[x])
		more, err := wb.Write(buf[0:4])
		if err != nil {
			log.Error().Err(err)
			return int64(all), err
		}
		if more != 4 {
			log.Error().Msg("Can not write uint32")
			return int64(all), err
		}
		all += more
	}

	// wbufWrap.Flush()

	return int64(all), err
}

func LoadDatokFile(file string) *DaTokenizer {
	f, err := os.Open(file)
	if err != nil {
		log.Error().Err(err)
		os.Exit(0)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		log.Error().Err(err)
		os.Exit(0)
	}
	defer gz.Close()

	// Todo: Read the whole file!
	return ParseDatok(gz)
}

func ParseDatok(ior io.Reader) *DaTokenizer {

	dat := &DaTokenizer{
		sigma:      make(map[rune]int),
		epsilon:    0,
		unknown:    0,
		identity:   0,
		final:      0,
		loadFactor: 0,
	}

	r := bufio.NewReader(ior)

	all := 0

	buf := make([]byte, 1024)
	buf = buf[0:len(MAGIC)]

	more, err := r.Read(buf)

	if err != nil {
		log.Error().Err(err)
		return nil
	}

	all += more

	if string(MAGIC) != string(buf) {
		log.Error().Msg("Not a datok file")
		return nil
	}

	more, err = io.ReadFull(r, buf[0:16])
	if err != nil {
		log.Error().Err(err)
		return nil
	}

	all += more

	version := bo.Uint16(buf[0:2])

	if version != VERSION {
		log.Error().Msg("Version not compatible")
		return nil
	}

	dat.epsilon = int(bo.Uint16(buf[2:4]))
	dat.unknown = int(bo.Uint16(buf[4:6]))
	dat.identity = int(bo.Uint16(buf[6:8]))
	dat.final = int(bo.Uint16(buf[8:10]))

	sigmaCount := int(bo.Uint16(buf[10:12]))
	arraySize := int(bo.Uint32(buf[12:16]))

	// Shouldn't be relevant though
	dat.maxSize = arraySize - 1

	for x := 0; x < sigmaCount; x++ {
		sym, more, err := r.ReadRune()
		if err == nil && sym != 0 {
			dat.sigma[sym] = x
		}
		all += more
	}

	more, err = io.ReadFull(r, buf[0:1])

	if err != nil {
		log.Error().Err(err)
		return nil
	}

	all += more

	if string("T") != string(buf[0:1]) {
		log.Error().Msg("Not a datok file")
		return nil
	}

	// Read based on length
	dat.array = make([]uint32, arraySize)

	for x := 0; x < arraySize; x++ {
		more, err = io.ReadFull(r, buf[0:4])
		if err != nil {
			if err == io.EOF {
				fmt.Println(arraySize, x)
				break
			}
			log.Error().Err(err)
			return nil
		}
		all += more
		dat.array[x] = bo.Uint32(buf[0:4])
	}

	return dat
}

// Match an input string against the double array
// FSA.
//
// Based on Mizobuchi et al (2000), p. 129,
// with additional support for IDENTITY, UNKNOWN
// and EPSILON transitions.
func (dat *DaTokenizer) Match(input string) bool {
	var a int
	var tu uint32
	var ok bool

	t := uint32(1) // Initial state
	chars := []rune(input)
	i := 0

	for i < len(chars) {
		a, ok = dat.sigma[chars[i]]

		// Support identity symbol if character is not in sigma
		if !ok && dat.identity != -1 {
			if DEBUG {
				fmt.Println("IDENTITY symbol", string(chars[i]), "->", dat.identity)
			}
			a = dat.identity
		} else if DEBUG {
			fmt.Println("Sigma transition is okay for [", string(chars[i]), "]")
		}
		tu = t
	CHECK:
		t = dat.getBase(tu) + uint32(a)

		// Check if the transition is valid according to the double array
		if t > dat.getCheck(1) || dat.getCheck(t) != tu {

			if DEBUG {
				fmt.Println("Match is not fine!", t, "and", dat.getCheck(t), "vs", tu)
			}

			if !ok && a == dat.identity {
				// Try again with unknown symbol, in case identity failed
				if DEBUG {
					fmt.Println("UNKNOWN symbol", string(chars[i]), "->", dat.unknown)
				}
				a = dat.unknown

			} else if a != dat.epsilon {
				// Try again with epsilon symbol, in case everything else failed
				if DEBUG {
					fmt.Println("EPSILON symbol", string(chars[i]), "->", dat.epsilon)
				}
				a = dat.epsilon
			} else {
				break
			}
			goto CHECK
		} else if dat.isSeparate(t) {
			// Move to representative state
			t = dat.getBase(t)
		}

		// Transition is fine
		if a != dat.epsilon {
			// Character consumed
			i++
		}

		// TODO:
		//   Prevent endless epsilon loops!
	}

	if i != len(chars) {
		if DEBUG {
			fmt.Println("Not at the end")
		}
		return false
	}

FINALCHECK:

	// Automaton is in a final state
	if dat.getCheck(dat.getBase(t)+uint32(dat.final)) == t {
		return true
	}

	// Check epsilon transitions until a final state is reached
	tu = t
	t = dat.getBase(tu) + uint32(dat.epsilon)

	// Epsilon transition failed
	if t > dat.getCheck(1) || dat.getCheck(t) != tu {
		if DEBUG {
			fmt.Println("Match is not fine!", t, "and", dat.getCheck(t), "vs", tu)
		}
		return false

	} else if dat.isSeparate(t) {
		// Move to representative state
		t = dat.getBase(t)
	}

	goto FINALCHECK
}

func showBuffer(buffer []rune, buffo int, buffi int) string {
	out := make([]rune, 0, 1024)
	for x := 0; x < len(buffer); x++ {
		if buffi == x {
			out = append(out, '^')
		}
		if buffo == x {
			out = append(out, '[', buffer[x], ']')
		} else {
			out = append(out, buffer[x])
		}
	}
	return string(out)
}

// Transduce an input string against the double array
// FSA. The rules are always greedy. If the automaton fails,
// it takes the last possible token ending branch.
//
// Based on Match with additional support
// for NONTOKEN and TOKENEND handling
func (dat *DaTokenizer) Transduce(r io.Reader, w io.Writer) bool {
	var a int
	var t0 uint32
	var ok, nontoken, tokenend bool

	// Remember the last position of a possible tokenend,
	// in case the automaton fails.
	epsilonState := uint32(0)
	epsilonOffset := 0

	// Implement a low level buffer for full control,
	// however - it is probably better to introduce
	// this on a higher level with a io.Reader interface
	// The buffer stores a single word and may have white
	// space at the end (but not at the beginning).
	//
	// This is the only backtracking requirement because of
	// epsilon transitions, to support tokenizations like:
	// "this is an example|.| And it works." vs
	// "this is an example.com| application."
	buffer := make([]rune, 1024)
	buffo := 0 // Buffer offset
	buffi := 0 // Buffer length

	reader := bufio.NewReader(r)
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	t := uint32(1) // Initial state

	var char rune
	var err error
	eof := false

	// TODO:
	//   Write all characters first into a buffer
	//   and flush when necessary
	// TODO:
	//   Create an epsilon stack
	for {

		// Get from reader if buffer is empty
		if buffo >= buffi {
			char, _, err = reader.ReadRune()
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

		a, ok = dat.sigma[char]

		// Support identity symbol if character is not in sigma
		if !ok && dat.identity != -1 {
			if DEBUG {
				fmt.Println("IDENTITY symbol", string(char), "->", dat.identity)
			}
			a = dat.identity
		}

		t0 = t

		if dat.getCheck(dat.getBase(t0)+uint32(dat.epsilon)) == t0 {
			if DEBUG {
				fmt.Println("Remember for epsilon tu:charcount", t0, buffo)
			}
			epsilonState = t0
			epsilonOffset = buffo
		}

	CHECK:
		nontoken = false
		tokenend = false

		t = dat.getBase(t0) + uint32(a)

		if DEBUG {
			fmt.Println("Check", t0, "-", a, "(", string(char), ")", "->", t)
			fmt.Println("Valid:", dat.outgoing(t0))
		}

		// Check if the transition is valid according to the double array
		if t > dat.getCheck(1) || dat.getCheck(t) != t0 {

			if DEBUG {
				fmt.Println("Match is not fine!", t, "and", dat.getCheck(t), "vs", t0)
			}

			if !ok && a == dat.identity {

				// Try again with unknown symbol, in case identity failed
				if DEBUG {
					fmt.Println("UNKNOWN symbol", string(char), "->", dat.unknown)
				}
				a = dat.unknown

			} else if a != dat.epsilon {

				// Try again with epsilon symbol, in case everything else failed
				if DEBUG {
					fmt.Println("EPSILON symbol", string(char), "->", dat.epsilon)
				}
				t0 = epsilonState
				a = dat.epsilon
				epsilonState = 0 // reset
				buffo = epsilonOffset
				if DEBUG {
					fmt.Println("Get from epsilon stack and set buffo!", showBuffer(buffer, buffo, buffi))
				}

			} else {
				break
			}

			goto CHECK

		}

		// Move to representative state
		nontoken = dat.isNonToken(t)
		tokenend = dat.isTokenEnd(t)

		if dat.isSeparate(t) {
			t = dat.getBase(t)

			if DEBUG {
				fmt.Println("Representative pointing to", t)
			}
		}

		// Transition is fine
		if a != dat.epsilon {

			// Character consumed
			buffo++
			if nontoken {

				if DEBUG {
					fmt.Println("Nontoken forward", showBuffer(buffer, buffo, buffi))
				}

				// Maybe remove the first character, if buffo == 0?
				if buffo == 1 {

					// TODO: Better as a ring buffer
					for x, i := range buffer[buffo:buffi] {
						buffer[x] = i
					}
					//	writer.WriteRune('\n')
					buffi -= buffo
					epsilonOffset -= buffo
					buffo = 0
				}
			}
		}

		if DEBUG {
			fmt.Println("  --> ok!")
		}

		if tokenend {
			data := []byte(string(buffer[:buffo]))
			if DEBUG {
				fmt.Println("-> Flush buffer:", string(data), showBuffer(buffer, buffo, buffi))
			}
			writer.Write(data)
			writer.WriteRune('\n')

			// Better as a ring buffer
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

		// TODO:
		//   Prevent endless epsilon loops!
	}

	if !eof {
		if DEBUG {
			fmt.Println("Not at the end - problem", t0, ":", dat.outgoing(t0))
		}
		return false
	}

FINALCHECK:

	// Automaton is in a final state
	if dat.getCheck(dat.getBase(t)+uint32(dat.final)) == t {

		if buffi > 0 {
			data := []byte(string(buffer[:buffi]))
			if DEBUG {
				fmt.Println("-> Flush buffer:", string(data))
			}
			writer.Write(data)
			// states are irrelevant here
		}

		if dat.isTokenEnd(t) {
			writer.WriteRune('\n')
		}

		// There may be a new line at the end, from an epsilon, so we go on!
		return true
	}

	// Check epsilon transitions until a final state is reached
	t0 = t
	t = dat.getBase(t0) + uint32(dat.epsilon)

	// Epsilon transition failed
	if t > dat.getCheck(1) || dat.getCheck(t) != t0 {
		if DEBUG {
			fmt.Println("Match is not fine!", t, "and", dat.getCheck(t), "vs", t0)
		}
		return false
	}

	// nontoken = dat.isNonToken(t)
	tokenend = dat.isTokenEnd(t)

	if dat.isSeparate(t) {
		// Move to representative state
		t = dat.getBase(t)
	}

	if tokenend {
		if buffi > 0 {
			data := []byte(string(buffer[:buffi]))
			if DEBUG {
				fmt.Println("-> Flush buffer:", string(data))
			}
			writer.Write(data)
			buffi = 0
			buffo = 0
			epsilonState = 0
		}
		writer.WriteRune('\n')
	}

	goto FINALCHECK
}

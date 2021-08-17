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
//
// Serialization is little endian.

// TODO:
// - Turn sigma into an array instead of using a map
//   and improve the mapping beforehand so that ASCII
//   is mapped directly and only non-ASCII needs to be
//   looked up in a map or similar.
// - replace maxSize with the check value
// - Add checksum to serialization.
// - Instead of memoizing the loadFactor, better remember
//   the number of set transitions
// - Replace/Enhance table with a map
// - Provide a bufio.Scanner compatible interface.
// - Mark epsilon transitions in bytes

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

	"log"
)

const (
	PROPS            = 1
	SIGMA            = 2
	STATES           = 3
	NONE             = 4
	DEBUG            = false
	MAGIC            = "DATOK"
	VERSION          = uint16(1)
	FIRSTBIT  uint32 = 1 << 31
	SECONDBIT uint32 = 1 << 30
	RESTBIT   uint32 = ^uint32(0) &^ (FIRSTBIT | SECONDBIT)
)

// Serialization is always little endian
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

// Tokenizer is the intermediate representation
// of the tokenizer.
type Tokenizer struct {
	sigmaRev    map[int]rune
	arcCount    int
	sigmaCount  int
	transitions []map[int]*edge

	// Special symbols in sigma
	epsilon  int
	unknown  int
	identity int
	final    int
	tokenend int
}

type bc struct {
	base  uint32
	check uint32
}

// DaTokenizer represents a tokenizer implemented as a
// Double Array FSA.
type DaTokenizer struct {
	sigma      map[rune]int
	sigmaASCII [256]int
	maxSize    int
	loadFactor float64
	array      []bc

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
func LoadFomaFile(file string) *Tokenizer {
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
func ParseFoma(ior io.Reader) *Tokenizer {
	r := bufio.NewReader(ior)

	tok := &Tokenizer{
		sigmaRev: make(map[int]rune),
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
				tok.sigmaCount++
				tok.final = tok.sigmaCount

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
				tok.arcCount = elemint[0]

				elemint[0], err = strconv.Atoi(elem[2])
				if err != nil {
					log.Print("Can't read statecount")
					return nil
				}

				// States start at 1 in Mizobuchi et al (2000),
				// as the state 0 is associated with a fail.
				// Initialize states and transitions
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

				nontoken := false
				tokenend := false

				// While the states in foma start with 0, the states in the
				// Mizobuchi FSA start with one - so we increase every state by 1.
				// We also increase sigma by 1, so there are no 0 transitions.
				inSym++
				outSym++

				// Only a limited list of transitions are allowed
				if inSym != outSym {
					if outSym == tok.tokenend && inSym == tok.epsilon {
						tokenend = true
					} else if outSym == tok.epsilon {
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
								string(tok.sigmaRev[inSym]) +
								":" +
								string(tok.sigmaRev[outSym]) +
								")")
						return nil
					}

				} else if inSym == tok.epsilon {
					log.Println("General epsilon transitions are not supported")
					return nil
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
				if tok.transitions[state+1] == nil {
					tok.transitions[state+1] = make(map[int]*edge)
				}

				// Ignore transitions with invalid symbols
				if inSym >= 0 {
					tok.transitions[state+1][inSym] = targetObj
				}

				// Add final transition
				if final == 1 {
					// TODO:
					//   Maybe this is less relevant for tokenizers
					tok.transitions[state+1][tok.final] = &edge{}
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
					log.Println(err)
					return nil
				}

				tok.sigmaCount = number

				var symbol rune

				// Read rune
				if utf8.RuneCountInString(elem[1]) == 1 {
					symbol = []rune(elem[1])[0]

				} else if utf8.RuneCountInString(elem[1]) > 1 {

					// Probably a MCS
					switch elem[1] {
					case "@_EPSILON_SYMBOL_@":
						{
							tok.epsilon = number
						}
					case "@_UNKNOWN_SYMBOL_@":
						{
							tok.unknown = number
						}

					case "@_IDENTITY_SYMBOL_@":
						{
							tok.identity = number
						}

					case "@_TOKEN_SYMBOL_@":
						{
							tok.tokenend = number
						}
					default:
						{
							log.Println("MCS not supported: " + line)
							return nil
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
						log.Println("MCS not supported:" + line)
						return nil
					}
					symbol = rune('\n')
				}

				tok.sigmaRev[number] = symbol
			}
		}
	}

	return tok
}

// Set alphabet A to the list of all symbols
// outgoing from s
func (tok *Tokenizer) getSet(s int, A *[]int) {
	for a := range tok.transitions[s] {
		*A = append(*A, a)
	}

	// Not required, but simplifies bug hunting
	// sort.Ints(*A)
}

// ToDoubleArray turns the intermediate tokenizer representation
// into a double array representation.
//
// This is based on Mizobuchi et al (2000), p.128
func (tok *Tokenizer) ToDoubleArray() *DaTokenizer {

	dat := &DaTokenizer{
		sigma:      make(map[rune]int),
		loadFactor: -1,
		final:      tok.final,
		unknown:    tok.unknown,
		identity:   tok.identity,
		epsilon:    tok.epsilon,
		tokenend:   tok.tokenend,
	}

	dat.resize(dat.final)

	for num, sym := range tok.sigmaRev {
		if int(sym) < 256 {
			dat.sigmaASCII[int(sym)] = num
		}
		dat.sigma[sym] = num
	}

	mark := 0
	size := 0
	var base uint32

	// Create a mapping from s (in Ms aka Intermediate FSA)
	// to t (in Mt aka Double Array FSA)
	table := make([]*mapping, tok.arcCount+1)

	// Initialize with the start state
	table[size] = &mapping{source: 1, target: 1}
	size++

	// Allocate space for the outgoing symbol range
	A := make([]int, 0, tok.sigmaCount)

	for mark < size {
		s := table[mark].source // This is a state in Ms
		t := table[mark].target // This is a state in Mt
		mark++

		// Following the paper, here the state t can be remembered
		// in the set of states St
		A = A[:0]
		tok.getSet(s, &A)

		// Set base to the first free slot in the double array
		base = dat.xCheck(A)
		dat.array[t].setBase(base)

		// TODO:
		//   Sort the outgoing transitions based on the
		//   outdegree of .end

		// Iterate over all outgoing symbols
		for _, a := range A {

			if a != tok.final {

				// Aka g(s, a)
				s1 := tok.transitions[s][a].end

				// Store the transition
				t1 := base + uint32(a)
				dat.array[t1].setCheck(t)

				// Set maxSize
				if dat.maxSize < int(t1) {
					dat.maxSize = int(t1)
				}

				if DEBUG {
					fmt.Println("Translate transition",
						s, "->", s1, "(", a, ")", "to", t, "->", t1)
				}

				// Mark the state as being the target of a nontoken transition
				if tok.transitions[s][a].nontoken {
					dat.array[t1].setNonToken(true)
					if DEBUG {
						fmt.Println("Set", t1, "to nontoken")
					}
				}

				// Mark the state as being the target of a tokenend transition
				if tok.transitions[s][a].tokenend {
					dat.array[t1].setTokenEnd(true)
					if DEBUG {
						fmt.Println("Set", t1, "to tokenend")
					}
				}

				// Check for representative states
				r := stateAlreadyInTable(s1, table, size)

				// No representative found
				if r == 0 {
					// Remember the mapping
					table[size] = &mapping{source: s1, target: t1}
					size++
				} else {
					// Overwrite with the representative state
					dat.array[t1].setBase(r)
					dat.array[t1].setSeparate(true)
				}
			} else {
				// Store a final transition
				dat.array[base+uint32(dat.final)].setCheck(t)

				if dat.maxSize < int(base+uint32(dat.final)) {
					dat.maxSize = int(base + uint32(dat.final))
				}
			}
		}
	}

	// Following Mizobuchi et al (2000) the size of the
	// FSA should be stored in check(1).
	// We make the size a bit smaller so we never have to check for boundaries.
	dat.setSize(dat.maxSize + 1)
	dat.array = dat.array[:dat.maxSize+1]
	return dat
}

// Check the table if a mapping of s
// exists and return this as a representative.
// Currently iterates through the whole table
// in a bruteforce manner.
func stateAlreadyInTable(s int, table []*mapping, size int) uint32 {
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
		dat.array = append(dat.array, make([]bc, l)...)
	}
}

// Set base value in double array
func (bc *bc) setBase(v uint32) {
	bc.base = v
}

// Get base value in double array
func (bc *bc) getBase() uint32 {
	return bc.base & RESTBIT
}

// Set check value in double array
func (bc *bc) setCheck(v uint32) {
	bc.check = v
}

// Get check value in double array
func (bc *bc) getCheck() uint32 {
	return bc.check & RESTBIT
}

// Returns true if a state is separate pointing to a representative
func (bc *bc) isSeparate() bool {
	return bc.base&FIRSTBIT != 0
}

// Mark a state as separate pointing to a representative
func (bc *bc) setSeparate(sep bool) {
	if sep {
		bc.base |= FIRSTBIT
	} else {
		bc.base &= (RESTBIT | SECONDBIT)
	}
}

// Returns true if a state is the target of a nontoken transition
func (bc *bc) isNonToken() bool {
	return bc.check&FIRSTBIT != 0
}

// Mark a state as being the target of a nontoken transition
func (bc *bc) setNonToken(sep bool) {
	if sep {
		bc.check |= FIRSTBIT
	} else {
		bc.check &= (RESTBIT | SECONDBIT)
	}
}

// Returns true if a state is the target of a tokenend transition
func (bc *bc) isTokenEnd() bool {
	return bc.check&SECONDBIT != 0
}

// Mark a state as being the target of a tokenend transition
func (bc *bc) setTokenEnd(sep bool) {
	if sep {
		bc.check |= SECONDBIT
	} else {
		bc.check &= (RESTBIT | FIRSTBIT)
	}
}

// Set size of double array
func (dat *DaTokenizer) setSize(v int) {
	dat.array[1].setCheck(uint32(v))
}

// Get size of double array
func (dat *DaTokenizer) GetSize() int {
	return int(dat.array[1].getCheck())
}

// Based on Mizobuchi et al (2000), p. 124
// This iterates for every state through the complete double array
// structure until it finds a gap that fits all outgoing transitions
// of the state. This is extremely slow, but is only necessary in the
// construction phase of the tokenizer.
func (dat *DaTokenizer) xCheck(symbols []int) uint32 {

	// Start at the first entry of the double array list
	base := uint32(1)

OVERLAP:
	// Resize the array if necessary
	dat.resize(int(base) + dat.final)
	for _, a := range symbols {
		if dat.array[int(base)+a].getCheck() != 0 {
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
		t1 := dat.array[t].getBase() + uint32(a)
		if t1 <= dat.array[1].getCheck() && dat.array[t1].getCheck() == t {
			valid = append(valid, a)
		}
	}

	for _, a := range []int{dat.epsilon, dat.unknown, dat.identity, dat.final} {
		t1 := dat.array[t].getBase() + uint32(a)
		if t1 <= dat.array[1].getCheck() && dat.array[t1].getCheck() == t {
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
	for x := 1; x < len(dat.array); x++ {
		if dat.array[x].getBase() != 0 {
			nonEmpty++
		}
	}
	dat.loadFactor = float64(nonEmpty) / float64(all) * 100
	return dat.loadFactor
}

// Save stores the double array data in a file
func (dat *DaTokenizer) Save(file string) (n int64, err error) {
	f, err := os.Create(file)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	n, err = dat.WriteTo(gz)
	if err != nil {
		log.Println(err)
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
		log.Println(err)
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
	bo.PutUint32(buf[12:16], uint32(len(dat.array)*2)) // Legacy support
	more, err := wb.Write(buf[0:16])
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
	more, err = wb.Write([]byte("T"))
	if err != nil {
		log.Println(err)
		return int64(all), err
	}
	all += more

	// for x := 0; x < len(dat.array); x++ {
	for _, bc := range dat.array {
		bo.PutUint32(buf[0:4], bc.base)
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
		bo.PutUint32(buf[0:4], bc.check)
		more, err = wb.Write(buf[0:4])
		if err != nil {
			log.Println(err)
			return int64(all), err
		}
		all += more
		if more != 4 {
			log.Println("Can not write check uint32")
			return int64(all), err
		}
	}

	return int64(all), err
}

// LoadDatokFile reads a double array represented tokenizer
// from a file.
func LoadDatokFile(file string) *DaTokenizer {
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
	return ParseDatok(gz)
}

// LoadDatokFile reads a double array represented tokenizer
// from an io.Reader
func ParseDatok(ior io.Reader) *DaTokenizer {

	// Initialize tokenizer with default values
	dat := &DaTokenizer{
		sigma:      make(map[rune]int),
		epsilon:    0,
		unknown:    0,
		identity:   0,
		final:      0,
		loadFactor: 0,
	}

	r := bufio.NewReader(ior)

	buf := make([]byte, 1024)
	buf = buf[0:len(MAGIC)]

	_, err := r.Read(buf)

	if err != nil {
		log.Println(err)
		return nil
	}

	if string(MAGIC) != string(buf) {
		log.Println("Not a datok file")
		return nil
	}

	more, err := io.ReadFull(r, buf[0:16])
	if err != nil {
		log.Println(err)
		return nil
	}

	if more != 16 {
		log.Println("Read bytes do not fit")
		return nil
	}

	version := bo.Uint16(buf[0:2])

	if version != VERSION {
		log.Println("Version not compatible")
		return nil
	}

	dat.epsilon = int(bo.Uint16(buf[2:4]))
	dat.unknown = int(bo.Uint16(buf[4:6]))
	dat.identity = int(bo.Uint16(buf[6:8]))
	dat.final = int(bo.Uint16(buf[8:10]))

	sigmaCount := int(bo.Uint16(buf[10:12]))
	arraySize := int(bo.Uint32(buf[12:16])) / 2 // Legacy support

	// Shouldn't be relevant though
	dat.maxSize = arraySize - 1

	for x := 0; x < sigmaCount; x++ {
		sym, _, err := r.ReadRune()
		if err == nil && sym != 0 {
			if int(sym) < 256 {
				dat.sigmaASCII[int(sym)] = x
			}
			dat.sigma[sym] = x
		}
	}

	_, err = io.ReadFull(r, buf[0:1])

	if err != nil {
		log.Print(err)
		return nil
	}

	if string("T") != string(buf[0:1]) {
		log.Println("Not a datok file")
		return nil
	}

	// Read based on length
	dat.array = make([]bc, arraySize)

	dataArray, err := io.ReadAll(r)

	if err == io.EOF {
		log.Println(err)
		return nil
	}

	if len(dataArray) < arraySize*8 {
		log.Println("Not enough bytes read")
		return nil
	}

	for x := 0; x < arraySize; x++ {
		dat.array[x].base = bo.Uint32(dataArray[x*8 : (x*8)+4])
		dat.array[x].check = bo.Uint32(dataArray[(x*8)+4 : (x*8)+8])
	}

	return dat
}

// Show the current state of the buffer,
// for testing puroses
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
// Based on Mizobuchi et al (2000), p. 129,
// with additional support for IDENTITY, UNKNOWN
// and EPSILON transitions and NONTOKEN and TOKENEND handling.
func (dat *DaTokenizer) Transduce(r io.Reader, w io.Writer) bool {
	var a int
	var t0 uint32
	t := uint32(1) // Initial state
	var ok, rewindBuffer bool

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
	//
	// TODO:
	//   Store a translation buffer as well, so characters don't
	//   have to be translated multiple times!
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

PARSECHAR:
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
				a = dat.sigmaASCII[int(char)]
			} else {
				a, ok = dat.sigma[char]
				if !ok {
					a = 0
				}
			}

			// Use identity symbol if character is not in sigma
			if a == 0 && dat.identity != -1 {
				a = dat.identity
			}

			t0 = t

			// Check for epsilon transitions and remember
			if dat.array[dat.array[t0].getBase()+uint32(dat.epsilon)].getCheck() == t0 {
				// Remember state for backtracking to last tokenend state
				epsilonState = t0
				epsilonOffset = buffo
			}
		}

		// Checks a transition based on t0, a and buffo
		t = dat.array[t0].getBase() + uint32(a)
		ta := dat.array[t]

		if DEBUG {
			// Char is only relevant if set
			fmt.Println("Check", t0, "-", a, "(", string(char), ")", "->", t)
			if false {
				fmt.Println(dat.outgoing(t0))
			}
		}

		// Check if the transition is invalid according to the double array
		if t > dat.array[1].getCheck() || ta.getCheck() != t0 {

			if DEBUG {
				fmt.Println("Match is not fine!", t, "and", ta.getCheck(), "vs", t0)
			}

			if !ok && a == dat.identity {

				// Try again with unknown symbol, in case identity failed
				// Char is only relevant when set
				if DEBUG {
					fmt.Println("UNKNOWN symbol", string(char), "->", dat.unknown)
				}
				a = dat.unknown

			} else if a != dat.epsilon {

				// Try again with epsilon symbol, in case everything else failed
				t0 = epsilonState
				epsilonState = 0 // reset
				buffo = epsilonOffset
				a = dat.epsilon

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
		if a != dat.epsilon {

			buffo++

			// Transition does not produce a character
			if buffo == 1 && ta.isNonToken() {
				if DEBUG {
					fmt.Println("Nontoken forward", showBuffer(buffer, buffo, buffi))
				}
				rewindBuffer = true
			}
		}

		// Transition marks the end of a token - so flush the buffer
		if ta.isTokenEnd() {

			if buffi > 0 {
				if DEBUG {
					fmt.Println("-> Flush buffer: [", string(buffer[:buffo]), "]", showBuffer(buffer, buffo, buffi))
				}
				writer.WriteString(string(buffer[:buffo]))
				rewindBuffer = true
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
		if ta.isSeparate() {
			t = ta.getBase()
			ta = dat.array[t]

			if DEBUG {
				fmt.Println("Representative pointing to", t)
			}
		}

		newchar = true

		// TODO:
		//   Prevent endless epsilon loops!
	}

	// Input reader is not yet finished
	if !eof {
		if DEBUG {
			fmt.Println("Not at the end - problem", t0, ":", dat.outgoing(t0))
		}
		return false
	}

	if DEBUG {
		fmt.Println("Entering final check")
	}

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

	// Check epsilon transitions until a final state is reached
	t0 = t
	t = dat.array[t0].getBase() + uint32(dat.epsilon)
	a = dat.epsilon
	newchar = false
	if dat.array[t].getCheck() == t0 {
		// Remember state for backtracking to last tokenend state
		goto PARSECHAR

	} else if epsilonState != 0 {
		t0 = epsilonState
		epsilonState = 0 // reset
		buffo = epsilonOffset
		if DEBUG {
			fmt.Println("Get from epsilon stack and set buffo!", showBuffer(buffer, buffo, buffi))
		}
		goto PARSECHAR
	}
	return false
}

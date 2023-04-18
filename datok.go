package datok

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
// - replace maxSize with the check value
// - Check if final states can be optional.
// - Introduce ELM (Morita et al. 2001) to speed
//   up construction. Can be ignored on serialization
//   to improve compression.
// - Add checksum to serialization.
// - Replace/Enhance table with a map
// - Provide a bufio.Scanner compatible interface.
// - Mark epsilon transitions in bytes

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"io"
	"math"
	"os"
	"sort"

	"log"
)

const (
	DEBUG            = false
	DAMAGIC          = "DATOK"
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
	transCount int
	array      []bc

	// Special symbols in sigma
	epsilon  int
	unknown  int
	identity int
	final    int
	tokenend int
}

// ToDoubleArray turns the intermediate tokenizer representation
// into a double array representation.
//
// This is based on Mizobuchi et al (2000), p.128
func (auto *Automaton) ToDoubleArray() *DaTokenizer {

	dat := &DaTokenizer{
		sigma:      make(map[rune]int),
		transCount: -1,
		final:      auto.final,
		unknown:    auto.unknown,
		identity:   auto.identity,
		epsilon:    auto.epsilon,
		tokenend:   auto.tokenend,
	}

	dat.resize(dat.final)

	// Init with identity
	if dat.identity != -1 {
		for i := 0; i < 256; i++ {
			dat.sigmaASCII[i] = dat.identity
		}
	}

	for num, sym := range auto.sigmaRev {
		if int(sym) < 256 {
			dat.sigmaASCII[int(sym)] = num
		}
		dat.sigma[sym] = num
	}

	mark := 0
	size := 0
	var base uint32
	var atrans *edge
	var s, s1 int
	var t, t1 uint32
	var diff int

	// Create a mapping from s (in Ms aka Intermediate FSA)
	// to t (in Mt aka Double Array FSA)
	table := make([]*mapping, auto.arcCount+1)
	// tableQueue := make([]int, tok.arcCount+1)

	// Initialize with the start state
	table[size] = &mapping{source: 1, target: 1}
	// tableQueue[size] = 1
	size++

	// Allocate space for the outgoing symbol range
	A := make([]int, 0, auto.sigmaCount)
	// tableLookup := make([]uint32, tok.arcCount+2) // make(map[int]uint32)
	// tableLookup[1] = 1

	// block_begin_pos := uint32(1)

	for mark < size {
		s = table[mark].source // This is a state in Ms
		t = table[mark].target // This is a state in Mt
		// s = tableQueue[mark]
		// t = tableLookup[s]
		mark++

		// Following the paper, here the state t can be remembered
		// in the set of states St
		A = A[:0]
		auto.getSet(s, &A)

		// Set base to the first free slot in the double array
		// base = dat.xCheck(A)
		// base = dat.xCheckSkip(A)
		// base = dat.xCheckNiu(A, &block_begin_pos)
		base = dat.xCheckSkipNiu(A)
		dat.array[t].setBase(base)

		// TODO:
		//   Sort the outgoing transitions based on the
		//   outdegree of .end

		// Iterate over all outgoing symbols
		for _, a := range A {

			if a != auto.final {

				atrans = auto.transitions[s][a]

				// Aka g(s, a)
				s1 = atrans.end

				// Store the transition
				t1 = base + uint32(a)
				dat.array[t1].setCheck(t)

				// Set maxSize
				if dat.maxSize < int(t1) {
					dat.maxSize = int(t1)
				}

				if DEBUG {
					log.Println("Translate transition",
						s, "->", s1, "(", a, ")", "to", t, "->", t1)
				}

				// Mark the state as being the target of a nontoken transition
				if atrans.nontoken {
					dat.array[t1].setNonToken(true)
					if DEBUG {
						log.Println("Set", t1, "to nontoken")
					}
				}

				// Mark the state as being the target of a tokenend transition
				if atrans.tokenend {
					dat.array[t1].setTokenEnd(true)
					if DEBUG {
						log.Println("Set", t1, "to tokenend")
					}
				}

				// Check for representative states
				r := stateAlreadyInTable(s1, table, size)
				// r := tableLookup[s1]

				// No representative found
				if r == 0 {
					// Remember the mapping
					table[size] = &mapping{source: s1, target: t1}
					// tableQueue[size] = s1
					// tableLookup[s1] = t1
					size++
				} else {
					// Overwrite with the representative state
					dat.array[t1].setBase(r)
					dat.array[t1].setSeparate(true)
				}
			} else {
				// Store a final transition
				dat.array[base+uint32(dat.final)].setCheck(t)

				if dat.maxSize < int(base)+dat.final {
					dat.maxSize = int(base) + dat.final
				}

				// Find max
				// see https://dev.to/jobinrjohnson/branchless-programming-does-it-really-matter-20j4
				diff = dat.maxSize - (int(base) + dat.final)
				dat.maxSize -= (diff & (diff >> 31))
			}
		}
	}

	// Following Mizobuchi et al (2000) the size of the
	// FSA should be stored in check(1).
	// We make the size a bit larger so we never have to check for boundaries.
	dat.setSize(dat.maxSize + dat.final)
	if len(dat.array) < dat.maxSize+dat.final {
		dat.array = append(dat.array, make([]bc, dat.final)...)
	}
	dat.array = dat.array[:dat.maxSize+dat.final]
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

// Type of tokenizer
func (DaTokenizer) Type() string {
	return DAMAGIC
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

// This is an implementation of xCheck with the skip-improvement
// proposed by Morita et al. (2001)
func (dat *DaTokenizer) xCheckSkip(symbols []int) uint32 {

	// Start at the first entry of the double array list
	base := uint32(math.Abs(float64(dat.maxSize-1) * .9))

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

// This is an implementation of xCheck with the skip-improvement
// proposed by Morita et al. (2001) for higher outdegrees as
// proposed by Niu et al. (2013)
func (dat *DaTokenizer) xCheckSkipNiu(symbols []int) uint32 {

	// Start at the first entry of the double array list
	base := uint32(1)

	// Or skip the first few entries
	if len(symbols) >= 3 {
		base = uint32(math.Abs(float64(dat.maxSize-1)*.9)) + 1
	}

OVERLAP:
	// Resize the array if necessary
	dat.resize(int(base) + dat.final + 1)
	for _, a := range symbols {
		if dat.array[int(base)+a].getCheck() != 0 {
			base++
			goto OVERLAP
		}
	}
	return base
}

// This is an implementation of xCheck wit an improvement
// proposed by Niu et al. (2013)
func (dat *DaTokenizer) xCheckNiu(symbols []int, block_begin_pos *uint32) uint32 {

	// Start at the first entry of the double array list
	base := uint32(1)

	if len(symbols) > 3 {
		sort.Ints(symbols)
		if *block_begin_pos > uint32(symbols[0]) {
			dat.resize(int(*block_begin_pos) + dat.final)
			*block_begin_pos += uint32(symbols[len(symbols)-1] + 1)
			return *block_begin_pos - uint32(symbols[0])
		}
	}

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

// TransCount as the number of transitions aka arcs in the
// finite state automaton
func (dat *DaTokenizer) TransCount() int {
	// Cache the transCount
	if dat.transCount > 0 {
		return dat.transCount
	}

	dat.transCount = 0
	for x := 1; x < len(dat.array); x++ {

		// Hopefully branchless
		if dat.array[x].getBase() != 0 {
			dat.transCount++
		}
	}

	return dat.transCount
}

// LoadFactor as defined in Kanda et al (2018),
// i.e. the proportion of non-empty elements to all elements.
func (dat *DaTokenizer) LoadFactor() float64 {
	return float64(dat.TransCount()) / float64(len(dat.array)) * 100
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
	all, err := wb.Write([]byte(DAMAGIC))
	if err != nil {
		log.Println(err)
		return int64(all), err
	}

	// Get sigma as a list
	sigmalist := make([]rune, len(dat.sigma)+16)
	max := 0
	for sym, num := range dat.sigma {
		sigmalist[num] = sym

		// Find max
		max -= ((max - num) & ((max - num) >> 31))
		// if num > max {
		//   max = num
		// }
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
		transCount: 0,
	}

	r := bufio.NewReader(ior)

	buf := make([]byte, 1024)
	buf = buf[0:len(DAMAGIC)]

	_, err := r.Read(buf)

	if err != nil {
		log.Println(err)
		return nil
	}

	if string(DAMAGIC) != string(buf) {
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

	// Init with identity
	if dat.identity != -1 {
		for i := 0; i < 256; i++ {
			dat.sigmaASCII[i] = dat.identity
		}
	}

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

// Show the current state of the buffer,
// for testing puroses
func showBufferNew(buffer []rune, bufft int, buffc int, buffi int) string {
	out := make([]rune, 0, 1024)
	for x := 0; x < len(buffer); x++ {
		if buffi == x {
			out = append(out, '^')
		}
		if bufft == x {
			out = append(out, '|')
		}
		if buffc == x {
			out = append(out, '[', buffer[x], ']')
		} else {
			out = append(out, buffer[x])
		}
	}
	return string(out)
}

// Transduce input to ouutput
func (dat *DaTokenizer) Transduce(r io.Reader, w io.Writer) bool {
	return dat.TransduceTokenWriter(r, NewTokenWriter(w, SIMPLE))
}

// TransduceTokenWriter transduces an input string against
// the double array FSA. The rules are always greedy. If the
// automaton fails, it takes the last possible token ending
// branch.
//
// Based on Mizobuchi et al (2000), p. 129,
// with additional support for IDENTITY, UNKNOWN
// and EPSILON transitions and NONTOKEN and TOKENEND handling.
func (dat *DaTokenizer) TransduceTokenWriter(r io.Reader, w *TokenWriter) bool {
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

PARSECHAR:
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
					eof = true
					break
				}
				buffer[buffi] = char
				buffi++
			}

			char = buffer[buffc]

			if DEBUG {
				log.Println("Current char", string(char), int(char), showBufferNew(buffer, bufft, buffc, buffi))
			}

			eot = false

			// TODO:
			//   Better not repeatedly check for a!
			//   Possibly keep a buffer with a.
			if int(char) < 256 {
				eot = int(char) == EOT
				a = dat.sigmaASCII[int(char)]
			} else {
				a, ok = dat.sigma[char]

				// Use identity symbol if character is not in sigma
				if !ok && dat.identity != -1 {
					a = dat.identity
				}
			}

			t0 = t

			// Check for epsilon transitions and remember
			if dat.array[dat.array[t0].getBase()+uint32(dat.epsilon)].getCheck() == t0 {

				// Remember state for backtracking to last tokenend state
				epsilonState = t0
				epsilonOffset = buffc

				if DEBUG {
					log.Println("epsilonOffset is set to", buffc)
				}
			}
		}

		// Checks a transition based on t0, a and buffo
		t = dat.array[t0].getBase() + uint32(a)
		ta := dat.array[t]

		if DEBUG {
			// Char is only relevant if set
			log.Println("Check", t0, "-", a, "(", string(char), ")", "->", t)
			if false {
				log.Println(dat.outgoing(t0))
			}
		}

		// Check if the transition is invalid according to the double array
		if t > dat.array[1].getCheck() || ta.getCheck() != t0 {

			if DEBUG {
				log.Println("Match is not fine!", t, "and", ta.getCheck(), "vs", t0)
			}

			if !ok && a == dat.identity {

				// Try again with unknown symbol, in case identity failed
				// Char is only relevant when set
				if DEBUG {
					log.Println("UNKNOWN symbol", string(char), "->", dat.unknown)
				}
				a = dat.unknown

			} else if a != dat.epsilon && epsilonState != 0 {

				// Try again with epsilon symbol, in case everything else failed
				t0 = epsilonState
				epsilonState = 0 // reset
				buffc = epsilonOffset
				a = dat.epsilon

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
				// Hopefully this is branchless code
				if buffc-bufft == 0 {
					buffc++
				}

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

				a = dat.epsilon

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

		// Transition consumes a character
		if a != dat.epsilon {

			buffc++

			// Transition does not produce a character
			// Hopefully this is branchless
			if buffc-bufft == 1 && ta.isNonToken() {
				if DEBUG {
					log.Println("Nontoken forward", showBufferNew(buffer, bufft, buffc, buffi))
				}
				bufft++
				// rewindBuffer = true
			}

		} else {

			// Transition marks the end of a token - so flush the buffer
			if buffc-bufft > 0 {
				if DEBUG {
					log.Println("-> Flush buffer: [", string(buffer[bufft:buffc]), "]", showBuffer(buffer, buffc, buffi))
				}
				w.Token(bufft, buffer[:buffc])
				rewindBuffer = true
				sentenceEnd = false
				textEnd = false
			} else {
				sentenceEnd = true
				w.SentenceEnd(0)
			}
		}

		if eot {
			eot = false
			textEnd = true
			w.TextEnd(0)
			if DEBUG {
				log.Println("END OF TEXT")
			}
		}

		// Rewind the buffer if necessary
		if rewindBuffer {

			if DEBUG {
				log.Println("-> Rewind buffer", bufft, buffc, buffi, epsilonOffset)
			}

			// TODO: Better as a ring buffer
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

		// Move to representative state
		if ta.isSeparate() {
			t = ta.getBase()
			ta = dat.array[t]

			if DEBUG {
				log.Println("Representative pointing to", t)
			}
		}

		newchar = true

		// TODO:
		//   Prevent endless epsilon loops by checking
		//   the model has no epsilon loops1
	}

	// Input reader is not yet finished
	if !eof {
		if DEBUG {
			log.Println("Not at the end - problem", t0, ":", dat.outgoing(t0))
		}
		// This should never happen
		return false
	}

	if DEBUG {
		log.Println("Entering final check")
	}

	// Check epsilon transitions as long as possible
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
		buffc = epsilonOffset
		if DEBUG {
			log.Println("Get from epsilon stack and set buffo!", showBufferNew(buffer, bufft, buffc, buffi))
		}
		goto PARSECHAR
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
		w.SentenceEnd(0)

		if DEBUG {
			log.Println("Sentence end")
		}
	}

	if !textEnd {
		w.TextEnd(0)

		if DEBUG {
			log.Println("Text end")
		}
	}

	return true
}

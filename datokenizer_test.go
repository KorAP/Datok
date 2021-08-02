package datokenizer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleString(t *testing.T) {
	assert := assert.New(t)

	// bau | bauamt
	r := strings.NewReader(`##foma-net 1.0##
##props##
1 6 7 8 2 2 1 1 1 1 1 2 5B57D486
##sigma##
0 @_EPSILON_SYMBOL_@
3 a
4 b
5 m
6 t
7 u
##states##
0 4 1 0
1 3 2 0
2 7 3 0
3 3 4 1
4 5 5 0
5 6 6 0
6 -1 -1 1
-1 -1 -1 -1 -1
##end##`)

	tok := parse(r) // ("tokenizer.fst")
	tok.buildDA()
	assert.True(tok.match("bau"))
	assert.True(tok.match("bauamt"))
	assert.False(tok.match("baum"))
}

package topbar_test

import (
	"strings"
	"testing"

	"github.com/ambientsound/pms/api"
	"github.com/ambientsound/pms/topbar"
	"github.com/stretchr/testify/assert"
)

type result struct {
	class int
	str   string
}

var topbarTests = []struct {
	input   string
	success bool
	width   int
	height  int
}{
	{"plain", true, 1, 1},
	{"plain|white", true, 2, 1},
	{"plain|white|tests;multiple lines", true, 3, 2},
	{";;more;lines|here", true, 2, 4},
	{"$shortname|$version", true, 2, 1},
	{"$bogus_variable", false, 0, 0},
	{"$$var1", false, 0, 0},
	{"{}", false, 0, 0},
	{"# comment", false, 0, 0},
	{"\"quoted\"", true, 1, 1},
}

func TestTopbarCount(t *testing.T) {
	for _, test := range topbarTests {

		a := api.NewTestAPI()

		matrix, err := topbar.Parse(a, test.input)
		if test.success {
			assert.Nil(t, err, "Expected success in topbar parser when parsing '%s'", test.input)
		} else {
			assert.NotNil(t, err, "Expected error in topbar parser when parsing '%s'", test.input)
			continue
		}

		assert.Equal(t, test.height, len(matrix),
			"Topbar input '%s' should yield %d lines, got %d instead", test.input, test.height, len(matrix))

		for y := 0; y < len(matrix); y++ {
			assert.Equal(t, test.width, len(matrix[y]),
				"Topbar input '%s' should yield %d columns on line %d, got %d instead", test.input, test.width, y+1, len(matrix[y]))
		}
	}
}

var fragmentTests = []struct {
	input     string
	success   bool
	statement topbar.FragmentStatement
}{
	// Valid forms
	{`plain`, true, topbar.FragmentStatement{`plain`, ``, ``}},
	{`plain; and more`, true, topbar.FragmentStatement{`plain`, ``, ``}},
	{`     |    `, true, topbar.FragmentStatement{`     `, ``, ``}},
	{`foo;bar`, true, topbar.FragmentStatement{`foo`, ``, ``}},
	{`$var`, true, topbar.FragmentStatement{``, `var`, ``}},
	{`${var}`, true, topbar.FragmentStatement{``, `var`, ``}},
	{`${var|param}`, true, topbar.FragmentStatement{``, `var`, `param`}},
	{`${  var  |  param  }`, true, topbar.FragmentStatement{``, `var`, `param`}},

	// Invalid forms
	{`${var`, false, topbar.FragmentStatement{}},
	{`${var|`, false, topbar.FragmentStatement{}},
	{`${var|param`, false, topbar.FragmentStatement{}},
	{`${var|}`, false, topbar.FragmentStatement{}},
	{`${|`, false, topbar.FragmentStatement{}},
	{`${}`, false, topbar.FragmentStatement{}},
	{`${{`, false, topbar.FragmentStatement{}},
	{`${$`, false, topbar.FragmentStatement{}},
	{`${   }`, false, topbar.FragmentStatement{}},
}

func TestFragments(t *testing.T) {
	for n, test := range fragmentTests {

		reader := strings.NewReader(test.input)
		parser := topbar.NewParser(reader)

		frag, err := parser.ParseFragment()

		t.Logf("### Test %d: '%s'", n+1, test.input)

		if test.success {
			assert.Nil(t, err, "Expected success in topbar parser when parsing '%s'", test.input)
		} else {
			assert.NotNil(t, err, "Expected error in topbar parser when parsing '%s'", test.input)
		}

		if frag != nil {
			assert.Equal(t, test.statement, *frag)
		}
	}
}

var pieceTests = []struct {
	input     string
	success   bool
	fragments int
}{
	// Valid forms
	{`plain`, true, 1},
	{`plain two more`, true, 5},
	{`${complex|form} and more whitespace `, true, 8},
	{`hax | piece`, true, 2},
	{`hax  ; more`, true, 2},
	{`|||||`, true, 0},

	// Invalid form
	{`token plus ${invalid`, false, 0},
}

func TestPieces(t *testing.T) {
	for n, test := range pieceTests {

		reader := strings.NewReader(test.input)
		parser := topbar.NewParser(reader)

		piece, err := parser.ParsePiece()

		t.Logf("### Test %d: '%s'", n+1, test.input)

		if test.success {
			assert.Nil(t, err, "Expected success in topbar parser when parsing '%s'", test.input)
		} else {
			assert.NotNil(t, err, "Expected error in topbar parser when parsing '%s'", test.input)
		}

		if piece != nil {
			assert.Equal(t, test.fragments, len(piece.Fragments))
		}
	}
}

var rowTests = []struct {
	input   string
	success bool
	pieces  int
}{
	// Valid forms
	{`plain`, true, 1},
	{`plain|  two  |more`, true, 3},
	{`${complex |  form}|and |more||||whitespace `, true, 7},
	{`||a`, true, 3},
	{`b||`, true, 2},
	{`||`, true, 2},

	// Invalid form
	{`token|plus|${invalid`, false, 0},
}

func TestRows(t *testing.T) {
	for n, test := range rowTests {

		reader := strings.NewReader(test.input)
		parser := topbar.NewParser(reader)

		row, err := parser.ParseRow()

		t.Logf("### Test %d: '%s'", n+1, test.input)

		if test.success {
			assert.Nil(t, err, "Expected success in topbar parser when parsing '%s'", test.input)
		} else {
			assert.NotNil(t, err, "Expected error in topbar parser when parsing '%s'", test.input)
		}

		if row != nil {
			assert.Equal(t, test.pieces, len(row.Pieces))
		}
	}
}

var matrixTests = []struct {
	input   string
	success bool
	rows    int
}{
	// Valid forms
	{`plain`, true, 1},
	{`plain|with|pieces`, true, 1},
	{`plain;with|pieces;and rows`, true, 3},
	{`;;a`, true, 3},
	{`b;;`, true, 2},
	{`;;`, true, 2},
	{`;||;||;||;`, true, 4},

	// Invalid form
	{`token;plus|${invalid`, false, 0},
}

func TestMatrix(t *testing.T) {
	for n, test := range matrixTests {

		reader := strings.NewReader(test.input)
		parser := topbar.NewParser(reader)

		matrix, err := parser.ParseMatrix()

		t.Logf("### Test %d: '%s'", n+1, test.input)

		if test.success {
			assert.Nil(t, err, "Expected success in topbar parser when parsing '%s'", test.input)
		} else {
			assert.NotNil(t, err, "Expected error in topbar parser when parsing '%s'", test.input)
		}

		if matrix != nil {
			assert.Equal(t, test.rows, len(matrix.Rows))
			t.Logf("%+v", matrix.Rows[0])
		}
	}
}
package format_test

import (
	"go/format"
	"testing"

	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/testify"
)

// TestMain lets production entry points prove every registered format domain.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// STANDARD_FLAT_SOURCE states a file the author wrote with no empty line and no tab, which is the
// form a formatter answers for and no author writes twice.
const STANDARD_FLAT_SOURCE = `package one
// COUNT names one count.
const COUNT = 0XFF
var table [4]int
type Pair struct {
Left int
Right string
}
func Fold(pair Pair) (sum int) {
sum = pair.Left*2 + 3
if sum > 0 {
sum++
}
for step := range 4 {
sum = sum + step
}
switch sum {
case 1:
sum = 2
default:
sum = 1E6
}
return sum
}
`

// STANDARD_SPACED_SOURCE states a file whose author left spaces inside brackets and empty lines
// in runs. Its octal stands in the 0o form and its literal stands spread across lines already,
// because the standard formatter keeps forms this dialect rewrites: the 0765 octal, and a literal
// the author broke behind its first element.
const STANDARD_SPACED_SOURCE = `package two

import "errors"



// Report states one report.
func Report(values []int) (held bool, err error) {
	held = len(values) > 0
	if !held {
		err = errors.New( "empty" )
		return held, err
	}
	total := values[ 0 ] + values[ len(values)-1 ]
	slice := values[1:]
	_ = slice
	numbers := []int{ 0o765, 0O17, 0B1010, 1_000 }
	_ = numbers
	pairs := map[string]int{
		"one": 1,
		"two": 2,
	}
	_ = pairs
	go Report(values)
	defer func() { held = false }()
	return total > 0, err
}
`

// STANDARD_WIDE_SOURCE states the forms a body holds beyond its statements: a constraint, a
// channel of each direction, a label, a type switch, and the signs that clash against a value.
const STANDARD_WIDE_SOURCE = `package three

type Reader interface {
	~int | string
}

type Handler func(one int, two string) (held bool)

var sender chan<- int

var receiver <-chan int

func Wait(values ...int) (sum int) {
	select {
	case value := <-receiver:
		sum = value
	case sender <- sum:
		sum = 0
	}
Loop:
	for range 4 {
		for range 2 {
			break Loop
		}
	}
	switch held := any(sum); held.(type) {
	case int:
		sum = 1
	}
	sum = sum & ^1
	sum = sum + +1
	sum = sum / 2
	return sum
}
`

// Test_Standard_Library_Format states that one format writes the form the standard formatter
// writes, over every source this dialect admits.
func Test_Standard_Library_Format(t *testing.T) {
	for _, one := range []string{
		STANDARD_FLAT_SOURCE, STANDARD_SPACED_SOURCE, STANDARD_WIDE_SOURCE,
	} {
		want, err := format.Source([]byte(one))
		testify.True(t, err == nil, "the standard formatter reads the source")
		testify.Equal(t, string(want), formatted(t, one),
			"one format writes the form the standard formatter writes")
	}
}

// Test_Standard_Library_Clean states that a source the standard formatter leaves alone is the
// source one clean report calls clean, and that every other source is not.
func Test_Standard_Library_Clean(t *testing.T) {
	for _, one := range []string{
		STANDARD_FLAT_SOURCE, STANDARD_SPACED_SOURCE, STANDARD_WIDE_SOURCE,
	} {
		want, err := format.Source([]byte(one))
		testify.True(t, err == nil, "the standard formatter reads the source")
		testify.True(t, bool(cleanliness(t, string(want))),
			"a source the standard formatter leaves alone is a clean source")
		if string(want) == one {
			continue
		}
		testify.False(t, bool(cleanliness(t, one)),
			"a source the standard formatter rewrites is no clean source")
	}
}

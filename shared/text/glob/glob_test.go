package glob_test

import (
	"regexp"
	"testing"

	"local/james-orcales/shared/text/glob"
)

type glob_case struct {
	Should     bool
	Pattern    string
	Fixture    string
	Separators []rune
}

func run_glob_cases(t *testing.T, cases []glob_case) {
	t.Helper()
	for case_index, row := range cases {
		compiled := glob.Must_Compile(row.Pattern, row.Separators...)
		got := glob.Match(compiled, row.Fixture)
		if got != row.Should {
			t.Errorf(
				"#%d Match(%q, %q) = %v, want %v",
				case_index, row.Pattern, row.Fixture, got, row.Should)
		}
	}
}

// Test_Glob_Wildcards checks *, **, ?, plain text, and escaping with no separators.
func Test_Glob_Wildcards(t *testing.T) {
	run_glob_cases(t, []glob_case{
		{Should: true, Pattern: "* ?at * eyes", Fixture: "my cat has very bright eyes"},
		{Should: true, Pattern: "", Fixture: ""},
		{Should: false, Pattern: "", Fixture: "b"},
		{Should: true, Pattern: "*ä", Fixture: "åä"},
		{Should: true, Pattern: "abc", Fixture: "abc"},
		{Should: true, Pattern: "a*c", Fixture: "abc"},
		{Should: true, Pattern: "a*c", Fixture: "a12345c"},
		{Should: true, Pattern: "a?c", Fixture: "a1c"},
		{Should: true, Pattern: "?at", Fixture: "cat"},
		{Should: true, Pattern: "?at", Fixture: "fat"},
		{Should: true, Pattern: "*", Fixture: "abc"},
		{Should: true, Pattern: `\*`, Fixture: "*"},
		{Should: false, Pattern: "?at", Fixture: "at"},
		{Should: true, Pattern: "???", Fixture: "abc"},
		{Should: true, Pattern: "?*?", Fixture: "abc"},
		{Should: true, Pattern: "?*?", Fixture: "ac"},
	})
}

// Test_Glob_Separators checks that a separator bounds * and ? while ** spans it.
func Test_Glob_Separators(t *testing.T) {
	dot := []rune{'.'}
	run_glob_cases(t, []glob_case{
		{Should: true, Pattern: "a.b", Fixture: "a.b", Separators: dot},
		{Should: true, Pattern: "a.*", Fixture: "a.b", Separators: dot},
		{Should: true, Pattern: "a.**", Fixture: "a.b.c", Separators: dot},
		{Should: true, Pattern: "a.?.c", Fixture: "a.b.c", Separators: dot},
		{Should: true, Pattern: "a.?.?", Fixture: "a.b.c", Separators: dot},
		{Should: true, Pattern: "**", Fixture: "a.b.c", Separators: dot},
		{Should: false, Pattern: "?at", Fixture: "fat", Separators: []rune{'f'}},
		{Should: false, Pattern: "a.*", Fixture: "a.b.c", Separators: dot},
		{Should: false, Pattern: "a.?.c", Fixture: "a.bb.c", Separators: dot},
		{Should: false, Pattern: "*", Fixture: "a.b.c", Separators: dot},
	})
}

// Test_Glob_Phrases checks prefix, suffix, and interior wildcards over a phrase.
func Test_Glob_Phrases(t *testing.T) {
	run_glob_cases(t, []glob_case{
		{Should: true, Pattern: "*test", Fixture: "this is a test"},
		{Should: true, Pattern: "this*", Fixture: "this is a test"},
		{Should: true, Pattern: "*is *", Fixture: "this is a test"},
		{Should: true, Pattern: "*is*a*", Fixture: "this is a test"},
		{Should: true, Pattern: "**test**", Fixture: "this is a test"},
		{Should: true, Pattern: "**is**a***test*", Fixture: "this is a test"},
		{Should: false, Pattern: "*is", Fixture: "this is a test"},
		{Should: false, Pattern: "*no*", Fixture: "this is a test"},
		{Should: true, Pattern: "[!a]*", Fixture: "this is a test3"},
		{Should: true, Pattern: "*abc", Fixture: "abcabc"},
		{Should: true, Pattern: "**abc", Fixture: "abcabc"},
		{Should: false, Pattern: "sta", Fixture: "stagnation"},
		{Should: true, Pattern: "sta*", Fixture: "stagnation"},
		{Should: false, Pattern: "sta?", Fixture: "stagnation"},
		{Should: false, Pattern: "sta?n", Fixture: "stagnation"},
	})
}

// Test_Glob_Braces checks {...} alternative groups, including nested combinations.
func Test_Glob_Braces(t *testing.T) {
	run_glob_cases(t, []glob_case{
		{Should: true, Pattern: "{abc,def}ghi", Fixture: "defghi"},
		{Should: true, Pattern: "{abc,abcd}a", Fixture: "abcda"},
		{Should: true, Pattern: "{a,ab}{bc,f}", Fixture: "abc"},
		{Should: true, Pattern: "{*,**}{a,b}", Fixture: "ab"},
		{Should: false, Pattern: "{*,**}{a,b}", Fixture: "ac"},
		{Should: true, Pattern: "/{rate,[a-z][a-z][a-z]}*", Fixture: "/rate"},
		{Should: true, Pattern: "/{rate,[0-9][0-9][0-9]}*", Fixture: "/rate"},
		{Should: true, Pattern: "/{rate,[a-z][a-z][a-z]}*", Fixture: "/usd"},
		{Should: true, Pattern: "*//{,*.}example.com", Fixture: "https://www.example.com"},
		{Should: true, Pattern: "*//{,*.}example.com", Fixture: "http://example.com"},
		{Should: false, Pattern: "*//{,*.}example.com", Fixture: "http://example.com.net"},
	})
}

// Test_Glob_Braces_With_Separators checks alternative groups under a separator.
func Test_Glob_Braces_With_Separators(t *testing.T) {
	dot := []rune{'.'}
	both := "{*.google.*,*.yandex.*}"
	tail := "{*.google.*,yandex.*}"
	run_glob_cases(t, []glob_case{
		{Should: true, Pattern: both, Fixture: "www.google.com", Separators: dot},
		{Should: true, Pattern: both, Fixture: "www.yandex.com", Separators: dot},
		{Should: false, Pattern: both, Fixture: "yandex.com", Separators: dot},
		{Should: false, Pattern: both, Fixture: "google.com", Separators: dot},
		{Should: true, Pattern: tail, Fixture: "www.google.com", Separators: dot},
		{Should: true, Pattern: tail, Fixture: "yandex.com", Separators: dot},
		{Should: false, Pattern: tail, Fixture: "www.yandex.com", Separators: dot},
		{Should: false, Pattern: tail, Fixture: "google.com", Separators: dot},
	})
}

// Test_Glob_Complex checks the multi-feature patterns and prefix/suffix shapes.
func Test_Glob_Complex(t *testing.T) {
	all := "[a-z][!a-x]*cat*[h][!b]*eyes*"
	run_glob_cases(t, []glob_case{
		{Should: true, Pattern: all, Fixture: "my cat has very bright eyes"},
		{Should: false, Pattern: all, Fixture: "my dog has very bright eyes"},
		{Should: true, Pattern: "google.com", Fixture: "google.com"},
		{Should: false, Pattern: "google.com", Fixture: "gobwas.com"},
		{Should: false, Pattern: "https://*.google.*", Fixture: "https://google.com"},
		{Should: true, Pattern: "{abc*def,abc?def,abc[zte]def}", Fixture: "abczdef"},
		{
			Should:  true,
			Pattern: "https://*.google.*",
			Fixture: "https://account.google.com",
		},
		{
			Should:  true,
			Pattern: "{abc*[a-c]def,abc?[d-g]def,abc[zte]?def}",
			Fixture: "abczqdef",
		},
		{Should: true, Pattern: "abc*", Fixture: "abcdef"},
		{Should: false, Pattern: "abc*", Fixture: "af"},
		{Should: true, Pattern: "*def", Fixture: "abcdef"},
		{Should: false, Pattern: "*def", Fixture: "af"},
		{Should: true, Pattern: "ab*ef", Fixture: "abcdef"},
		{Should: false, Pattern: "ab*ef", Fixture: "af"},
	})
}

// Test_Glob_Alternatives_Long checks the long alternative-list patterns.
func Test_Glob_Alternatives_Long(t *testing.T) {
	run_glob_cases(t, []glob_case{
		{
			Should:  true,
			Pattern: "{https://*.google.*,*yandex.*,*yahoo.*,*mail.ru}",
			Fixture: "http://yahoo.com",
		},
		{
			Should:  false,
			Pattern: "{https://*.google.*,*yandex.*,*yahoo.*,*mail.ru}",
			Fixture: "http://google.com",
		},
		{
			Should:  true,
			Pattern: "{https://*gobwas.com,http://exclude.gobwas.com}",
			Fixture: "https://safe.gobwas.com",
		},
		{
			Should:  false,
			Pattern: "{https://*gobwas.com,http://exclude.gobwas.com}",
			Fixture: "http://safe.gobwas.com",
		},
		{
			Should:  true,
			Pattern: "{https://*gobwas.com,http://exclude.gobwas.com}",
			Fixture: "http://exclude.gobwas.com",
		},
	})
}

type quote_case struct {
	Input  string
	Output string
}

// Test_Quote_Meta_Table checks Quote_Meta escapes metacharacters and that the
// escaped text compiles to a matcher for the original.
func Test_Quote_Meta_Table(t *testing.T) {
	cases := []quote_case{
		{Input: `[foo*]`, Output: `\[foo\*\]`},
		{Input: `{foo*}`, Output: `\{foo\*\}`},
		{Input: `*?\[]{}`, Output: `\*\?\\\[\]\{\}`},
		{
			Input:  `some text and *?\[]{}`,
			Output: `some text and \*\?\\\[\]\{\}`,
		},
	}
	for case_index, row := range cases {
		got := glob.Quote_Meta(row.Input)
		if got != row.Output {
			t.Errorf(
				"#%d Quote_Meta(%q) = %q, want %q",
				case_index, row.Input, got, row.Output)
		}
		if _, err := glob.Compile(got); err != nil {
			t.Errorf("#%d Compile(%q) errored: %v", case_index, got, err)
		}
	}
}

// Pattern/regexp/fixture constants ported from upstream glob_test.go's const
// block; the individual benchmarks below each reference one family.

const PATTERN_ALL = "[a-z][!a-x]*cat*[h][!b]*eyes*"
const REGEXP_ALL = `^[a-z][^a-x].*cat.*[h][^b].*eyes.*$`
const FIXTURE_ALL_MATCH = "my cat has very bright eyes"
const FIXTURE_ALL_MISMATCH = "my dog has very bright eyes"

const PATTERN_PLAIN = "google.com"
const REGEXP_PLAIN = `^google\.com$`
const FIXTURE_PLAIN_MATCH = "google.com"
const FIXTURE_PLAIN_MISMATCH = "gobwas.com"

const PATTERN_MULTIPLE = "https://*.google.*"
const REGEXP_MULTIPLE = `^https:\/\/.*\.google\..*$`
const FIXTURE_MULTIPLE_MATCH = "https://account.google.com"
const FIXTURE_MULTIPLE_MISMATCH = "https://google.com"

const PATTERN_ALTERNATIVES = "{https://*.google.*,*yandex.*,*yahoo.*,*mail.ru}"
const REGEXP_ALTERNATIVES = `^(https:\/\/.*\.google\..*|.*yandex\..*|.*yahoo\..*|.*mail\.ru)$`
const FIXTURE_ALTERNATIVES_MATCH = "http://yahoo.com"
const FIXTURE_ALTERNATIVES_MISMATCH = "http://google.com"

const PATTERN_ALTERNATIVES_SUFFIX = "{https://*gobwas.com,http://exclude.gobwas.com}"
const REGEXP_ALTERNATIVES_SUFFIX = `^(https:\/\/.*gobwas\.com|http://exclude.gobwas.com)$`
const FIXTURE_ALTERNATIVES_SUFFIX_FIRST_MATCH = "https://safe.gobwas.com"
const FIXTURE_ALTERNATIVES_SUFFIX_FIRST_MISMATCH = "http://safe.gobwas.com"
const FIXTURE_ALTERNATIVES_SUFFIX_SECOND = "http://exclude.gobwas.com"

const PATTERN_PREFIX = "abc*"
const REGEXP_PREFIX = `^abc.*$`
const PATTERN_SUFFIX = "*def"
const REGEXP_SUFFIX = `^.*def$`
const PATTERN_PREFIX_SUFFIX = "ab*ef"
const REGEXP_PREFIX_SUFFIX = `^ab.*ef$`
const FIXTURE_PREFIX_SUFFIX_MATCH = "abcdef"
const FIXTURE_PREFIX_SUFFIX_MISMATCH = "af"

const PATTERN_ALTERNATIVES_COMBINE_LITE = "{abc*def,abc?def,abc[zte]def}"
const REGEXP_ALTERNATIVES_COMBINE_LITE = `^(abc.*def|abc.def|abc[zte]def)$`
const FIXTURE_ALTERNATIVES_COMBINE_LITE = "abczdef"

const PATTERN_ALTERNATIVES_COMBINE_HARD = "{abc*[a-c]def,abc?[d-g]def,abc[zte]?def}"
const REGEXP_ALTERNATIVES_COMBINE_HARD = `^(abc.*[a-c]def|abc.[d-g]def|abc[zte].def)$`
const FIXTURE_ALTERNATIVES_COMBINE_HARD = "abczqdef"

func Benchmark_Parse_Glob(b *testing.B) {
	for b.Loop() {
		glob.Compile(PATTERN_ALL)
	}
}

func Benchmark_Parse_Regexp(b *testing.B) {
	for b.Loop() {
		regexp.MustCompile(REGEXP_ALL)
	}
}

func Benchmark_All_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALL)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALL_MATCH)
	}
}

func Benchmark_All_Glob_Match_Parallel(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALL)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			glob.Match(compiled, FIXTURE_ALL_MATCH)
		}
	})
}

func Benchmark_All_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALL)
	fixture := []byte(FIXTURE_ALL_MATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_All_Glob_Mismatch(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALL)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALL_MISMATCH)
	}
}

func Benchmark_All_Glob_Mismatch_Parallel(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALL)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			glob.Match(compiled, FIXTURE_ALL_MISMATCH)
		}
	})
}

func Benchmark_All_Regexp_Mismatch(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALL)
	fixture := []byte(FIXTURE_ALL_MISMATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Multiple_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_MULTIPLE)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_MULTIPLE_MATCH)
	}
}

func Benchmark_Multiple_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_MULTIPLE)
	fixture := []byte(FIXTURE_MULTIPLE_MATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Multiple_Glob_Mismatch(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_MULTIPLE)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_MULTIPLE_MISMATCH)
	}
}

func Benchmark_Multiple_Regexp_Mismatch(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_MULTIPLE)
	fixture := []byte(FIXTURE_MULTIPLE_MISMATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Alternatives_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALTERNATIVES)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALTERNATIVES_MATCH)
	}
}

func Benchmark_Alternatives_Glob_Mismatch(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALTERNATIVES)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALTERNATIVES_MISMATCH)
	}
}

func Benchmark_Alternatives_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALTERNATIVES)
	fixture := []byte(FIXTURE_ALTERNATIVES_MATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Alternatives_Regexp_Mismatch(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALTERNATIVES)
	fixture := []byte(FIXTURE_ALTERNATIVES_MISMATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Alternatives_Suffix_First_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALTERNATIVES_SUFFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALTERNATIVES_SUFFIX_FIRST_MATCH)
	}
}

func Benchmark_Alternatives_Suffix_First_Glob_Mismatch(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALTERNATIVES_SUFFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALTERNATIVES_SUFFIX_FIRST_MISMATCH)
	}
}

func Benchmark_Alternatives_Suffix_Second_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALTERNATIVES_SUFFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALTERNATIVES_SUFFIX_SECOND)
	}
}

func Benchmark_Alternatives_Combine_Lite_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALTERNATIVES_COMBINE_LITE)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALTERNATIVES_COMBINE_LITE)
	}
}

func Benchmark_Alternatives_Combine_Hard_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_ALTERNATIVES_COMBINE_HARD)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_ALTERNATIVES_COMBINE_HARD)
	}
}

func Benchmark_Alternatives_Suffix_First_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALTERNATIVES_SUFFIX)
	fixture := []byte(FIXTURE_ALTERNATIVES_SUFFIX_FIRST_MATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Alternatives_Suffix_First_Regexp_Mismatch(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALTERNATIVES_SUFFIX)
	fixture := []byte(FIXTURE_ALTERNATIVES_SUFFIX_FIRST_MISMATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Alternatives_Suffix_Second_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALTERNATIVES_SUFFIX)
	fixture := []byte(FIXTURE_ALTERNATIVES_SUFFIX_SECOND)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Alternatives_Combine_Lite_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALTERNATIVES_COMBINE_LITE)
	fixture := []byte(FIXTURE_ALTERNATIVES_COMBINE_LITE)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Alternatives_Combine_Hard_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_ALTERNATIVES_COMBINE_HARD)
	fixture := []byte(FIXTURE_ALTERNATIVES_COMBINE_HARD)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Plain_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_PLAIN)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_PLAIN_MATCH)
	}
}

func Benchmark_Plain_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_PLAIN)
	fixture := []byte(FIXTURE_PLAIN_MATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Plain_Glob_Mismatch(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_PLAIN)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_PLAIN_MISMATCH)
	}
}

func Benchmark_Plain_Regexp_Mismatch(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_PLAIN)
	fixture := []byte(FIXTURE_PLAIN_MISMATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Prefix_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_PREFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_PREFIX_SUFFIX_MATCH)
	}
}

func Benchmark_Prefix_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_PREFIX)
	fixture := []byte(FIXTURE_PREFIX_SUFFIX_MATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Prefix_Glob_Mismatch(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_PREFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_PREFIX_SUFFIX_MISMATCH)
	}
}

func Benchmark_Prefix_Regexp_Mismatch(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_PREFIX)
	fixture := []byte(FIXTURE_PREFIX_SUFFIX_MISMATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Suffix_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_SUFFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_PREFIX_SUFFIX_MATCH)
	}
}

func Benchmark_Suffix_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_SUFFIX)
	fixture := []byte(FIXTURE_PREFIX_SUFFIX_MATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Suffix_Glob_Mismatch(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_SUFFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_PREFIX_SUFFIX_MISMATCH)
	}
}

func Benchmark_Suffix_Regexp_Mismatch(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_SUFFIX)
	fixture := []byte(FIXTURE_PREFIX_SUFFIX_MISMATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Prefix_Suffix_Glob_Match(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_PREFIX_SUFFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_PREFIX_SUFFIX_MATCH)
	}
}

func Benchmark_Prefix_Suffix_Regexp_Match(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_PREFIX_SUFFIX)
	fixture := []byte(FIXTURE_PREFIX_SUFFIX_MATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

func Benchmark_Prefix_Suffix_Glob_Mismatch(b *testing.B) {
	compiled := glob.Must_Compile(PATTERN_PREFIX_SUFFIX)
	for b.Loop() {
		glob.Match(compiled, FIXTURE_PREFIX_SUFFIX_MISMATCH)
	}
}

func Benchmark_Prefix_Suffix_Regexp_Mismatch(b *testing.B) {
	expression := regexp.MustCompile(REGEXP_PREFIX_SUFFIX)
	fixture := []byte(FIXTURE_PREFIX_SUFFIX_MISMATCH)
	for b.Loop() {
		expression.Match(fixture)
	}
}

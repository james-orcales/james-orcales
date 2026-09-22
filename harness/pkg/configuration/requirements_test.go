package configuration_test

import "testing"

import "local/james-orcales/harness/pkg/configuration"
import "local/james-orcales/shared/testify"

// Test_Parse_Shape verifies an object of the seven keys parses whatever whitespace stands between
// its tokens, and a key whose value is of another type states the byte where that value opens.
func Test_Parse_Shape(t *testing.T) {
  subject := configuration_new()
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, MINIMAL).Kind)
  spaced := "\n{ \"shared_component\" :\t\"s\" ,\r\n \"instrumentation_packages\" : [ ] , " +
    "\"ignore\":[\n\n \"a\" , \"b\" ]," +
    "\"opt_out_assertion_mandate_packages\":[],\"opt_out_recursion_ban\":[]," +
    "\"opt_out_interface_ban\":[],\"word_replacements\":{ \"a\" : [ ] , \"b\":[\"c\"] } }\n"
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, spaced).Kind)
  for _, shape := range []Shape_Fault{
    {Source: `{"shared_component":1}`, Kind: configuration.FAULT_STRING, Offset: 20},
    {Source: `{"ignore":"x"}`, Kind: configuration.FAULT_ARRAY, Offset: 10},
    {Source: `{"ignore":[1]}`, Kind: configuration.FAULT_STRING, Offset: 11},
    {Source: `{"word_replacements":["a"]}`, Kind: configuration.FAULT_OBJECT, Offset: 21},
    {Source: `{"word_replacements":{"a":"b"}}`, Kind: configuration.FAULT_ARRAY, Offset: 26},
    {Source: `["a"]`, Kind: configuration.FAULT_OBJECT, Offset: 0},
  } {
    fault := parse(subject, shape.Source)
    testify.Equal(t, shape.Kind, fault.Kind, shape.Source)
    testify.Equal(t, configuration.Fault_Offset(shape.Offset), fault.Offset, shape.Source)
  }
}

// Test_Parse_Values verifies each string decodes into one value naming its key, a replacement
// names its word, and each escape decodes as JSON states it.
func Test_Parse_Values(t *testing.T) {
  subject := configuration_new()
  source := `{"shared_component":"shared","instrumentation_packages":["a/**"],` +
    `"ignore":["c","d"],` +
    `"opt_out_assertion_mandate_packages":["**","!e/**"],"opt_out_recursion_ban":[],` +
    `"opt_out_interface_ban":["f/"],` +
    `"word_replacements":{"len":[],"abs":["absolute"],"act":["action","active"]}}`
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, source).Kind)
  testify.Equal(
    t,
    []string{
      "shared_component=shared",
      "instrumentation_packages=a/**",
      "ignore=c",
      "ignore=d",
      "opt_out_assertion_mandate_packages=**",
      "opt_out_assertion_mandate_packages=!e/**",
      "opt_out_interface_ban=f/",
      "word_replacements=len",
      "word_replacements=abs",
      "word_replacements=abs:absolute",
      "word_replacements=act",
      "word_replacements=act:action",
      "word_replacements=act:active",
    },
    rendered(subject),
  )
  escaped := `"a\"\\\/\b\f\n\r\t` + BACKSLASH + `u00e9` + BACKSLASH + `ud83d` + BACKSLASH +
    `ude00"`
  source = MINIMAL[:len(`{"shared_component":`)] +
    escaped +
    MINIMAL[len(`{"shared_component":"s"`):]
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, source).Kind)
  decoded := "shared_component=a\"\\/\b\f\n\r\t\xc3\xa9\xf0\x9f\x98\x80"
  testify.Equal(t, decoded, rendered(subject)[0])
}

// Test_Parse_Globs verifies each entry lint refuses as a glob is a fault at its opening quote,
// naming its key, in lint's order of lists, and a list of negations alone is a fault at its first
// entry.
func Test_Parse_Globs(t *testing.T) {
  subject := configuration_new()
  for _, entries := range []string{`["**","/a/","a/*.go","!b","a\\*","a\\\\"]`, `[]`} {
    source := glob_list(configuration.KEY_IGNORE, entries)
    testify.Equal(t, configuration.FAULT_NONE, parse(subject, source).Kind, entries)
  }
  for _, glob := range []Glob_Fault{
    {Entries: `[" "]`, Needle: `" "`, Kind: configuration.FAULT_GLOB_EMPTY},
    {Entries: `["a",""]`, Needle: `""`, Kind: configuration.FAULT_GLOB_EMPTY},
    {
      Entries: `["` + BACKSLASH + `u2003"]`,
      Needle:  `"` + BACKSLASH,
      Kind:    configuration.FAULT_GLOB_EMPTY,
    },
    {Entries: `["! "]`, Needle: `"!`, Kind: configuration.FAULT_GLOB_NEGATION},
    {Entries: `["a?"]`, Needle: `"a`, Kind: configuration.FAULT_GLOB_TOKEN},
    {Entries: `["!a[b"]`, Needle: `"!`, Kind: configuration.FAULT_GLOB_TOKEN},
    {Entries: `["a\\"]`, Needle: `"a`, Kind: configuration.FAULT_GLOB_ESCAPE},
    {Entries: `["a\\/b"]`, Needle: `"a`, Kind: configuration.FAULT_GLOB_ESCAPE},
    {Entries: `["!a","!b"]`, Needle: `"!a`, Kind: configuration.FAULT_GLOB_NEGATED},
  } {
    source := glob_list(configuration.KEY_IGNORE, glob.Entries)
    fault := parse(subject, source)
    testify.Equal(t, glob.Kind, fault.Kind, glob.Entries)
    testify.Equal(t, configuration.KEY_IGNORE, fault.Key, glob.Entries)
    testify.Equal(t, configuration.Fault_Offset(offset_of(source, glob.Needle)), fault.Offset)
  }
  // Lint checks the ignore list before the instrumentation packages, whatever order the file
  // states them in.
  source := glob_list(configuration.KEY_INSTRUMENTATION_PACKAGES, `["?"]`)
  source = strings_replace(source, `"ignore":[]`, `"ignore":["?"]`)
  testify.Equal(t, configuration.KEY_IGNORE, parse(subject, source).Key)
  for _, kind := range []configuration.Fault_Kind{
    configuration.FAULT_KEY_ABSENT,
    configuration.FAULT_GLOB_EMPTY,
    configuration.FAULT_GLOB_NEGATED,
  } {
    testify.True(t, bool(configuration.Fault_Keyed(configuration.Recorded_Fault(kind))))
  }
  testify.False(
    t,
    bool(configuration.Fault_Keyed(configuration.Recorded_Fault(configuration.FAULT_SURPLUS))),
  )
}

// Test_Parse_Exemptions verifies an assertion opt-out equal to an instrumentation entry is a fault
// at the opt-out, and one a broader glob covers is not.
func Test_Parse_Exemptions(t *testing.T) {
  subject := configuration_new()
  source := glob_list(configuration.KEY_INSTRUMENTATION_PACKAGES, `["a/**","b/"]`)
  covered := glob_list(configuration.KEY_ASSERTION_EXEMPT, `["**"]`)
  covered = strings_replace(
    covered,
    `"instrumentation_packages":[]`,
    `"instrumentation_packages":["a/**"]`,
  )
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, covered).Kind)
  source = strings_replace(
    source,
    `"opt_out_assertion_mandate_packages":[]`,
    `"opt_out_assertion_mandate_packages":["c/","b/"]`,
  )
  fault := parse(subject, source)
  testify.Equal(t, configuration.FAULT_ASSERTION_REPEATED, fault.Kind)
  testify.Equal(t, configuration.KEY_ASSERTION_EXEMPT, fault.Key)
  exemption := offset_of(source, `"c/","b/"]`) + len(`"c/",`)
  testify.Equal(t, configuration.Fault_Offset(exemption), fault.Offset)
  text := configuration.Fault_Text(configuration.Recorded_Fault(fault.Kind))
  testify.Equal(t, configuration.Fault_Message(configuration.MESSAGE_ASSERTION_REPEATED), text)
}

// Test_Parse_Order verifies Entry_Next yields each path list entry once, in lint's order of lists,
// whatever order the file states them in, and nothing past the last.
func Test_Parse_Order(t *testing.T) {
  subject := configuration_new()
  source := glob_list(configuration.KEY_INTERFACE_EXEMPT, `["e/"]`)
  source = strings_replace(source, `"ignore":[]`, `"ignore":["a/","b/"]`)
  source = strings_replace(
    source,
    `"instrumentation_packages":[]`,
    `"instrumentation_packages":["c/"]`,
  )
  source = strings_replace(source, `"opt_out_recursion_ban":[]`, `"opt_out_recursion_ban":["d/"]`)
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, source).Kind)
  entries := []string{}
  for bool(configuration.Entry_Next(subject)) {
    value := subject.Values[subject.Cells[configuration.CELL_ENTRY]]
    entries = append(entries, string(subject.Text[value.Start:value.End]))
  }
  testify.Equal(t, []string{"a/", "b/", "c/", "d/", "e/"}, entries)
  testify.False(t, bool(configuration.Entry_Next(subject)))
}

// Test_Parse_Faults verifies each fault states itself at the byte it stands on, the end of the
// source for an unfinished value, and an absent key or an empty value at the brace that closes
// the object.
func Test_Parse_Faults(t *testing.T) {
  subject := configuration_new()
  opening := `{"shared_component":`
  for _, shape := range []Shape_Fault{
    {Source: ``, Kind: configuration.FAULT_UNFINISHED, Offset: 0},
    {Source: `{`, Kind: configuration.FAULT_UNFINISHED, Offset: 1},
    {Source: `{"x":[]}`, Kind: configuration.FAULT_KEY_UNKNOWN, Offset: 1},
    {Source: `{"shared_component" "s"}`, Kind: configuration.FAULT_COLON, Offset: 20},
    {Source: opening + `"s""`, Kind: configuration.FAULT_COMMA, Offset: 23},
    {Source: opening + `"s"`, Kind: configuration.FAULT_UNFINISHED, Offset: 23},
    {Source: opening + `"s`, Kind: configuration.FAULT_UNFINISHED, Offset: 22},
    {
      Source: opening + `"s","shared_component":"t"}`,
      Kind:   configuration.FAULT_KEY_REPEATED,
      Offset: 24,
    },
    {Source: opening + "\"\x01\"}", Kind: configuration.FAULT_CONTROL, Offset: 21},
    {Source: opening + `"\q"}`, Kind: configuration.FAULT_ESCAPE, Offset: 21},
    {Source: opening + `"\u00g0"}`, Kind: configuration.FAULT_ESCAPE, Offset: 21},
    {Source: opening + `"\u00`, Kind: configuration.FAULT_ESCAPE, Offset: 21},
    {Source: opening + `"\ud800"}`, Kind: configuration.FAULT_ESCAPE, Offset: 21},
    {
      Source: opening + `"` + BACKSLASH + `ud800` + BACKSLASH + `u0041"}`,
      Kind:   configuration.FAULT_ESCAPE,
      Offset: 21,
    },
    {Source: opening + `"\udc00"}`, Kind: configuration.FAULT_ESCAPE, Offset: 21},
    {Source: opening + "\"\xff\"}", Kind: configuration.FAULT_ENCODING, Offset: 21},
    {Source: opening + "\"\xe2\x82\"}", Kind: configuration.FAULT_ENCODING, Offset: 21},
    {Source: MINIMAL + ` x`, Kind: configuration.FAULT_SURPLUS, Offset: len(MINIMAL) + 1},
  } {
    fault := parse(subject, shape.Source)
    testify.Equal(t, shape.Kind, fault.Kind, shape.Source)
    testify.Equal(t, configuration.Fault_Offset(shape.Offset), fault.Offset, shape.Source)
  }
  fault := parse(subject, `{}`)
  testify.Equal(t, configuration.FAULT_KEY_ABSENT, fault.Kind)
  testify.Equal(t, configuration.KEY_SHARED_COMPONENT, fault.Key)
  testify.Equal(t, configuration.Fault_Offset(1), fault.Offset)
  text := configuration.Fault_Text(configuration.Recorded_Fault(fault.Kind))
  testify.Equal(t, configuration.Fault_Message("is absent"), text)
  fault = parse(subject, `{"shared_component":"s"}`)
  testify.Equal(t, configuration.KEY_INSTRUMENTATION_PACKAGES, fault.Key)
  empty := `{"shared_component":""` + MINIMAL[len(`{"shared_component":"s"`):]
  fault = parse(subject, empty)
  testify.Equal(t, configuration.FAULT_COMPONENT_EMPTY, fault.Kind)
  testify.Equal(t, configuration.Fault_Offset(len(empty)-1), fault.Offset)
  words := MINIMAL[:len(MINIMAL)-len(`{"a":[]}}`)] + `{}}`
  fault = parse(subject, words)
  testify.Equal(t, configuration.FAULT_WORDS_EMPTY, fault.Kind)
  testify.Equal(t, configuration.Fault_Offset(len(words)-1), fault.Offset)
}

// Test_Parse_Bounds verifies a source past its bound and a string past the value table are
// faults, and a table filled to its last slot is not.
func Test_Parse_Bounds(t *testing.T) {
  subject := configuration_new()
  wide := make([]byte, configuration.SOURCE_SIZE_MAXIMUM+1)
  fault := configuration.Parse(subject, configuration.Source(wide))
  testify.Equal(t, configuration.FAULT_OVERSIZED, fault.Kind)
  testify.Equal(t, configuration.Fault_Offset(configuration.SOURCE_SIZE_MAXIMUM), fault.Offset)
  count := configuration.VALUE_COUNT_MAXIMUM
  full := `{"ignore":[` + repeat(`"",`, count) + `""]}`
  fault = parse(subject, full)
  testify.Equal(t, configuration.FAULT_FULL, fault.Kind)
  testify.Equal(t, configuration.Fault_Offset(len(`{"ignore":[`)+3*count), fault.Offset)
  // The component and one word take two slots, thus the ignore list takes the rest.
  filled := `{"ignore":[` + repeat(`"a",`, count-3) + `"a"],` + replace_ignore(MINIMAL[len(`{`):])
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, filled).Kind)
  testify.Equal(t, configuration.Cell(count), subject.Cells[configuration.CELL_VALUE])
}

// BACKSLASH opens an escape, spelled apart so no escape stands in this source.
const BACKSLASH = `\`

// MINIMAL is the smallest configuration Parse takes.
const MINIMAL = `{"shared_component":"s","instrumentation_packages":[],` +
  `"opt_out_assertion_mandate_packages":[],` +
  `"opt_out_recursion_ban":[],"opt_out_interface_ban":[],"ignore":[],"word_replacements":{"a":[]}}`

// Shape_Fault is one source and the fault it states.
type Shape_Fault struct {
  Source string
  Kind   configuration.Fault_Kind
  Offset int
}

// Builds the stores Parse writes into.
func configuration_new() (subject configuration.Configuration_Handle) {
  return &configuration.Configuration{
    Source: make(configuration.Source_Store, configuration.SOURCE_STORAGE_SIZE),
    Text:   make(configuration.Text, configuration.SOURCE_SIZE_MAXIMUM),
    Values: make(configuration.Values, configuration.VALUE_COUNT_MAXIMUM),
    Cells:  make(configuration.Cells, configuration.CELL_COUNT),
  }
}

// Parses one source.
func parse(subject configuration.Configuration_Handle, source string) (fault configuration.Fault) {
  return configuration.Parse(subject, configuration.Source(source))
}

// States each value as its key and text, a replacement past its word.
func rendered(subject configuration.Configuration_Handle) (lines []string) {
  for _, value := range subject.Values[:subject.Cells[configuration.CELL_VALUE]] {
    line := string(configuration.Key_Text(value.Key)) + "="
    if value.Word != configuration.WORD_NONE {
      word := subject.Values[value.Word]
      line += string(subject.Text[word.Start:word.End]) + ":"
    }
    lines = append(lines, line+string(subject.Text[value.Start:value.End]))
  }
  return lines
}

// Repeats one text.
func repeat(text string, count int) (repeated string) {
  for range count {
    repeated += text
  }
  return repeated
}

// Drops the empty ignore list of a configuration, which the caller states first.
func replace_ignore(rest string) (dropped string) {
  marker := `"ignore":[],`
  for index := 0; index+len(marker) < len(rest)+1; index++ {
    if rest[index:index+len(marker)] == marker {
      return rest[:index] + rest[index+len(marker):]
    }
  }
  return rest
}

// Glob_Fault is one list of entries, the text the faulty entry opens with, and the fault it states.
type Glob_Fault struct {
  Entries string
  Needle  string
  Kind    configuration.Fault_Kind
}

// The smallest configuration with one list replaced by the entries.
func glob_list(key configuration.Key, entries string) (source string) {
  name := string(configuration.Key_Text(key))
  return strings_replace(MINIMAL, "\""+name+"\":[]", "\""+name+"\":"+entries)
}

// Replaces the first text in the source with another.
func strings_replace(source string, old string, new string) (replaced string) {
  index := offset_of(source, old)
  if index < 0 {
    return source
  }
  return source[:index] + new + source[index+len(old):]
}

// States where the needle first stands in the source, or -1.
func offset_of(source string, needle string) (offset int) {
  for index := 0; index+len(needle) < len(source)+1; index++ {
    if source[index:index+len(needle)] == needle {
      return index
    }
  }
  return -1
}

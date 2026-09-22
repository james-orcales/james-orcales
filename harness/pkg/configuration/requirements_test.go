package configuration_test

import "testing"

import "local/james-orcales/harness/pkg/configuration"
import "local/james-orcales/shared/encoding/json"
import "local/james-orcales/shared/testify"

// Test_Parse_Shape verifies an object of the seven keys parses whatever whitespace stands between
// its tokens, and a key whose value is of another type states that key.
func Test_Parse_Shape(t *testing.T) {
  subject := configuration_new()
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, MINIMAL).Kind)
  spaced := "\n{ \"shared_component\" :\t\"s\" ,\r\n \"instrumentation_packages\" : [ ] , " +
    "\"ignore\":[\n\n \"a\" , \"b\" ]," +
    "\"opt_out_assertion_mandate_packages\":[],\"opt_out_recursion_ban\":[]," +
    "\"opt_out_interface_ban\":[],\"word_replacements\":{ \"a\" : [ ] , \"b\":[\"c\"] } }\n"
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, spaced).Kind)
  for _, shape := range []Shape_Fault{
    {Source: `{"shared_component":1}`, Key: configuration.KEY_SHARED_COMPONENT},
    {Source: `{"shared_component":["s"]}`, Key: configuration.KEY_SHARED_COMPONENT},
    {Source: `{"ignore":"x"}`, Key: configuration.KEY_IGNORE},
    {Source: `{"ignore":[1]}`, Key: configuration.KEY_IGNORE},
    {Source: `{"ignore":[[]]}`, Key: configuration.KEY_IGNORE},
    {Source: `{"word_replacements":["a"]}`, Key: configuration.KEY_WORD_REPLACEMENTS},
    {Source: `{"word_replacements":{"a":"b"}}`, Key: configuration.KEY_WORD_REPLACEMENTS},
  } {
    fault := parse(subject, shape.Source)
    testify.Equal(t, configuration.FAULT_TYPE, fault.Kind, shape.Source)
    testify.Equal(t, shape.Key, fault.Key, shape.Source)
  }
  testify.Equal(t, configuration.FAULT_OBJECT, parse(subject, `["a"]`).Kind)
}

// Test_Parse_Values verifies each string decodes into one value naming its key, a replacement
// names its word, and each string decodes as shared/encoding/json decodes it.
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
  escaped := `"a\"\\\/\b\f\n\r\t` +
    BACKSLASH +
    `u00e9` +
    BACKSLASH +
    `ud83d` +
    BACKSLASH +
    `ude00` +
    BACKSLASH +
    "ud800\xff\""
  source = MINIMAL[:len(`{"shared_component":`)] +
    escaped +
    MINIMAL[len(`{"shared_component":"s"`):]
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, source).Kind)
  decoded := "shared_component=a\"\\/\b\f\n\r\t\xc3\xa9\xf0\x9f\x98\x80��"
  testify.Equal(t, decoded, rendered(subject)[0])
}

// Test_Parse_Globs verifies each entry lint refuses as a glob is a fault naming its key and the
// entry, in lint's order of lists, and a list of negations alone is a fault naming its key.
func Test_Parse_Globs(t *testing.T) {
  subject := configuration_new()
  for _, entries := range []string{`["**","/a/","a/*.go","!b","a\\*","a\\\\"]`, `[]`} {
    source := glob_list(configuration.KEY_IGNORE, entries)
    testify.Equal(t, configuration.FAULT_NONE, parse(subject, source).Kind, entries)
  }
  for _, glob := range []Glob_Fault{
    {Entries: `[" "]`, Entry: " ", Kind: configuration.FAULT_GLOB_EMPTY},
    {Entries: `["a",""]`, Entry: "", Kind: configuration.FAULT_GLOB_EMPTY},
    {Entries: `["` + BACKSLASH + `u2003"]`, Entry: " ", Kind: configuration.FAULT_GLOB_EMPTY},
    {Entries: `["! "]`, Entry: "! ", Kind: configuration.FAULT_GLOB_NEGATION},
    {Entries: `["a?"]`, Entry: "a?", Kind: configuration.FAULT_GLOB_TOKEN},
    {Entries: `["!a[b"]`, Entry: "!a[b", Kind: configuration.FAULT_GLOB_TOKEN},
    {Entries: `["a\\"]`, Entry: `a\`, Kind: configuration.FAULT_GLOB_ESCAPE},
    {Entries: `["a\\/b"]`, Entry: `a\/b`, Kind: configuration.FAULT_GLOB_ESCAPE},
  } {
    source := glob_list(configuration.KEY_IGNORE, glob.Entries)
    fault := parse(subject, source)
    testify.Equal(t, glob.Kind, fault.Kind, glob.Entries)
    testify.Equal(t, configuration.KEY_IGNORE, fault.Key, glob.Entries)
    testify.Equal(t, glob.Entry, quote(subject), glob.Entries)
  }
  negated := parse(subject, glob_list(configuration.KEY_IGNORE, `["!a","!b"]`))
  testify.Equal(t, configuration.FAULT_GLOB_NEGATED, negated.Kind)
  testify.Equal(t, configuration.KEY_IGNORE, negated.Key)
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
    bool(configuration.Fault_Keyed(configuration.Recorded_Fault(configuration.FAULT_SYNTAX))),
  )
  for _, kind := range []configuration.Fault_Kind{
    configuration.FAULT_KEY_UNKNOWN,
    configuration.FAULT_GLOB_ESCAPE,
    configuration.FAULT_ASSERTION_REPEATED,
  } {
    testify.True(t, bool(configuration.Fault_Quoted(configuration.Recorded_Fault(kind))))
  }
  testify.False(
    t,
    bool(
      configuration.Fault_Quoted(configuration.Recorded_Fault(configuration.FAULT_GLOB_NEGATED)),
    ),
  )
}

// Test_Parse_Exemptions verifies an assertion opt-out equal to an instrumentation entry is a fault
// naming the opt-out, and one a broader glob covers is not.
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
  testify.Equal(t, "b/", quote(subject))
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

// Test_Parse_Faults verifies a source that is no JSON value states the byte shared/encoding/json
// states, an unknown key names itself, a repeated key names its key, and past every member an
// absent key or an empty value names its key.
func Test_Parse_Faults(t *testing.T) {
  subject := configuration_new()
  opening := `{"shared_component":`
  for _, source := range []string{
    ``,
    `{`,
    `{"shared_component" "s"}`,
    opening + `"s""`,
    opening + "\"\x01\"}",
    opening + `"\q"}`,
    MINIMAL + ` x`,
  } {
    fault := parse(subject, source)
    position, _ := json.Validate(json.Encoded(source))
    testify.Equal(t, configuration.FAULT_SYNTAX, fault.Kind, source)
    testify.Equal(t, configuration.Fault_Offset(position), fault.Offset, source)
  }
  fault := parse(subject, `{"x":[]}`)
  testify.Equal(t, configuration.FAULT_KEY_UNKNOWN, fault.Kind)
  testify.Equal(t, "x", quote(subject))
  fault = parse(subject, opening+`"s","shared_component":"t"}`)
  testify.Equal(t, configuration.FAULT_KEY_REPEATED, fault.Kind)
  testify.Equal(t, configuration.KEY_SHARED_COMPONENT, fault.Key)
  fault = parse(subject, `{}`)
  testify.Equal(t, configuration.FAULT_KEY_ABSENT, fault.Kind)
  testify.Equal(t, configuration.KEY_SHARED_COMPONENT, fault.Key)
  text := configuration.Fault_Text(configuration.Recorded_Fault(fault.Kind))
  testify.Equal(t, configuration.Fault_Message("is absent"), text)
  fault = parse(subject, `{"shared_component":"s"}`)
  testify.Equal(t, configuration.KEY_INSTRUMENTATION_PACKAGES, fault.Key)
  empty := `{"shared_component":""` + MINIMAL[len(`{"shared_component":"s"`):]
  fault = parse(subject, empty)
  testify.Equal(t, configuration.FAULT_EMPTY, fault.Kind)
  testify.Equal(t, configuration.KEY_SHARED_COMPONENT, fault.Key)
  words := MINIMAL[:len(MINIMAL)-len(`{"a":[]}}`)] + `{}}`
  fault = parse(subject, words)
  testify.Equal(t, configuration.FAULT_EMPTY, fault.Kind)
  testify.Equal(t, configuration.KEY_WORD_REPLACEMENTS, fault.Key)
}

// Test_Parse_Bounds verifies a source past its bound and a string past the value table are
// faults, and a table filled to its last slot is not.
func Test_Parse_Bounds(t *testing.T) {
  subject := configuration_new()
  wide := make([]byte, configuration.SOURCE_SIZE_MAXIMUM+1)
  testify.Equal(
    t,
    configuration.FAULT_OVERSIZED,
    configuration.Parse(subject, configuration.Source(wide)).Kind,
  )
  count := configuration.VALUE_COUNT_MAXIMUM
  full := `{"ignore":[` + repeat(`"",`, count) + `""]}`
  testify.Equal(t, configuration.FAULT_FULL, parse(subject, full).Kind)
  // The component and one word take two slots, thus the ignore list takes the rest.
  filled := `{"ignore":[` + repeat(`"a",`, count-3) + `"a"],` + replace_ignore(MINIMAL[len(`{`):])
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, filled).Kind)
  testify.Equal(t, configuration.Cell(count), subject.Cells[configuration.CELL_VALUE])
  // Each word spends three tokens, thus a table of words fills the token table as well.
  words := `{"word_replacements":{` +
    repeat(`"a":[],`, count-2) +
    `"a":[]},` +
    MINIMAL[len(`{`):len(MINIMAL)-
      len(`,"word_replacements":{"a":[]}}`)] +
    `}`
  testify.Equal(t, configuration.FAULT_NONE, parse(subject, words).Kind)
  words = `{"word_replacements":{` + repeat(`"a":[],`, count) + `"a":[]}}`
  testify.Equal(t, configuration.FAULT_FULL, parse(subject, words).Kind)
}

// BACKSLASH opens an escape, spelled apart so no escape stands in this source.
const BACKSLASH = `\`

// MINIMAL is the smallest configuration Parse takes.
const MINIMAL = `{"shared_component":"s","instrumentation_packages":[],` +
  `"opt_out_assertion_mandate_packages":[],` +
  `"opt_out_recursion_ban":[],"opt_out_interface_ban":[],"ignore":[],"word_replacements":{"a":[]}}`

// Shape_Fault is one source and the key its fault names.
type Shape_Fault struct {
  Source string
  Key    configuration.Key
}

// Builds the stores Parse writes into.
func configuration_new() (subject configuration.Configuration_Handle) {
  return &configuration.Configuration{
    Tokens: make(configuration.Tokens, configuration.TOKEN_COUNT_MAXIMUM),
    Text:   make(configuration.Text, configuration.SOURCE_SIZE_MAXIMUM),
    Values: make(configuration.Values, configuration.VALUE_COUNT_MAXIMUM),
    Cells:  make(configuration.Cells, configuration.CELL_COUNT),
  }
}

// Parses one source.
func parse(subject configuration.Configuration_Handle, source string) (fault configuration.Fault) {
  return configuration.Parse(subject, configuration.Source(source))
}

// The text the last fault quotes, which its cells name in the text store.
func quote(subject configuration.Configuration_Handle) (text string) {
  cells := subject.Cells
  return string(
    subject.Text[cells[configuration.CELL_FAULT_START]:cells[configuration.CELL_FAULT_END]],
  )
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

// Glob_Fault is one list of entries, the entry its fault names, and the fault it states.
type Glob_Fault struct {
  Entries string
  Entry   string
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

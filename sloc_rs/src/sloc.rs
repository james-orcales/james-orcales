//! Counts the code, comment, and blank lines of source files. A generic
//! per-line scanner, configured by a `Language`, classifies each physical line;
//! a walker counts a tree across scoped threads; a renderer prints an aligned
//! table or JSON. The dialect bans `mut`, so state threads through return
//! values rather than in-place mutation, and reads use visitor parameters.

use std::collections;
use std::iter;
use std::thread;

// Core data.

/// The line partition of a file or a group of files.
#[derive(Copy, Clone, Default, Debug, PartialEq, Eq)]
pub struct Counts {
    pub code: usize,
    pub comment: usize,
    pub blank: usize,
}

/// Describes how one language's lines are read: the tokens that begin a comment
/// or string, and whether its block comments nest. The scanner is generic over
/// this value, so seeding another language is adding a `Language`.
#[derive(Clone, Default, Debug)]
pub struct Language {
    pub name: String,
    /// Tokens that begin a comment running to end of line.
    pub line_comment: Vec<String>,
    /// Begins a block comment, or empty when the language has none.
    pub block_comment_open: String,
    pub block_comment_close: String,
    /// True when an inner open raises the nesting depth (Rust); false when the
    /// first close ends the comment (Go, C).
    pub block_comment_nests: bool,
    /// Multi-line string delimiters whose bodies are verbatim.
    pub verbatim_strings: Vec<Verbatim_Delimiter>,
    /// Single-line string and character delimiters.
    pub quote_strings: Vec<Quote_Delimiter>,
    /// Enables Lua's leveled long brackets: `[[ … ]]` strings and `--[[ … ]]`
    /// comments.
    pub long_bracket: bool,
    /// Base-name prefixes that mark a test file, like Python's `test_`.
    pub test_prefixes: Vec<String>,
    /// Case-sensitive base-name substrings that mark a test file, like Go's
    /// `_test.`, delimited so `latest.go` does not match.
    pub test_infixes: Vec<String>,
    /// Enables `<<DELIM` heredocs (Shell, Ruby, Perl): the body up to a line
    /// equal to DELIM is code, so a `#` inside it is not a comment.
    pub heredoc: bool,
}

/// A string whose body is taken verbatim and may span lines, so a comment token
/// inside it is inert.
#[derive(Clone, Default, Debug)]
pub struct Verbatim_Delimiter {
    /// Opening delimiter (a backtick or triple quote), or — when `hashable` —
    /// the lead before the hashes (Rust's `r` or `br`).
    pub open: String,
    /// Terminator of a fixed verbatim string, unused when `hashable`.
    pub close: String,
    /// Marks a Rust-style raw string: the lead, then N hashes, then a quote; it
    /// closes only on a quote followed by exactly N hashes.
    pub hashable: bool,
}

/// A single-line string or character literal.
#[derive(Clone, Default, Debug)]
pub struct Quote_Delimiter {
    pub open: String,
    pub close: String,
    /// The byte that escapes the following character, or zero for none.
    pub escape: u8,
    /// Marks a character or rune literal, whose apostrophe must be told apart
    /// from a Rust lifetime tick that opens nothing.
    pub character_like: bool,
}

/// The one open construct that crosses a line, as a `Copy` descriptor: its
/// closer is reconstructed from the language and the leveled count rather than
/// stored as an owned string, so the hot scan state never allocates. Heredocs
/// are carried separately, since their terminator is an arbitrary source word.
#[derive(Copy, Clone, Debug, PartialEq, Eq, Default)]
enum Span {
    /// Not inside any cross-line construct.
    #[default]
    None,
    /// Inside a block comment, at the given nesting depth (≥ 1).
    Block_Comment(usize),
    /// Inside a fixed verbatim string; the index selects its delimiter.
    Verbatim(usize),
    /// Inside a Rust-style raw string; the close is `"` then N `#`.
    Raw_Hash(usize),
    /// Inside a Lua long string; the close is `]` then N `=` then `]`.
    Long_String(usize),
    /// Inside a Lua long-bracket comment; the close has the long-string shape.
    Long_Comment(usize),
}

/// The scanner state that crosses line boundaries. Normal strings and character
/// literals never cross a line, so they are not carried.
#[derive(Clone, Default, Debug)]
struct Carry {
    /// The open construct continuing onto the next line, or `None`.
    pub span: Span,
    /// The line that ends an open heredoc, or empty.
    pub heredoc_terminator: String,
}

/// The partition a single physical line falls into.
#[derive(Copy, Clone, Debug, PartialEq, Eq)]
enum Line_Kind {
    Blank,
    Code,
    Comment,
}

/// Accumulates one line's verdict as the scanner walks it. It is `Copy` and
/// allocation-free: `cursor` is the next byte to inspect, `span` the open
/// construct, and `heredoc` the byte offset where a heredoc opened (its owned
/// terminator is materialized once at line end, never threaded per byte).
#[derive(Copy, Clone, Debug)]
struct Line_Scan {
    pub cursor: usize,
    pub span: Span,
    pub has_code: bool,
    pub has_comment: bool,
    pub heredoc: Option<usize>,
}

/// One file's bytes and the language to read them as.
#[derive(Clone, Debug)]
pub struct Classify_File_Input {
    pub source: Vec<u8>,
    pub language: Language,
}

// Counts helpers.

/// The total physical line count: the three partitions sum to it because every
/// line is counted exactly once.
fn counts_lines(counts: Counts) -> usize {
    counts.code + counts.comment + counts.blank
}

/// Adds one line's verdict to a running partition, returning the new partition.
fn counts_tally(counts: Counts, kind: Line_Kind) -> Counts {
    match kind {
        Line_Kind::Code => Counts { code: counts.code + 1, ..counts },
        Line_Kind::Comment => Counts { comment: counts.comment + 1, ..counts },
        Line_Kind::Blank => Counts { blank: counts.blank + 1, ..counts },
    }
}

/// Sums two partitions.
fn counts_add(into: Counts, more: Counts) -> Counts {
    Counts {
        code: into.code + more.code,
        comment: into.comment + more.comment,
        blank: into.blank + more.blank,
    }
}

// Scanner.

/// Partitions every physical line of the source into code, comment, and blank
/// counts. Each line is counted once, so the three sum to the line count.
pub fn classify_file(input: Classify_File_Input) -> Counts {
    classify_bytes(&input.source, &input.language)
}

/// The counting core over a borrowed language, so a tree walk classifies each
/// file without cloning its language.
fn classify_bytes(source: &[u8], language: &Language) -> Counts {
    let trigger = build_trigger(language);
    // Lines are walked in place rather than materialized into a slice: one pass
    // over millions of lines should not also allocate a header per line. The
    // fold carries the start of the current line, the running counts, and the
    // cross-line scanner state.
    let (final_start, counts, carry) = (0..source.len()).fold(
        (0usize, Counts::default(), Carry::default()),
        |(start, counts, carry), index| {
            if source[index] != b'\n' {
                (start, counts, carry)
            } else {
                let (kind, next) = classify_line(&source[start..index], carry, language, &trigger);
                (index + 1, counts_tally(counts, kind), next)
            }
        },
    );
    // Bytes after the last newline are a final line only when non-empty, so a
    // trailing newline adds no phantom line and the empty file is zero lines.
    if final_start < source.len() {
        let (kind, _carry) = classify_line(&source[final_start..], carry, language, &trigger);
        counts_tally(counts, kind)
    } else {
        counts
    }
}

/// Builds the trigger table: the union of the first byte of every opener the
/// language defines, taken from its fields alone so no language is special-cased.
/// A miss would skip a real opener as if it were code.
fn build_trigger(language: &Language) -> [bool; 256] {
    std::array::from_fn(|byte| is_trigger_byte(byte as u8, language))
}

/// Reports whether a byte can begin one of the language's openers.
fn is_trigger_byte(byte: u8, language: &Language) -> bool {
    language.line_comment.iter().any(|token| token.as_bytes()[0] == byte)
        || (!language.block_comment_open.is_empty()
            && language.block_comment_open.as_bytes()[0] == byte)
        || language.verbatim_strings.iter().any(|delimiter| delimiter.open.as_bytes()[0] == byte)
        || language.quote_strings.iter().any(|delimiter| delimiter.open.as_bytes()[0] == byte)
        || (language.heredoc && byte == b'<')
        || (language.long_bracket && byte == b'[')
}

/// Returns the partition of one line and the carry for the next. The per-byte
/// walk is an iterator over cursor stops — one step per opener or skipped run,
/// like a cursor that jumps — over a `Copy` scan state, so the hot loop neither
/// allocates nor moves an owned string.
fn classify_line(
    line: &[u8],
    carry: Carry,
    language: &Language,
    trigger: &[bool; 256],
) -> (Line_Kind, Carry) {
    // A whitespace-only line is blank regardless of carried state, and neither
    // opens nor closes anything, so the carry passes through unchanged.
    if line_is_blank(line) {
        return (Line_Kind::Blank, carry);
    }
    if !carry.heredoc_terminator.is_empty() {
        return classify_heredoc_line(line, carry);
    }
    let init = Line_Scan { cursor: 0, span: carry.span, has_code: false, has_comment: false, heredoc: None };
    let scan = iter::successors(Some(init), |scan| {
        if scan.cursor < line.len() {
            Some(line_scan_step(*scan, line, scan.cursor, language, trigger))
        } else {
            None
        }
    })
    .last()
    .unwrap_or(init);
    // A heredoc that opened this line recorded only its offset (a `Copy` field),
    // so the owned terminator is materialized once here, not threaded per byte.
    let heredoc_terminator = match scan.heredoc {
        Some(offset) => heredoc_open(line, offset).map(|(terminator, _)| terminator).unwrap_or_default(),
        None => String::new(),
    };
    (line_scan_verdict(&scan), Carry { span: scan.span, heredoc_terminator })
}

/// Reads a line inside a heredoc body: the line is code, and a line equal to the
/// terminator ends the heredoc. Any other carried span passes through.
fn classify_heredoc_line(line: &[u8], carry: Carry) -> (Line_Kind, Carry) {
    if line.trim_ascii() == carry.heredoc_terminator.as_bytes() {
        (Line_Kind::Code, Carry { heredoc_terminator: String::new(), ..carry })
    } else {
        (Line_Kind::Code, carry)
    }
}

/// Consumes the token at the cursor, returning the advanced scan. The
/// carried-span branches are checked before the fresh dispatch, so the fresh
/// path's bulk skip can never run inside a comment, raw string, or long bracket.
fn line_scan_step(
    scan: Line_Scan,
    line: &[u8],
    cursor: usize,
    language: &Language,
    trigger: &[bool; 256],
) -> Line_Scan {
    match scan.span {
        Span::Long_Comment(level) => line_scan_long_comment_body(scan, line, cursor, level),
        Span::Verbatim(_) | Span::Raw_Hash(_) | Span::Long_String(_) => {
            line_scan_string_body(scan, line, cursor, language)
        }
        Span::Block_Comment(depth) => line_scan_block(scan, line, cursor, language, depth),
        Span::None => line_scan_fresh(scan, line, cursor, language, trigger),
    }
}

/// Advances inside a long-bracket comment, where every byte is comment and only
/// the matching leveled closer ends it.
fn line_scan_long_comment_body(scan: Line_Scan, line: &[u8], cursor: usize, level: usize) -> Line_Scan {
    match long_bracket_close_at(line, cursor, level) {
        Some(length) => Line_Scan { cursor: cursor + length, span: Span::None, has_comment: true, ..scan },
        None => Line_Scan { cursor: cursor + 1, has_comment: true, ..scan },
    }
}

/// Advances inside a verbatim string, where every byte is code and only the
/// matching close — reconstructed from the span — ends it.
fn line_scan_string_body(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language) -> Line_Scan {
    let close = match scan.span {
        Span::Verbatim(index) => verbatim_close_at(line, cursor, &language.verbatim_strings[index]),
        Span::Raw_Hash(hashes) => raw_hash_close_at(line, cursor, hashes),
        Span::Long_String(level) => long_bracket_close_at(line, cursor, level),
        _ => None,
    };
    match close {
        Some(length) => Line_Scan { cursor: cursor + length, span: Span::None, has_code: true, ..scan },
        None => Line_Scan { cursor: cursor + 1, has_code: true, ..scan },
    }
}

/// Advances inside a block comment, where every byte is comment and only an open
/// (when nesting) or a close moves the depth.
fn line_scan_block(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language, depth: usize) -> Line_Scan {
    if language.block_comment_nests
        && has_prefix_at(line, cursor, language.block_comment_open.as_bytes())
    {
        return Line_Scan {
            cursor: cursor + language.block_comment_open.len(),
            span: Span::Block_Comment(depth + 1),
            has_comment: true,
            ..scan
        };
    }
    if has_prefix_at(line, cursor, language.block_comment_close.as_bytes()) {
        let span = if depth <= 1 { Span::None } else { Span::Block_Comment(depth - 1) };
        return Line_Scan {
            cursor: cursor + language.block_comment_close.len(),
            span,
            has_comment: true,
            ..scan
        };
    }
    Line_Scan { cursor: cursor + 1, has_comment: true, ..scan }
}

/// Dispatches the token at the cursor when not inside a comment or string.
fn line_scan_fresh(
    scan: Line_Scan,
    line: &[u8],
    cursor: usize,
    language: &Language,
    trigger: &[bool; 256],
) -> Line_Scan {
    let byte = line[cursor];
    // A non-trigger byte cannot begin any opener, so it is either insignificant
    // whitespace or plain code. The first code byte marks the line, after which
    // the run of ordinary bytes is skipped to the next trigger in one jump
    // rather than re-dispatching on each byte.
    if !trigger[byte as usize] {
        if byte_is_space(byte) {
            return Line_Scan { cursor: cursor + 1, ..scan };
        }
        let stop = (cursor + 1..line.len())
            .find(|&index| trigger[line[index] as usize])
            .unwrap_or(line.len());
        return Line_Scan { cursor: stop, has_code: true, ..scan };
    }
    // The scan is `Copy`, so each opener takes it by value and the chain costs
    // nothing until one fires; the order matches Go's dispatch.
    open_block(scan, line, cursor, language)
        .or_else(|| open_long_comment(scan, line, cursor, language))
        .or_else(|| open_line_comment(scan, line, cursor, language))
        .or_else(|| open_long_string(scan, line, cursor, language))
        .or_else(|| open_heredoc(scan, line, cursor, language))
        .or_else(|| open_verbatim(scan, line, cursor, language))
        .or_else(|| open_quote(scan, line, cursor, language))
        // A trigger byte that opened nothing — a lone `/`, a shift `<<`, a
        // division — is just code; advance one byte to re-enter the dispatch.
        .unwrap_or(Line_Scan { cursor: cursor + 1, has_code: true, ..scan })
}

/// Reports whether a block comment opens at the cursor.
fn open_block(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language) -> Option<Line_Scan> {
    if language.block_comment_open.is_empty() {
        return None;
    }
    if !has_prefix_at(line, cursor, language.block_comment_open.as_bytes()) {
        return None;
    }
    Some(Line_Scan {
        cursor: cursor + language.block_comment_open.len(),
        span: Span::Block_Comment(1),
        has_comment: true,
        ..scan
    })
}

/// Reports whether a long-bracket comment — a line-comment token then a long
/// bracket, like `--[[` or `--[=[` — opens at the cursor.
fn open_long_comment(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language) -> Option<Line_Scan> {
    if !language.long_bracket {
        return None;
    }
    language.line_comment.iter().find_map(|token| {
        if !has_prefix_at(line, cursor, token.as_bytes()) {
            return None;
        }
        let (level, opener_size) = long_bracket_open(line, cursor + token.len())?;
        Some(Line_Scan {
            cursor: cursor + token.len() + opener_size,
            span: Span::Long_Comment(level),
            has_comment: true,
            ..scan
        })
    })
}

/// Reports whether a line comment opens at the cursor; it runs to end of line.
fn open_line_comment(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language) -> Option<Line_Scan> {
    if !starts_with_any(line, cursor, &language.line_comment) {
        return None;
    }
    Some(Line_Scan { cursor: line.len(), has_comment: true, ..scan })
}

/// Reports whether a long-bracket string — like `[[` or `[=[` — opens at the
/// cursor.
fn open_long_string(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language) -> Option<Line_Scan> {
    if !language.long_bracket {
        return None;
    }
    let (level, opener_size) = long_bracket_open(line, cursor)?;
    Some(Line_Scan { cursor: cursor + opener_size, span: Span::Long_String(level), has_code: true, ..scan })
}

/// Reports whether a heredoc opens at the cursor, recording only its offset so
/// the terminator is materialized once at line end, not threaded per byte.
fn open_heredoc(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language) -> Option<Line_Scan> {
    if !language.heredoc {
        return None;
    }
    let (_terminator, opener_size) = heredoc_open(line, cursor)?;
    Some(Line_Scan { cursor: cursor + opener_size, has_code: true, heredoc: Some(cursor), ..scan })
}

/// Reports whether a verbatim string opens at the cursor.
fn open_verbatim(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language) -> Option<Line_Scan> {
    let (span, opener_size) = verbatim_open(line, cursor, language)?;
    Some(Line_Scan { cursor: cursor + opener_size, span, has_code: true, ..scan })
}

/// Reports whether a single-line string or character literal opens at the cursor.
fn open_quote(scan: Line_Scan, line: &[u8], cursor: usize, language: &Language) -> Option<Line_Scan> {
    let consumed = quote_open(line, cursor, language)?;
    Some(Line_Scan { cursor: cursor + consumed, has_code: true, ..scan })
}

/// Reads the line's partition from the accumulated scan: code wins a line it
/// shares with a comment, then a comment, else blank.
fn line_scan_verdict(scan: &Line_Scan) -> Line_Kind {
    if scan.has_code {
        Line_Kind::Code
    } else if scan.has_comment {
        Line_Kind::Comment
    } else {
        Line_Kind::Blank
    }
}

/// Reports whether a long bracket — `[` then a run of `=` then `[` — opens at
/// the cursor, returning the level (count of `=`) and the opener's byte length.
fn long_bracket_open(line: &[u8], cursor: usize) -> Option<(usize, usize)> {
    if line.get(cursor) != Some(&b'[') {
        return None;
    }
    let level = (cursor + 1..line.len()).take_while(|&index| line[index] == b'=').count();
    if line.get(cursor + 1 + level) != Some(&b'[') {
        return None;
    }
    Some((level, level + 2))
}

/// Reports the byte length of a leveled long-bracket close — `]` then N `=` then
/// `]` — at the cursor, or `None`.
fn long_bracket_close_at(line: &[u8], cursor: usize, level: usize) -> Option<usize> {
    if line.get(cursor) != Some(&b']') {
        return None;
    }
    if !(cursor + 1..cursor + 1 + level).all(|index| line.get(index) == Some(&b'=')) {
        return None;
    }
    if line.get(cursor + 1 + level) != Some(&b']') {
        return None;
    }
    Some(level + 2)
}

/// Reports the byte length of a Rust raw-string close — `"` then N `#` — at the
/// cursor, or `None`.
fn raw_hash_close_at(line: &[u8], cursor: usize, hashes: usize) -> Option<usize> {
    if line.get(cursor) != Some(&b'"') {
        return None;
    }
    if !(cursor + 1..cursor + 1 + hashes).all(|index| line.get(index) == Some(&b'#')) {
        return None;
    }
    Some(hashes + 1)
}

/// Reports the byte length of a fixed verbatim close at the cursor, or `None`.
fn verbatim_close_at(line: &[u8], cursor: usize, delimiter: &Verbatim_Delimiter) -> Option<usize> {
    if has_prefix_at(line, cursor, delimiter.close.as_bytes()) {
        Some(delimiter.close.len())
    } else {
        None
    }
}

/// Reports whether a heredoc opener begins at the cursor — `<<` then an optional
/// `-` or `~`, optional space, then a quoted word or an uppercase/underscore
/// word — returning its terminator word and the opener's byte length.
fn heredoc_open(line: &[u8], cursor: usize) -> Option<(String, usize)> {
    if !has_prefix_at(line, cursor, b"<<") {
        return None;
    }
    let after_sigil = heredoc_skip_sigil(line, cursor + 2);
    let after_space = heredoc_skip_spaces(line, after_sigil);
    let quoted = after_space < line.len() && heredoc_is_quote(line[after_space]);
    let word_start = if quoted { after_space + 1 } else { after_space };
    let word_end = heredoc_skip_identifier(line, word_start);
    if word_end == word_start {
        return None;
    }
    // The uppercase/underscore rule keeps the `a << b` shift operator from
    // looking like a heredoc.
    if !quoted && !heredoc_word_start(line[word_start]) {
        return None;
    }
    let terminator = String::from_utf8_lossy(&line[word_start..word_end]).into_owned();
    let consumed_end = if quoted && word_end < line.len() { word_end + 1 } else { word_end };
    Some((terminator, consumed_end - cursor))
}

/// Skips an optional `<<-` or `<<~` heredoc sigil.
fn heredoc_skip_sigil(line: &[u8], cursor: usize) -> usize {
    if cursor < line.len() && (line[cursor] == b'-' || line[cursor] == b'~') {
        cursor + 1
    } else {
        cursor
    }
}

/// Skips spaces and tabs between the heredoc operator and its delimiter.
fn heredoc_skip_spaces(line: &[u8], cursor: usize) -> usize {
    (cursor..line.len()).find(|&index| !byte_is_space(line[index])).unwrap_or(line.len())
}

/// Skips a run of identifier bytes.
fn heredoc_skip_identifier(line: &[u8], cursor: usize) -> usize {
    (cursor..line.len()).find(|&index| !byte_is_identifier(line[index])).unwrap_or(line.len())
}

/// Reports whether a byte opens a quoted heredoc delimiter.
fn heredoc_is_quote(character: u8) -> bool {
    matches!(character, b'\'' | b'"' | b'`')
}

/// Reports whether a byte may begin an unquoted heredoc delimiter.
fn heredoc_word_start(character: u8) -> bool {
    character == b'_' || character.is_ascii_uppercase()
}

/// Reports whether a verbatim string begins at the cursor and, if so, the span
/// continuing it and the opener's byte length.
fn verbatim_open(line: &[u8], cursor: usize, language: &Language) -> Option<(Span, usize)> {
    language.verbatim_strings.iter().enumerate().find_map(|(index, delimiter)| {
        if !has_prefix_at(line, cursor, delimiter.open.as_bytes()) {
            return None;
        }
        if verbatim_lead_middle_identifier(line, cursor, delimiter) {
            return None;
        }
        if !delimiter.hashable {
            return Some((Span::Verbatim(index), delimiter.open.len()));
        }
        verbatim_hashable(line, cursor, delimiter)
    })
}

/// Reports whether a letter-led opener sits in the middle of an identifier,
/// where the lead is a name character rather than a string start.
fn verbatim_lead_middle_identifier(
    line: &[u8],
    cursor: usize,
    delimiter: &Verbatim_Delimiter,
) -> bool {
    if !byte_is_identifier(delimiter.open.as_bytes()[0]) {
        return false;
    }
    if cursor == 0 {
        return false;
    }
    byte_is_identifier(line[cursor - 1])
}

/// Matches a Rust-style raw-string opener: the lead, then hashes, then a quote.
/// Without the quote the lead is a raw identifier, not a string.
fn verbatim_hashable(
    line: &[u8],
    cursor: usize,
    delimiter: &Verbatim_Delimiter,
) -> Option<(Span, usize)> {
    let after_open = cursor + delimiter.open.len();
    let hashes = (after_open..line.len()).take_while(|&index| line[index] == b'#').count();
    let quote = after_open + hashes;
    if line.get(quote) != Some(&b'"') {
        return None;
    }
    Some((Span::Raw_Hash(hashes), delimiter.open.len() + hashes + 1))
}

/// Reports whether a single-line string or character literal begins at the
/// cursor and, if so, the byte length it consumes.
fn quote_open(line: &[u8], cursor: usize, language: &Language) -> Option<usize> {
    language.quote_strings.iter().find_map(|delimiter| {
        if !has_prefix_at(line, cursor, delimiter.open.as_bytes()) {
            return None;
        }
        if delimiter.character_like {
            Some(scan_character_or_lifetime(line, cursor))
        } else {
            Some(scan_quoted(line, cursor, delimiter))
        }
    })
}

/// The live cursor and recorded end of a quoted-string scan.
#[derive(Copy, Clone)]
struct Quote_Walk {
    pub pos: usize,
    pub end: usize,
}

/// Returns the byte length of a single-line quoted string starting at its
/// opening delimiter, stopping at the first unescaped close or end of line.
fn scan_quoted(line: &[u8], cursor: usize, delimiter: &Quote_Delimiter) -> usize {
    let start = cursor + delimiter.open.len();
    // An unterminated string runs to end of line, so `end` defaults to the line
    // length and is overwritten only when the close is met.
    let walk = (start..line.len()).fold(Quote_Walk { pos: start, end: line.len() }, |walk, index| {
        if index < walk.pos {
            walk
        } else if quote_escapes_here(line, index, delimiter) {
            Quote_Walk { pos: index + 2, end: walk.end }
        } else if has_prefix_at(line, index, delimiter.close.as_bytes()) {
            Quote_Walk { pos: line.len(), end: index + delimiter.close.len() }
        } else {
            Quote_Walk { pos: index + 1, end: walk.end }
        }
    });
    walk.end - cursor
}

/// Reports whether an escape sequence begins at the scan position.
fn quote_escapes_here(line: &[u8], index: usize, delimiter: &Quote_Delimiter) -> bool {
    delimiter.escape != 0 && line[index] == delimiter.escape
}

/// Returns the byte length of a character or rune literal starting at the
/// apostrophe, or 1 when the apostrophe is a Rust lifetime tick.
fn scan_character_or_lifetime(line: &[u8], cursor: usize) -> usize {
    if character_is_escaped(line, cursor) {
        return scan_escaped_character(line, cursor);
    }
    let simple = simple_character(line, cursor);
    if simple > 0 { simple } else { 1 }
}

/// Reports whether a backslash escape follows the apostrophe.
fn character_is_escaped(line: &[u8], cursor: usize) -> bool {
    cursor + 1 < line.len() && line[cursor + 1] == b'\\'
}

/// Returns the length of an escaped character literal, bounded so a stray
/// apostrophe cannot scan the whole line.
fn scan_escaped_character(line: &[u8], cursor: usize) -> usize {
    let upper = (cursor + 13).min(line.len());
    (cursor + 3..upper)
        .find(|&index| line[index] == b'\'')
        .map(|index| index - cursor + 1)
        .unwrap_or(1)
}

/// Returns the length of a single-rune character literal, or zero when the
/// apostrophe does not open one.
fn simple_character(line: &[u8], cursor: usize) -> usize {
    if cursor + 1 >= line.len() || line[cursor + 1] == b'\'' {
        return 0;
    }
    let size = utf8_len(line[cursor + 1]);
    if size == 0 || cursor + 1 + size >= line.len() || line[cursor + 1 + size] != b'\'' {
        return 0;
    }
    1 + size + 1
}

/// The byte length of a UTF-8 sequence from its lead byte, or zero for a
/// continuation byte.
fn utf8_len(lead: u8) -> usize {
    if lead < 0x80 {
        1
    } else if lead >= 0xF0 {
        4
    } else if lead >= 0xE0 {
        3
    } else if lead >= 0xC0 {
        2
    } else {
        0
    }
}

/// Reports whether a line is empty or only ASCII whitespace.
fn line_is_blank(line: &[u8]) -> bool {
    line.iter().all(|&character| byte_is_space(character))
}

/// Reports whether a byte is ASCII whitespace. Newline is excluded because the
/// source is already split on it.
fn byte_is_space(character: u8) -> bool {
    matches!(character, b' ' | b'\t' | b'\r' | 0x0c | 0x0b)
}

/// Reports whether a byte may appear in an identifier.
fn byte_is_identifier(character: u8) -> bool {
    character == b'_' || character.is_ascii_alphanumeric()
}

/// Reports whether prefix occurs in line at the cursor.
fn has_prefix_at(line: &[u8], cursor: usize, prefix: &[u8]) -> bool {
    line.get(cursor..cursor + prefix.len()) == Some(prefix)
}

/// Reports whether any prefix occurs in line at the cursor.
fn starts_with_any(line: &[u8], cursor: usize, prefixes: &[String]) -> bool {
    prefixes.iter().any(|prefix| has_prefix_at(line, cursor, prefix.as_bytes()))
}

// Languages.

/// The double-quoted string and single-quoted character delimiters shared by the
/// C-family languages.
fn c_family_quotes() -> Vec<Quote_Delimiter> {
    vec![
        Quote_Delimiter { open: "\"".into(), close: "\"".into(), escape: b'\\', ..Default::default() },
        Quote_Delimiter {
            open: "'".into(),
            close: "'".into(),
            escape: b'\\',
            character_like: true,
        },
    ]
}

/// Plain double- and single-quoted string delimiters with backslash escapes.
fn plain_quotes() -> Vec<Quote_Delimiter> {
    vec![
        Quote_Delimiter { open: "\"".into(), close: "\"".into(), escape: b'\\', ..Default::default() },
        Quote_Delimiter { open: "'".into(), close: "'".into(), escape: b'\\', ..Default::default() },
    ]
}

/// A single double-quoted string delimiter with backslash escapes.
fn double_quote() -> Vec<Quote_Delimiter> {
    vec![Quote_Delimiter { open: "\"".into(), close: "\"".into(), escape: b'\\', ..Default::default() }]
}

pub fn language_go() -> Language {
    Language {
        name: "Go".into(),
        test_infixes: vec!["_test.".into()],
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        verbatim_strings: vec![Verbatim_Delimiter { open: "`".into(), close: "`".into(), ..Default::default() }],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_rust() -> Language {
    Language {
        name: "Rust".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        block_comment_nests: true,
        verbatim_strings: vec![
            Verbatim_Delimiter { open: "br".into(), hashable: true, ..Default::default() },
            Verbatim_Delimiter { open: "r".into(), hashable: true, ..Default::default() },
        ],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_python() -> Language {
    Language {
        name: "Python".into(),
        test_prefixes: vec!["test_".into()],
        test_infixes: vec!["_test.".into()],
        line_comment: vec!["#".into()],
        // The triple quotes precede the single quotes so the scanner takes the
        // whole `"""` rather than an empty string followed by a quote.
        verbatim_strings: vec![
            Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() },
            Verbatim_Delimiter { open: "'''".into(), close: "'''".into(), ..Default::default() },
        ],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_java_script() -> Language {
    Language {
        name: "JavaScript".into(),
        test_infixes: vec![".test.".into(), ".spec.".into()],
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        verbatim_strings: vec![Verbatim_Delimiter { open: "`".into(), close: "`".into(), ..Default::default() }],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_type_script() -> Language {
    Language {
        name: "TypeScript".into(),
        test_infixes: vec![".test.".into(), ".spec.".into()],
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        verbatim_strings: vec![Verbatim_Delimiter { open: "`".into(), close: "`".into(), ..Default::default() }],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_c() -> Language {
    Language {
        name: "C".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_cpp() -> Language {
    Language {
        name: "C++".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_c_sharp() -> Language {
    Language {
        name: "C#".into(),
        test_infixes: vec!["Test.".into(), "Tests.".into()],
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        verbatim_strings: vec![Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() }],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_java() -> Language {
    Language {
        name: "Java".into(),
        test_infixes: vec!["Test.".into(), "Tests.".into()],
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        verbatim_strings: vec![Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() }],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_swift() -> Language {
    Language {
        name: "Swift".into(),
        test_infixes: vec!["Tests.".into(), "Test.".into()],
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        block_comment_nests: true,
        verbatim_strings: vec![Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() }],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_kotlin() -> Language {
    Language {
        name: "Kotlin".into(),
        test_infixes: vec!["Test.".into(), "Tests.".into()],
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        block_comment_nests: true,
        verbatim_strings: vec![Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() }],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_scala() -> Language {
    Language {
        name: "Scala".into(),
        test_infixes: vec!["Test.".into(), "Tests.".into(), "Spec.".into()],
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        block_comment_nests: true,
        verbatim_strings: vec![Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() }],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_shell() -> Language {
    Language {
        name: "Shell".into(),
        line_comment: vec!["#".into()],
        heredoc: true,
        quote_strings: vec![
            Quote_Delimiter { open: "\"".into(), close: "\"".into(), escape: b'\\', ..Default::default() },
            Quote_Delimiter { open: "'".into(), close: "'".into(), ..Default::default() },
        ],
        ..Default::default()
    }
}

pub fn language_ruby() -> Language {
    Language {
        name: "Ruby".into(),
        test_infixes: vec!["_spec.".into(), "_test.".into()],
        line_comment: vec!["#".into()],
        heredoc: true,
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_yaml() -> Language {
    Language {
        name: "YAML".into(),
        line_comment: vec!["#".into()],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_toml() -> Language {
    Language {
        name: "TOML".into(),
        line_comment: vec!["#".into()],
        verbatim_strings: vec![
            Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() },
            Verbatim_Delimiter { open: "'''".into(), close: "'''".into(), ..Default::default() },
        ],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_sql() -> Language {
    Language {
        name: "SQL".into(),
        line_comment: vec!["--".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_makefile() -> Language {
    Language { name: "Makefile".into(), line_comment: vec!["#".into()], ..Default::default() }
}

pub fn language_dockerfile() -> Language {
    Language { name: "Dockerfile".into(), line_comment: vec!["#".into()], ..Default::default() }
}

pub fn language_html() -> Language {
    Language {
        name: "HTML".into(),
        block_comment_open: "<!--".into(),
        block_comment_close: "-->".into(),
        ..Default::default()
    }
}

pub fn language_xml() -> Language {
    Language {
        name: "XML".into(),
        block_comment_open: "<!--".into(),
        block_comment_close: "-->".into(),
        ..Default::default()
    }
}

pub fn language_css() -> Language {
    Language {
        name: "CSS".into(),
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_scss() -> Language {
    Language {
        name: "SCSS".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_less() -> Language {
    Language {
        name: "LESS".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_lua() -> Language {
    Language {
        name: "Lua".into(),
        line_comment: vec!["--".into()],
        long_bracket: true,
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_odin() -> Language {
    Language {
        name: "Odin".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        block_comment_nests: true,
        verbatim_strings: vec![Verbatim_Delimiter { open: "`".into(), close: "`".into(), ..Default::default() }],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_zig() -> Language {
    Language {
        name: "Zig".into(),
        line_comment: vec!["//".into()],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_objective_c() -> Language {
    Language {
        name: "Objective-C".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_dart() -> Language {
    Language {
        name: "Dart".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        block_comment_nests: true,
        verbatim_strings: vec![
            Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() },
            Verbatim_Delimiter { open: "'''".into(), close: "'''".into(), ..Default::default() },
        ],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_php() -> Language {
    Language {
        name: "PHP".into(),
        test_infixes: vec!["Test.".into()],
        line_comment: vec!["//".into(), "#".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_solidity() -> Language {
    Language {
        name: "Solidity".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_groovy() -> Language {
    Language {
        name: "Groovy".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        verbatim_strings: vec![
            Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() },
            Verbatim_Delimiter { open: "'''".into(), close: "'''".into(), ..Default::default() },
        ],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_verilog() -> Language {
    Language {
        name: "Verilog".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_glsl() -> Language {
    Language {
        name: "GLSL".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_hlsl() -> Language {
    Language {
        name: "HLSL".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_arduino() -> Language {
    Language {
        name: "Arduino".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_protobuf() -> Language {
    Language {
        name: "Protobuf".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_thrift() -> Language {
    Language {
        name: "Thrift".into(),
        line_comment: vec!["//".into(), "#".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_jsonc() -> Language {
    Language {
        name: "JSONC".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_hcl() -> Language {
    Language {
        name: "HCL".into(),
        line_comment: vec!["#".into(), "//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_nix() -> Language {
    Language {
        name: "Nix".into(),
        line_comment: vec!["#".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        verbatim_strings: vec![Verbatim_Delimiter { open: "''".into(), close: "''".into(), ..Default::default() }],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_markdown() -> Language {
    Language {
        name: "Markdown".into(),
        block_comment_open: "<!--".into(),
        block_comment_close: "-->".into(),
        ..Default::default()
    }
}

pub fn language_vue() -> Language {
    Language {
        name: "Vue".into(),
        block_comment_open: "<!--".into(),
        block_comment_close: "-->".into(),
        ..Default::default()
    }
}

pub fn language_svelte() -> Language {
    Language {
        name: "Svelte".into(),
        block_comment_open: "<!--".into(),
        block_comment_close: "-->".into(),
        ..Default::default()
    }
}

pub fn language_astro() -> Language {
    Language {
        name: "Astro".into(),
        block_comment_open: "<!--".into(),
        block_comment_close: "-->".into(),
        ..Default::default()
    }
}

pub fn language_xaml() -> Language {
    Language {
        name: "XAML".into(),
        block_comment_open: "<!--".into(),
        block_comment_close: "-->".into(),
        ..Default::default()
    }
}

pub fn language_xslt() -> Language {
    Language {
        name: "XSLT".into(),
        block_comment_open: "<!--".into(),
        block_comment_close: "-->".into(),
        ..Default::default()
    }
}

pub fn language_haskell() -> Language {
    Language {
        name: "Haskell".into(),
        line_comment: vec!["--".into()],
        block_comment_open: "{-".into(),
        block_comment_close: "-}".into(),
        block_comment_nests: true,
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_ocaml() -> Language {
    Language {
        name: "OCaml".into(),
        block_comment_open: "(*".into(),
        block_comment_close: "*)".into(),
        block_comment_nests: true,
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_f_sharp() -> Language {
    Language {
        name: "F#".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "(*".into(),
        block_comment_close: "*)".into(),
        block_comment_nests: true,
        verbatim_strings: vec![Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() }],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_julia() -> Language {
    Language {
        name: "Julia".into(),
        line_comment: vec!["#".into()],
        block_comment_open: "#=".into(),
        block_comment_close: "=#".into(),
        block_comment_nests: true,
        verbatim_strings: vec![Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() }],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_nim() -> Language {
    Language {
        name: "Nim".into(),
        line_comment: vec!["#".into()],
        block_comment_open: "#[".into(),
        block_comment_close: "]#".into(),
        block_comment_nests: true,
        verbatim_strings: vec![Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() }],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_common_lisp() -> Language {
    Language {
        name: "Common Lisp".into(),
        line_comment: vec![";".into()],
        block_comment_open: "#|".into(),
        block_comment_close: "|#".into(),
        block_comment_nests: true,
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_scheme() -> Language {
    Language {
        name: "Scheme".into(),
        line_comment: vec![";".into()],
        block_comment_open: "#|".into(),
        block_comment_close: "|#".into(),
        block_comment_nests: true,
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_racket() -> Language {
    Language {
        name: "Racket".into(),
        line_comment: vec![";".into()],
        block_comment_open: "#|".into(),
        block_comment_close: "|#".into(),
        block_comment_nests: true,
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_clojure() -> Language {
    Language {
        name: "Clojure".into(),
        line_comment: vec![";".into()],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_emacs_lisp() -> Language {
    Language {
        name: "Emacs Lisp".into(),
        line_comment: vec![";".into()],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_erlang() -> Language {
    Language {
        name: "Erlang".into(),
        line_comment: vec!["%".into()],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_fortran() -> Language {
    Language {
        name: "Fortran".into(),
        line_comment: vec!["!".into()],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_ada() -> Language {
    Language {
        name: "Ada".into(),
        line_comment: vec!["--".into()],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_d() -> Language {
    Language {
        name: "D".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "/*".into(),
        block_comment_close: "*/".into(),
        verbatim_strings: vec![Verbatim_Delimiter { open: "`".into(), close: "`".into(), ..Default::default() }],
        quote_strings: c_family_quotes(),
        ..Default::default()
    }
}

pub fn language_pascal() -> Language {
    Language {
        name: "Pascal".into(),
        line_comment: vec!["//".into()],
        block_comment_open: "{".into(),
        block_comment_close: "}".into(),
        quote_strings: vec![Quote_Delimiter { open: "'".into(), close: "'".into(), ..Default::default() }],
        ..Default::default()
    }
}

pub fn language_r() -> Language {
    Language {
        name: "R".into(),
        line_comment: vec!["#".into()],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_elixir() -> Language {
    Language {
        name: "Elixir".into(),
        test_infixes: vec!["_test.".into()],
        line_comment: vec!["#".into()],
        verbatim_strings: vec![
            Verbatim_Delimiter { open: "\"\"\"".into(), close: "\"\"\"".into(), ..Default::default() },
            Verbatim_Delimiter { open: "'''".into(), close: "'''".into(), ..Default::default() },
        ],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_crystal() -> Language {
    Language {
        name: "Crystal".into(),
        line_comment: vec!["#".into()],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_power_shell() -> Language {
    Language {
        name: "PowerShell".into(),
        line_comment: vec!["#".into()],
        block_comment_open: "<#".into(),
        block_comment_close: "#>".into(),
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_fish() -> Language {
    Language {
        name: "Fish".into(),
        line_comment: vec!["#".into()],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_nushell() -> Language {
    Language {
        name: "Nushell".into(),
        line_comment: vec!["#".into()],
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_cmake() -> Language {
    Language {
        name: "CMake".into(),
        line_comment: vec!["#".into()],
        long_bracket: true,
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_tcl() -> Language {
    Language {
        name: "Tcl".into(),
        line_comment: vec!["#".into()],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

pub fn language_perl() -> Language {
    Language {
        name: "Perl".into(),
        line_comment: vec!["#".into()],
        heredoc: true,
        quote_strings: plain_quotes(),
        ..Default::default()
    }
}

pub fn language_tex() -> Language {
    Language { name: "TeX".into(), line_comment: vec!["%".into()], ..Default::default() }
}

pub fn language_visual_basic() -> Language {
    Language {
        name: "Visual Basic".into(),
        line_comment: vec!["'".into()],
        quote_strings: double_quote(),
        ..Default::default()
    }
}

/// Returns the seeded language for a file extension, with the leading dot.
pub fn language_for_extension(extension: &str) -> Option<Language> {
    let language = match extension {
        ".go" => language_go(),
        ".rs" => language_rust(),
        ".py" => language_python(),
        ".js" | ".jsx" | ".mjs" | ".cjs" => language_java_script(),
        ".ts" | ".tsx" => language_type_script(),
        ".lua" => language_lua(),
        ".odin" => language_odin(),
        ".zig" => language_zig(),
        ".c" | ".h" => language_c(),
        ".cpp" | ".cc" | ".cxx" | ".hpp" | ".hh" | ".hxx" => language_cpp(),
        ".cs" => language_c_sharp(),
        ".java" => language_java(),
        ".swift" => language_swift(),
        ".kt" | ".kts" => language_kotlin(),
        ".scala" | ".sc" => language_scala(),
        ".sh" | ".bash" | ".zsh" => language_shell(),
        ".rb" => language_ruby(),
        ".yaml" | ".yml" => language_yaml(),
        ".toml" => language_toml(),
        ".sql" => language_sql(),
        ".mk" => language_makefile(),
        ".dockerfile" => language_dockerfile(),
        ".html" | ".htm" => language_html(),
        ".xml" | ".svg" => language_xml(),
        ".css" => language_css(),
        ".scss" => language_scss(),
        ".less" => language_less(),
        _ => return extension_match_more(extension),
    };
    Some(language)
}

/// Continues the lookup for the C-style and markup additions.
fn extension_match_more(extension: &str) -> Option<Language> {
    let language = match extension {
        ".m" | ".mm" => language_objective_c(),
        ".dart" => language_dart(),
        ".php" | ".phtml" => language_php(),
        ".sol" => language_solidity(),
        ".groovy" | ".gradle" => language_groovy(),
        ".v" | ".sv" | ".svh" => language_verilog(),
        ".glsl" | ".vert" | ".frag" | ".comp" | ".geom" => language_glsl(),
        ".hlsl" => language_hlsl(),
        ".ino" => language_arduino(),
        ".proto" => language_protobuf(),
        ".thrift" => language_thrift(),
        ".jsonc" | ".json5" => language_jsonc(),
        ".tf" | ".hcl" | ".tfvars" => language_hcl(),
        ".nix" => language_nix(),
        ".md" | ".markdown" => language_markdown(),
        ".vue" => language_vue(),
        ".svelte" => language_svelte(),
        ".astro" => language_astro(),
        ".xaml" => language_xaml(),
        ".xsl" | ".xslt" => language_xslt(),
        _ => return extension_match_rest(extension),
    };
    Some(language)
}

/// Continues the lookup for the remaining languages.
fn extension_match_rest(extension: &str) -> Option<Language> {
    let language = match extension {
        ".hs" | ".lhs" => language_haskell(),
        ".ml" | ".mli" => language_ocaml(),
        ".fs" | ".fsx" | ".fsi" => language_f_sharp(),
        ".jl" => language_julia(),
        ".nim" | ".nims" => language_nim(),
        ".lisp" | ".lsp" | ".cl" => language_common_lisp(),
        ".scm" | ".ss" => language_scheme(),
        ".rkt" => language_racket(),
        ".clj" | ".cljs" | ".cljc" | ".edn" => language_clojure(),
        ".el" => language_emacs_lisp(),
        ".erl" | ".hrl" => language_erlang(),
        ".f90" | ".f95" | ".f03" | ".f08" | ".f" | ".for" => language_fortran(),
        ".adb" | ".ads" | ".ada" => language_ada(),
        ".d" => language_d(),
        ".pas" | ".pp" | ".dpr" => language_pascal(),
        ".r" | ".R" => language_r(),
        ".ex" | ".exs" => language_elixir(),
        ".cr" => language_crystal(),
        ".ps1" | ".psm1" | ".psd1" => language_power_shell(),
        ".fish" => language_fish(),
        ".nu" => language_nushell(),
        ".cmake" => language_cmake(),
        ".tcl" => language_tcl(),
        ".pl" | ".pm" | ".t" | ".pod" => language_perl(),
        ".tex" | ".sty" | ".cls" | ".ltx" => language_tex(),
        ".vb" => language_visual_basic(),
        _ => return None,
    };
    Some(language)
}

/// Returns the language for an extensionless file recognized by its name.
pub fn language_for_filename(name: &str) -> Option<Language> {
    match name {
        "Makefile" | "makefile" | "GNUmakefile" => Some(language_makefile()),
        "Dockerfile" => Some(language_dockerfile()),
        "CMakeLists.txt" => Some(language_cmake()),
        _ => None,
    }
}

/// Resolves the language for a path by its extension, or for an extensionless
/// file by its name.
fn language_for_path(file_path: &str) -> Option<Language> {
    language_for_extension(&path_extension(file_path))
        .or_else(|| language_for_filename(&path_base(file_path)))
}

/// The final element of a slash-separated path.
fn path_base(file_path: &str) -> String {
    file_path.rsplit('/').next().unwrap_or(file_path).to_string()
}

/// The suffix beginning at the final dot in the final path element, with the
/// dot, or empty when there is none.
fn path_extension(file_path: &str) -> String {
    let base = path_base(file_path);
    match base.rfind('.') {
        Some(dot) => base[dot..].to_string(),
        None => String::new(),
    }
}

// Count.

/// One counted file: its path, the language it was read as, and its line
/// partition.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct File_Count {
    pub path: String,
    pub language: String,
    pub counts: Counts,
    pub is_test: bool,
}

/// The result of counting a tree: one `File_Count` per counted file, in the
/// lexical order the walk visited them.
#[derive(Clone, Debug, Default)]
pub struct Report {
    pub files: Vec<File_Count>,
}

/// One directory entry the walk seam reports: a base name and whether it is a
/// directory.
#[derive(Clone, Debug)]
pub struct Walk_Entry {
    pub name: String,
    pub is_directory: bool,
}

/// A file tree to count and the options to apply while walking it. The tree
/// itself and the reads are injected as closures, so the library does no
/// ambient I/O.
#[derive(Clone, Debug, Default)]
pub struct Count_Input {
    /// The directory the walk starts at; paths in the report are relative to it.
    pub root: String,
    /// Count dot-prefixed entries that are skipped by default.
    pub include_hidden: bool,
    /// Bounds the read-and-classify worker pool; below one means one.
    pub concurrency: usize,
}

/// A file the walk selected for counting and the language to read it as.
#[derive(Clone, Debug)]
struct Candidate {
    pub path: String,
    pub language: Language,
}

/// Walks the tree, classifies every recognized file, and returns one
/// `File_Count` per file in lexical walk order. Reading and classifying fan out
/// across scoped threads, which does not affect the result order.
pub fn count<List, Read, Ignore>(
    input: &Count_Input,
    list_dir: List,
    read_file: Read,
    is_ignored: Ignore,
) -> Report
where
    List: Fn(&str) -> Vec<Walk_Entry>,
    Read: Fn(&str) -> Option<Vec<u8>> + Sync,
    Ignore: Fn(&str, bool) -> bool,
{
    let candidates = walk_dir(&input.root, &list_dir, &is_ignored, input.include_hidden);
    let files = count_classify(&read_file, &candidates, input.concurrency);
    Report { files }
}

/// Walks one directory, returning the recognized files to count, pruning hidden
/// and ignored directories so their contents are never read.
fn walk_dir<List, Ignore>(
    dir: &str,
    list_dir: &List,
    is_ignored: &Ignore,
    include_hidden: bool,
) -> Vec<Candidate>
where
    List: Fn(&str) -> Vec<Walk_Entry>,
    Ignore: Fn(&str, bool) -> bool,
{
    list_dir(dir)
        .iter()
        .flat_map(|entry| count_visit(dir, entry, list_dir, is_ignored, include_hidden))
        .collect()
}

/// Decides one walked entry: prune it (returning nothing and not descending),
/// skip it, or take it as a candidate.
fn count_visit<List, Ignore>(
    dir: &str,
    entry: &Walk_Entry,
    list_dir: &List,
    is_ignored: &Ignore,
    include_hidden: bool,
) -> Vec<Candidate>
where
    List: Fn(&str) -> Vec<Walk_Entry>,
    Ignore: Fn(&str, bool) -> bool,
{
    let child = join_path(dir, &entry.name);
    if !include_hidden && path_is_hidden(&child) {
        return Vec::new();
    }
    if is_ignored(&child, entry.is_directory) {
        return Vec::new();
    }
    if entry.is_directory {
        return walk_dir(&child, list_dir, is_ignored, include_hidden);
    }
    // Detection is by extension; an unrecognized file is skipped silently.
    match language_for_path(&child) {
        Some(language) => vec![Candidate { path: child, language }],
        None => Vec::new(),
    }
}

/// Joins a directory and a child name, leaving a `.` root off the front so the
/// report's top-level paths are bare names.
fn join_path(dir: &str, name: &str) -> String {
    if dir == "." {
        name.to_string()
    } else {
        format!("{dir}/{name}")
    }
}

/// Reads and classifies each candidate across scoped threads, dropping any
/// unreadable or binary file, and returns the results in candidate order.
fn count_classify<Read>(read_file: &Read, candidates: &[Candidate], concurrency: usize) -> Vec<File_Count>
where
    Read: Fn(&str) -> Option<Vec<u8>> + Sync,
{
    if candidates.is_empty() {
        return Vec::new();
    }
    let worker_count = concurrency.max(1).min(candidates.len());
    // The dialect bans the shared mutable state a dynamic job queue needs, so
    // work is assigned statically. A round-robin stride (thread t takes every
    // worker_count-th candidate from t) spreads clusters of large files — a
    // vendored subtree, say — across all threads instead of stranding them in
    // one contiguous chunk, so the threads finish together. Each result carries
    // its candidate index, and a BTreeMap restores walk order without a `mut`
    // sort, identical to a sequential run.
    let ordered: collections::BTreeMap<usize, File_Count> = thread::scope(|scope| {
        let handles: Vec<_> = (0..worker_count)
            .map(|offset| {
                scope.spawn(move || {
                    (offset..candidates.len())
                        .step_by(worker_count)
                        .filter_map(|index| {
                            count_one(read_file, &candidates[index]).map(|file| (index, file))
                        })
                        .collect::<Vec<(usize, File_Count)>>()
                })
            })
            .collect();
        handles.into_iter().flat_map(|handle| handle.join().unwrap_or_default()).collect()
    });
    ordered.into_values().collect()
}

/// Reads and classifies a single candidate, returning `None` to drop a file that
/// is unreadable or binary.
fn count_one<Read>(read_file: &Read, candidate: &Candidate) -> Option<File_Count>
where
    Read: Fn(&str) -> Option<Vec<u8>>,
{
    let content = read_file(&candidate.path)?;
    // Detection is by extension, so a recognized extension holding binary data
    // is dropped here rather than counted as garbage lines.
    if content_is_binary(&content) {
        return None;
    }
    // Borrow the candidate's language rather than cloning it per file.
    let counts = classify_bytes(&content, &candidate.language);
    Some(File_Count {
        path: candidate.path.clone(),
        language: candidate.language.name.clone(),
        counts,
        is_test: path_is_test(&candidate.path, &candidate.language),
    })
}

/// Bounds how far the binary sniff scans for a NUL byte.
const BINARY_SNIFF_BYTES: usize = 8192;

/// Reports whether content holds a NUL byte in its first chunk, the cheap
/// heuristic for "not text".
fn content_is_binary(content: &[u8]) -> bool {
    content.iter().take(BINARY_SNIFF_BYTES).any(|&byte| byte == 0)
}

/// Reports whether a path is test code: it lives under a test directory, or its
/// base name carries the language's test prefix or infix.
fn path_is_test(file_path: &str, language: &Language) -> bool {
    if path_has_test_directory(file_path) {
        return true;
    }
    let base = path_base(file_path);
    language.test_prefixes.iter().any(|prefix| base.starts_with(prefix.as_str()))
        || language.test_infixes.iter().any(|infix| base.contains(infix.as_str()))
}

/// Reports whether any component of a path is a conventional test directory.
fn path_has_test_directory(file_path: &str) -> bool {
    file_path.split('/').any(|component| matches!(component, "test" | "tests" | "spec" | "__tests__"))
}

/// Reports whether a path's final element begins with a dot.
fn path_is_hidden(file_path: &str) -> bool {
    path_base(file_path).starts_with('.')
}

// In-memory tree (for tests and the explicit-file path).

/// An in-memory file tree: the seam the tests count against without touching a
/// real filesystem.
#[derive(Clone, Debug, Default)]
pub struct File_Tree {
    pub files: Vec<Tree_File>,
}

/// One file in an in-memory tree.
#[derive(Clone, Debug)]
pub struct Tree_File {
    pub path: String,
    pub data: Vec<u8>,
}

/// Lists the immediate children of a directory in an in-memory tree, synthesizing
/// the intermediate directories the file paths imply.
pub fn tree_list_dir(tree: &File_Tree, dir: &str) -> Vec<Walk_Entry> {
    let directories: collections::BTreeSet<String> = tree
        .files
        .iter()
        .filter_map(|file| tree_child(dir, &file.path))
        .filter(|(_, is_directory)| *is_directory)
        .map(|(name, _)| name)
        .collect();
    let names: collections::BTreeSet<String> = tree
        .files
        .iter()
        .filter_map(|file| tree_child(dir, &file.path))
        .map(|(name, _)| name)
        .collect();
    names
        .into_iter()
        .map(|name| Walk_Entry { is_directory: directories.contains(&name), name })
        .collect()
}

/// The immediate child of `dir` on the way to `path`, with whether it is a
/// directory, or `None` when `path` is not under `dir`.
fn tree_child(dir: &str, path: &str) -> Option<(String, bool)> {
    let relative = if dir == "." { path } else { path.strip_prefix(&format!("{dir}/"))? };
    if relative.is_empty() {
        return None;
    }
    match relative.split_once('/') {
        Some((head, _rest)) => Some((head.to_string(), true)),
        None => Some((relative.to_string(), false)),
    }
}

/// Reads one file's bytes from an in-memory tree.
pub fn tree_read_file(tree: &File_Tree, path: &str) -> Option<Vec<u8>> {
    tree.files.iter().find(|file| file.path == path).map(|file| file.data.clone())
}

// Render.

/// A report and whether to break each language into its files.
#[derive(Clone, Debug)]
pub struct Render_Input {
    pub report: Report,
    pub show_files: bool,
}

/// One language's files and their summed partition.
#[derive(Clone, Debug, Default)]
struct Language_Group {
    pub name: String,
    pub category: String,
    pub files: usize,
    pub counts: Counts,
    pub source_files: usize,
    pub source: Counts,
    pub test_files: usize,
    pub test: Counts,
    pub members: Vec<File_Count>,
}

/// A category and the language groups it holds, in name order.
#[derive(Clone, Debug, Default)]
struct Category_Group {
    pub name: String,
    pub languages: Vec<Language_Group>,
}

/// One printable table row; every cell is already a string so the header and the
/// numeric rows share one width and formatting path.
#[derive(Clone, Debug, Default)]
struct Render_Row {
    pub name: String,
    pub files: String,
    pub lines: String,
    pub code: String,
    pub comments: String,
    pub blanks: String,
    pub percent: String,
}

/// The printed width of each table column.
#[derive(Copy, Clone, Debug, Default)]
struct Render_Column_Widths {
    pub name: usize,
    pub files: usize,
    pub lines: usize,
    pub code: usize,
    pub comments: usize,
    pub blanks: usize,
    pub percent: usize,
}

/// A row's label, file count, partition, and the code total its code is a share
/// of.
struct Counts_Row_Input {
    pub name: String,
    pub files: usize,
    pub counts: Counts,
    /// The denominator for the %Code share, or zero to leave it blank.
    pub total_code: usize,
}

/// A report's summed totals split into source and test.
#[derive(Copy, Clone, Debug, Default)]
struct Render_Totals {
    pub files: usize,
    pub counts: Counts,
    pub source_files: usize,
    pub source: Counts,
    pub test_files: usize,
    pub test: Counts,
}

/// Writes the report as an aligned table: one row per language sorted by name
/// and grouped by category, a Total row, and — with `show_files` — each file
/// indented under its language. Columns are sized to their widest cell, so the
/// output is stable for a given report regardless of counting order.
pub fn render(input: &Render_Input) -> String {
    let groups = report_groups(&input.report);
    let categories = report_categories(&groups);
    let rows = report_rows(&categories, input.show_files);
    let totals = report_total(&groups);
    let widths = fold_widths(fold_widths(Render_Column_Widths::default(), &rows), &totals);
    let rule = "─".repeat(render_column_widths_total(&widths));
    let body = rows[1..].iter().map(|row| render_row_format(row, &widths));
    let total_lines = totals.iter().map(|row| render_row_format(row, &widths));
    let lines = iter::once(rule.clone())
        .chain(iter::once(render_row_format(&rows[0], &widths)))
        .chain(iter::once(rule.clone()))
        .chain(body)
        .chain(iter::once(rule.clone()))
        .chain(total_lines)
        .chain(iter::once(rule));
    // Each emitted line is trimmed of the trailing padding a label or blank
    // percentage leaves, then terminated, matching the Go renderer's Fprintln.
    lines.map(|line| format!("{}\n", line.trim_end_matches(' '))).collect()
}

/// Folds a report's files into per-language groups sorted by name, splitting each
/// group's partition into source and test and preserving its files in report
/// order.
fn report_groups(report: &Report) -> Vec<Language_Group> {
    let names: collections::BTreeSet<String> =
        report.files.iter().map(|file| file.language.clone()).collect();
    names.iter().map(|name| build_group(report, name)).collect()
}

/// Builds one language's group: its members in report order and their summed
/// partition.
fn build_group(report: &Report, name: &str) -> Language_Group {
    let members: Vec<File_Count> =
        report.files.iter().filter(|file| file.language == name).cloned().collect();
    let base = Language_Group {
        name: name.to_string(),
        category: language_category(name),
        members: members.clone(),
        ..Default::default()
    };
    members.iter().fold(base, group_add)
}

/// Adds one file into a language group's running counts and source/test split.
fn group_add(group: Language_Group, file: &File_Count) -> Language_Group {
    let counts = counts_add(group.counts, file.counts);
    if file.is_test {
        Language_Group {
            files: group.files + 1,
            counts,
            test_files: group.test_files + 1,
            test: counts_add(group.test, file.counts),
            ..group
        }
    } else {
        Language_Group {
            files: group.files + 1,
            counts,
            source_files: group.source_files + 1,
            source: counts_add(group.source, file.counts),
            ..group
        }
    }
}

/// Returns a language's taxonomy category by display name. This match is the
/// single place the classification lives.
fn language_category(name: &str) -> String {
    let category = match name {
        "C" | "C++" | "Rust" | "Zig" | "Odin" | "Ada" | "Fortran" | "Pascal" | "Arduino"
        | "Solidity" => "Systems",
        "Go" | "Java" | "C#" | "Kotlin" | "Scala" | "Dart" | "Crystal" | "Nim" | "D" | "Swift"
        | "Objective-C" | "Haskell" | "OCaml" | "F#" | "Visual Basic" => "Managed",
        "Python" | "Ruby" | "JavaScript" | "TypeScript" | "Lua" | "Perl" | "PHP" | "R" | "Tcl"
        | "Julia" | "Groovy" | "Elixir" | "Erlang" | "Clojure" | "Scheme" | "Common Lisp"
        | "Racket" | "Emacs Lisp" => "Dynamically Typed",
        "Shell" | "PowerShell" | "Fish" | "Nushell" => "Shell",
        "HTML" | "XML" | "CSS" | "SCSS" | "LESS" | "Markdown" | "YAML" | "TOML" | "JSONC"
        | "XAML" | "XSLT" | "Vue" | "Svelte" | "Astro" | "Protobuf" | "Thrift" | "TeX" => {
            "Markup & Data"
        }
        "Makefile" | "Dockerfile" | "CMake" | "HCL" | "Nix" => "Build & Config",
        "SQL" => "Query",
        "Verilog" | "GLSL" | "HLSL" => "Hardware",
        _ => "Other",
    };
    category.to_string()
}

/// The fixed display order of the categories.
fn category_order() -> Vec<String> {
    [
        "Systems",
        "Managed",
        "Dynamically Typed",
        "Shell",
        "Markup & Data",
        "Build & Config",
        "Query",
        "Hardware",
        "Other",
    ]
    .iter()
    .map(|name| name.to_string())
    .collect()
}

/// Buckets the name-sorted language groups into categories in the fixed display
/// order.
fn report_categories(groups: &[Language_Group]) -> Vec<Category_Group> {
    category_order()
        .iter()
        .map(|category| Category_Group {
            name: category.clone(),
            languages: groups.iter().filter(|group| &group.category == category).cloned().collect(),
        })
        .collect()
}

/// Builds the header, then each non-empty category: a label row followed by its
/// languages.
fn report_rows(categories: &[Category_Group], show_files: bool) -> Vec<Render_Row> {
    let body = categories
        .iter()
        .filter(|category| !category.languages.is_empty())
        .flat_map(|category| category_rows(category, show_files));
    iter::once(render_header_row()).chain(body).collect()
}

/// Builds one category's rows: a label row then each of its languages' rows.
fn category_rows(category: &Category_Group, show_files: bool) -> Vec<Render_Row> {
    let label = Render_Row { name: category.name.clone(), ..Default::default() };
    let languages = category.languages.iter().flat_map(|group| language_rows(group, show_files));
    iter::once(label).chain(languages).collect()
}

/// Appends a language's indented row, then its files (with `show_files`) or its
/// source and test sub-rows.
fn language_rows(group: &Language_Group, show_files: bool) -> Vec<Render_Row> {
    let head = counts_row(&Counts_Row_Input {
        name: format!("  {}", group.name),
        files: group.files,
        counts: group.counts,
        total_code: 0,
    });
    if show_files {
        let files = group.members.iter().map(|member| file_row(format!("    {}", member.path), member.counts));
        iter::once(head).chain(files).collect()
    } else {
        let split = split_rows("    ", group.source_files, group.source, group.test_files, group.test);
        iter::once(head).chain(split).collect()
    }
}

/// Returns indented source and test sub-rows, but only when test files are
/// present — a language without tests shows just its single total row. The %Code
/// on a sub-row is its share of the group's own code, so source and tests sum to
/// 100%.
fn split_rows(
    indent: &str,
    source_files: usize,
    source: Counts,
    test_files: usize,
    test: Counts,
) -> Vec<Render_Row> {
    if test_files == 0 {
        return Vec::new();
    }
    let own_code = source.code + test.code;
    vec![
        counts_row(&Counts_Row_Input {
            name: format!("{indent}source"),
            files: source_files,
            counts: source,
            total_code: own_code,
        }),
        counts_row(&Counts_Row_Input {
            name: format!("{indent}tests"),
            files: test_files,
            counts: test,
            total_code: own_code,
        }),
    ]
}

/// Sums every group into the Total row and its source and test sub-rows.
fn report_total(groups: &[Language_Group]) -> Vec<Render_Row> {
    let combined = groups.iter().fold(Render_Totals::default(), |totals, group| Render_Totals {
        files: totals.files + group.files,
        counts: counts_add(totals.counts, group.counts),
        source_files: totals.source_files + group.source_files,
        source: counts_add(totals.source, group.source),
        test_files: totals.test_files + group.test_files,
        test: counts_add(totals.test, group.test),
    });
    let total = counts_row(&Counts_Row_Input {
        name: "Total".to_string(),
        files: combined.files,
        counts: combined.counts,
        total_code: 0,
    });
    let split = split_rows("  ", combined.source_files, combined.source, combined.test_files, combined.test);
    iter::once(total).chain(split).collect()
}

/// The table's column header row.
fn render_header_row() -> Render_Row {
    Render_Row {
        name: "Language".to_string(),
        files: "Files".to_string(),
        lines: "Lines".to_string(),
        code: "Code".to_string(),
        comments: "Comments".to_string(),
        blanks: "Blanks".to_string(),
        percent: "%Code".to_string(),
    }
}

/// Builds an aggregate row: a name, a file count, the partition, and the code's
/// share of the given code total — blank when that total is zero.
fn counts_row(input: &Counts_Row_Input) -> Render_Row {
    let percent = if input.total_code > 0 {
        let share = input.counts.code as f64 / input.total_code as f64 * 100.0;
        format!("{share:.1}%")
    } else {
        String::new()
    };
    Render_Row {
        name: input.name.clone(),
        files: with_thousands_separators(input.files),
        lines: with_thousands_separators(counts_lines(input.counts)),
        code: with_thousands_separators(input.counts.code),
        comments: with_thousands_separators(input.counts.comment),
        blanks: with_thousands_separators(input.counts.blank),
        percent,
    }
}

/// Builds a per-file row: like an aggregate row but without a file count, since
/// the row is itself one file, and without a percentage.
fn file_row(name: String, counts: Counts) -> Render_Row {
    let row = counts_row(&Counts_Row_Input { name, files: 0, counts, total_code: 0 });
    Render_Row { files: String::new(), ..row }
}

/// Sizes each column to its widest cell across the given rows, folded onto a
/// running width.
fn fold_widths(start: Render_Column_Widths, rows: &[Render_Row]) -> Render_Column_Widths {
    rows.iter().fold(start, |widths, row| Render_Column_Widths {
        name: widths.name.max(row.name.len()),
        files: widths.files.max(row.files.len()),
        lines: widths.lines.max(row.lines.len()),
        code: widths.code.max(row.code.len()),
        comments: widths.comments.max(row.comments.len()),
        blanks: widths.blanks.max(row.blanks.len()),
        percent: widths.percent.max(row.percent.len()),
    })
}

/// The printed character width of any row, which the rules match. Every cell is
/// ASCII, so character width equals byte length.
fn render_column_widths_total(widths: &Render_Column_Widths) -> usize {
    1 + widths.name
        + 2 + widths.files
        + 2 + widths.lines
        + 2 + widths.code
        + 2 + widths.comments
        + 2 + widths.blanks
        + 2 + widths.percent
}

/// Lays out one row: a leading space, the left-justified name, then each
/// right-justified numeric column behind a two-space gap.
fn render_row_format(row: &Render_Row, widths: &Render_Column_Widths) -> String {
    format!(
        " {}  {}  {}  {}  {}  {}  {}",
        pad_right(&row.name, widths.name),
        pad_left(&row.files, widths.files),
        pad_left(&row.lines, widths.lines),
        pad_left(&row.code, widths.code),
        pad_left(&row.comments, widths.comments),
        pad_left(&row.blanks, widths.blanks),
        pad_left(&row.percent, widths.percent),
    )
}

/// Right-justifies text in width by prefixing spaces.
fn pad_left(text: &str, width: usize) -> String {
    format!("{text:>width$}")
}

/// Left-justifies text in width by suffixing spaces.
fn pad_right(text: &str, width: usize) -> String {
    format!("{text:<width$}")
}

/// Renders a non-negative count with a comma between each group of three digits.
fn with_thousands_separators(value: usize) -> String {
    let digits = value.to_string();
    if digits.len() <= 3 {
        return digits;
    }
    // The lead group holds the digits that do not fill a whole group of three.
    let lead_count = match digits.len() % 3 {
        0 => 3,
        remainder => remainder,
    };
    let groups = digits.as_bytes()[lead_count..]
        .chunks(3)
        .map(|chunk| String::from_utf8_lossy(chunk).into_owned());
    iter::once(digits[..lead_count].to_string()).chain(groups).collect::<Vec<String>>().join(",")
}

// JSON.

/// Writes the report as indented JSON: a name-sorted languages array, each with
/// its category and source/test split, and a total. The format matches Go's
/// `encoding/json` with two-space indent down to the byte.
pub fn render_json(report: &Report) -> String {
    let groups = report_groups(report);
    let total = groups.iter().fold(Render_Totals::default(), |totals, group| Render_Totals {
        files: totals.files + group.files,
        counts: counts_add(totals.counts, group.counts),
        source_files: totals.source_files + group.source_files,
        source: counts_add(totals.source, group.source),
        test_files: totals.test_files + group.test_files,
        test: counts_add(totals.test, group.test),
    });
    let languages = if groups.is_empty() {
        "  \"languages\": [],\n".to_string()
    } else {
        let entries =
            groups.iter().map(json_language_entry).collect::<Vec<String>>().join(",\n");
        format!("  \"languages\": [\n{entries}\n  ],\n")
    };
    format!("{{\n{languages}  \"total\": {}\n}}\n", json_total_object(&total))
}

/// Serializes one language group as an object inside the languages array.
fn json_language_entry(group: &Language_Group) -> String {
    format!(
        "    {{\n      \"name\": {},\n      \"category\": {},\n      \"files\": {},\n      \"code\": {},\n      \"comments\": {},\n      \"blanks\": {},\n      \"source\": {},\n      \"tests\": {}\n    }}",
        json_string(&group.name),
        json_string(&group.category),
        group.files,
        group.counts.code,
        group.counts.comment,
        group.counts.blank,
        json_counts_block(group.source_files, group.source, "      "),
        json_counts_block(group.test_files, group.test, "      "),
    )
}

/// Serializes the grand total object (not nested in an array, so one indent
/// level shallower than a language entry).
fn json_total_object(total: &Render_Totals) -> String {
    format!(
        "{{\n    \"files\": {},\n    \"code\": {},\n    \"comments\": {},\n    \"blanks\": {},\n    \"source\": {},\n    \"tests\": {}\n  }}",
        total.files,
        total.counts.code,
        total.counts.comment,
        total.counts.blank,
        json_counts_block(total.source_files, total.source, "    "),
        json_counts_block(total.test_files, total.test, "    "),
    )
}

/// Serializes a files-and-partition object at the given brace indent; its fields
/// sit two spaces deeper.
fn json_counts_block(files: usize, counts: Counts, indent: &str) -> String {
    format!(
        "{{\n{indent}  \"files\": {},\n{indent}  \"code\": {},\n{indent}  \"comments\": {},\n{indent}  \"blanks\": {}\n{indent}}}",
        files, counts.code, counts.comment, counts.blank,
    )
}

/// Quotes a string for JSON, matching Go's `encoding/json`: the structural
/// escapes plus the HTML-safe `<`, `>`, and `&` escapes the default encoder
/// emits (the category names carry the `&`).
fn json_string(value: &str) -> String {
    let escaped: String = value
        .chars()
        .flat_map(|character| match character {
            '"' => vec!['\\', '"'],
            '\\' => vec!['\\', '\\'],
            '\n' => vec!['\\', 'n'],
            '\r' => vec!['\\', 'r'],
            '\t' => vec!['\\', 't'],
            '<' => "\\u003c".chars().collect(),
            '>' => "\\u003e".chars().collect(),
            '&' => "\\u0026".chars().collect(),
            other => vec![other],
        })
        .collect();
    format!("\"{escaped}\"")
}

// Run.

/// The resolved scope and override flags of one run.
#[derive(Clone, Debug, Default)]
struct Main_Scope {
    pub paths: Vec<String>,
    pub no_ignore: bool,
    pub include_hidden: bool,
    pub json: bool,
    pub show_files: bool,
}

/// The set of paths git would keep under a root, with whether the filter is
/// active. An inactive set ignores nothing (no git tree, or `--no-ignore`).
#[derive(Clone, Debug, Default)]
pub struct Ignore_Set {
    pub active: bool,
    pub kept_files: collections::HashSet<String>,
    pub kept_directories: collections::HashSet<String>,
}

/// The rendered output of a run and the process exit code.
#[derive(Clone, Debug, Default)]
pub struct Run_Result {
    pub code: i32,
    pub stdout: String,
    pub stderr: String,
}

/// Parses the command line, counts every path, and renders the table or JSON,
/// returning the output and a process exit code. The host bindings — directory
/// test, directory listing, file reads, and the ignore filter — are injected.
pub fn run<Is_Dir, List, Read_In, Read_One, Ignore_For>(
    arguments: &[String],
    concurrency: usize,
    is_directory: Is_Dir,
    list_dir: List,
    read_in: Read_In,
    read_file: Read_One,
    ignore_for: Ignore_For,
) -> Run_Result
where
    Is_Dir: Fn(&str) -> bool,
    List: Fn(&str, &str) -> Vec<Walk_Entry>,
    Read_In: Fn(&str, &str) -> Option<Vec<u8>> + Sync,
    Read_One: Fn(&str) -> Option<Vec<u8>>,
    Ignore_For: Fn(&str) -> Ignore_Set,
{
    let scope = parse_arguments(arguments);
    let files: Vec<File_Count> = scope
        .paths
        .iter()
        .flat_map(|root| {
            run_one(root, &scope, concurrency, &is_directory, &list_dir, &read_in, &read_file, &ignore_for)
        })
        .collect();
    let report = Report { files };
    let stdout = if scope.json {
        render_json(&report)
    } else {
        render(&Render_Input { report, show_files: scope.show_files })
    };
    Run_Result { code: 0, stdout, stderr: String::new() }
}

/// Counts a single path: a directory is walked (and its files re-prefixed with
/// the root), a file is classified directly.
#[allow(clippy::too_many_arguments)]
fn run_one<Is_Dir, List, Read_In, Read_One, Ignore_For>(
    root: &str,
    scope: &Main_Scope,
    concurrency: usize,
    is_directory: &Is_Dir,
    list_dir: &List,
    read_in: &Read_In,
    read_file: &Read_One,
    ignore_for: &Ignore_For,
) -> Vec<File_Count>
where
    Is_Dir: Fn(&str) -> bool,
    List: Fn(&str, &str) -> Vec<Walk_Entry>,
    Read_In: Fn(&str, &str) -> Option<Vec<u8>> + Sync,
    Read_One: Fn(&str) -> Option<Vec<u8>>,
    Ignore_For: Fn(&str) -> Ignore_Set,
{
    if is_directory(root) {
        let ignore = if scope.no_ignore { Ignore_Set::default() } else { ignore_for(root) };
        let report = count(
            &Count_Input { root: ".".to_string(), include_hidden: scope.include_hidden, concurrency },
            |dir| list_dir(root, dir),
            |relative| read_in(root, relative),
            |path, is_dir| ignore.active && ignore_set_ignores(&ignore, path, is_dir),
        );
        // The walk counts paths relative to the root; re-prefix them so files
        // from different roots stay distinguishable.
        report
            .files
            .into_iter()
            .map(|file| File_Count { path: join_path(root, &file.path), ..file })
            .collect()
    } else {
        run_one_file(root, read_file)
    }
}

/// Classifies one explicitly named file, or nothing when its extension is
/// unrecognized or it cannot be read.
fn run_one_file<Read_One>(name: &str, read_file: &Read_One) -> Vec<File_Count>
where
    Read_One: Fn(&str) -> Option<Vec<u8>>,
{
    let Some(language) = language_for_path(name) else {
        return Vec::new();
    };
    match read_file(name) {
        Some(content) => {
            let counts =
                classify_file(Classify_File_Input { source: content, language: language.clone() });
            vec![File_Count { path: name.to_string(), language: language.name, counts, is_test: false }]
        }
        None => Vec::new(),
    }
}

/// Reports whether an ignore set excludes a path: an ignored directory is pruned,
/// an ignored file is skipped.
fn ignore_set_ignores(set: &Ignore_Set, path: &str, is_directory: bool) -> bool {
    if is_directory {
        !set.kept_directories.contains(path)
    } else {
        !set.kept_files.contains(path)
    }
}

/// Resolves the scope from the command line: the positional paths (defaulting to
/// the current directory) and the override flags.
fn parse_arguments(arguments: &[String]) -> Main_Scope {
    let rest = arguments.get(1..).unwrap_or(&[]);
    // Flags carry a single leading dash (the shared/cli convention), so a token
    // beginning with one is a flag and anything else is a positional path.
    let paths: Vec<String> =
        rest.iter().filter(|argument| !argument.starts_with('-')).cloned().collect();
    // No path given counts the current directory, the obvious default for a tool
    // run inside a project.
    let paths = if paths.is_empty() { vec![".".to_string()] } else { paths };
    Main_Scope {
        paths,
        no_ignore: rest.iter().any(|argument| argument == "-no-ignore"),
        include_hidden: rest.iter().any(|argument| argument == "-hidden"),
        json: rest.iter().any(|argument| argument == "-json"),
        show_files: rest.iter().any(|argument| argument == "-files"),
    }
}

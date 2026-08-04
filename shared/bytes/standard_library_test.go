// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in LICENSE.bsd3.golang.

package bytes_test

import (
	"encoding/base64"
	"fmt"
	standard_io "io"
	"iter"
	shared_bytes "local/james-orcales/shared/bytes"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/io"
	snap "local/james-orcales/shared/snap/default"
	"local/james-orcales/shared/unicode/ucd"
	"math"
	"math/rand"
	"slices"
	"strconv"
	standard_strings "strings"
	"testing"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// TestMain registers the package invariant roots before the specification runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

const STANDARD_MINIMUM_READ_SIZE = shared_bytes.MINIMUM_READ_SIZE

type standard_buffer struct {
	Port shared_bytes.Buffer
}

func standard_new_buffer(content []byte) (buffer *standard_buffer) {
	return &standard_buffer{Port: *shared_bytes.New_Buffer(content)}
}

func standard_new_buffer_text(content string) (buffer *standard_buffer) {
	return &standard_buffer{Port: *shared_bytes.New_Buffer_Text(shared_bytes.Text(content))}
}

func standard_buffer_port(buffer *standard_buffer) (port *shared_bytes.Buffer) {
	return &buffer.Port
}

func standard_buffer_bytes(buffer *standard_buffer) (content []byte) {
	return []byte(shared_bytes.Buffer_Bytes(standard_buffer_port(buffer)))
}

func standard_buffer_available_slice(buffer *standard_buffer) (available []byte) {
	return []byte(shared_bytes.Buffer_Available_Slice(standard_buffer_port(buffer)))
}

func standard_buffer_string(buffer *standard_buffer) (content string) {
	if buffer == nil {
		return "<nil>"
	}
	return string(shared_bytes.Buffer_String(standard_buffer_port(buffer)))
}

func standard_buffer_peek(
	buffer *standard_buffer,
	count int,
) (content []byte, err error) {
	port_content, port_err := shared_bytes.Buffer_Peek(
		standard_buffer_port(buffer), shared_bytes.Boundary(count),
	)
	return []byte(port_content), standard_stream_error(port_err)
}

func standard_buffer_capacity(buffer *standard_buffer) (capacity int) {
	return int(shared_bytes.Buffer_Capacity(standard_buffer_port(buffer)))
}

func standard_buffer_available(buffer *standard_buffer) (available int) {
	return int(shared_bytes.Buffer_Available(standard_buffer_port(buffer)))
}

func standard_buffer_truncate(buffer *standard_buffer, count int) {
	shared_bytes.Buffer_Truncate(
		standard_buffer_port(buffer), shared_bytes.Boundary(count),
	)
}

func standard_buffer_reset(buffer *standard_buffer) {
	shared_bytes.Buffer_Reset(standard_buffer_port(buffer))
}

func standard_buffer_grow(buffer *standard_buffer, count int) {
	if count > shared_bytes.GROWTH_COUNT_MAXIMUM {
		panic(shared_bytes.Error_Too_Large)
	}
	shared_bytes.Buffer_Grow(
		standard_buffer_port(buffer), shared_bytes.Growth_Count(count),
	)
}

func standard_buffer_write(
	buffer *standard_buffer,
	source []byte,
) (count int, err error) {
	written, write_err := shared_bytes.Buffer_Write(
		standard_buffer_port(buffer),
		shared_bytes.Slice(source),
	)
	return int(written), standard_stream_error(write_err)
}

func standard_buffer_write_text(
	buffer *standard_buffer,
	source string,
) (count int, err error) {
	written, write_err := shared_bytes.Buffer_Write_Text(
		standard_buffer_port(buffer), shared_bytes.Text(source),
	)
	return int(written), write_err
}

func standard_buffer_read_from(
	buffer *standard_buffer,
	source io.Stream,
) (count int64, err error) {
	read_count, read_err := shared_bytes.Buffer_Read_From(
		standard_buffer_port(buffer),
		source,
	)
	if read_err == io.Stream_Negative_Read {
		panic(read_err)
	}
	if read_err == io.Stream_Short_Buffer {
		panic(read_err)
	}
	return int64(read_count), standard_stream_error(read_err)
}

func standard_buffer_write_to(
	buffer *standard_buffer,
	destination io.Stream,
) (count int64, err error) {
	written, write_err := shared_bytes.Buffer_Write_To(
		standard_buffer_port(buffer),
		destination,
	)
	if write_err == io.Stream_Negative_Write {
		panic(write_err)
	}
	if write_err == io.Stream_Invalid_Write {
		panic(write_err)
	}
	return int64(written), standard_stream_error(write_err)
}

func standard_buffer_write_byte(buffer *standard_buffer, value byte) (err error) {
	return shared_bytes.Buffer_Write_Byte(
		standard_buffer_port(buffer), shared_bytes.Byte(value),
	)
}

func standard_buffer_write_character(
	buffer *standard_buffer,
	character rune,
) (count int, err error) {
	written, write_err := shared_bytes.Buffer_Write_Character(
		standard_buffer_port(buffer), shared_bytes.Character(character),
	)
	return int(written), write_err
}

func standard_buffer_read(
	buffer *standard_buffer,
	destination []byte,
) (count int, err error) {
	read_count, read_err := shared_bytes.Buffer_Read(
		standard_buffer_port(buffer),
		shared_bytes.Slice(destination),
	)
	if len(destination) == 0 {
		if read_err == io.Stream_EOF {
			read_err = nil
		}
	}
	return int(read_count), standard_stream_error(read_err)
}

func standard_buffer_next(buffer *standard_buffer, count int) (content []byte) {
	return []byte(shared_bytes.Buffer_Next(
		standard_buffer_port(buffer), shared_bytes.Boundary(count),
	))
}

func standard_buffer_read_bytes(
	buffer *standard_buffer,
	delimiter byte,
) (line []byte, err error) {
	port_line, port_err := shared_bytes.Buffer_Read_Bytes(
		standard_buffer_port(buffer), shared_bytes.Byte(delimiter),
	)
	return []byte(port_line), standard_stream_error(port_err)
}

func standard_buffer_read_text(
	buffer *standard_buffer,
	delimiter byte,
) (line string, err error) {
	port_line, port_err := shared_bytes.Buffer_Read_Text(
		standard_buffer_port(buffer), shared_bytes.Byte(delimiter),
	)
	return string(port_line), standard_stream_error(port_err)
}

type standard_reader struct {
	Port shared_bytes.Reader
}

func standard_new_reader(source []byte) (reader *standard_reader) {
	return &standard_reader{Port: *shared_bytes.New_Reader(source)}
}

func standard_reader_port(reader *standard_reader) (port *shared_bytes.Reader) {
	return &reader.Port
}

func standard_reader_size(reader *standard_reader) (size int64) {
	return int64(shared_bytes.Reader_Size(standard_reader_port(reader)))
}

func standard_reader_read(
	reader *standard_reader,
	destination []byte,
) (count int, err error) {
	read_count, read_err := shared_bytes.Reader_Read(
		standard_reader_port(reader),
		shared_bytes.Slice(destination),
	)
	return int(read_count), standard_stream_error(read_err)
}

func standard_reader_read_at(
	reader *standard_reader,
	destination []byte,
	position int64,
) (count int, err error) {
	read_count, read_err := shared_bytes.Reader_Read_At(
		standard_reader_port(reader),
		shared_bytes.Slice(destination),
		shared_bytes.Reader_Offset(position),
	)
	if read_err == nil {
		if int(read_count) < len(destination) {
			read_err = io.Stream_EOF
		}
	}
	if read_err == io.Stream_Invalid_Offset {
		read_err = io.Stream_EOF
	}
	return int(read_count), standard_stream_error(read_err)
}

func standard_reader_seek(
	reader *standard_reader,
	offset int64, origin int,
) (position int64, err error) {
	port_position, seek_err := shared_bytes.Reader_Seek(
		standard_reader_port(reader),
		shared_bytes.Reader_Offset(offset),
		io.Seek_From(origin),
	)
	return int64(port_position), standard_stream_error(seek_err)
}

func standard_reader_write_to(
	reader *standard_reader,
	destination io.Stream,
) (count int64, err error) {
	written, write_err := shared_bytes.Reader_Write_To(
		standard_reader_port(reader),
		destination,
	)
	if write_err == io.Stream_Negative_Write {
		panic(write_err)
	}
	if write_err == io.Stream_Invalid_Write {
		panic(write_err)
	}
	return int64(written), standard_stream_error(write_err)
}

func standard_reader_reset(reader *standard_reader, source []byte) {
	shared_bytes.Reader_Reset(standard_reader_port(reader), source)
}

type standard_stream_read func(buffer []byte) (count int, err error)

type standard_stream_write func(buffer []byte) (count int, err error)

func standard_input_stream(read standard_stream_read) (stream io.Stream) {
	return io.Stream{Procedure: standard_input_stream_procedure, Data: read}
}

func standard_output_stream(write standard_stream_write) (stream io.Stream) {
	return io.Stream{Procedure: standard_output_stream_procedure, Data: write}
}

func standard_input_stream_procedure(
	data any,
	mode io.Stream_Mode,
	buffer []byte,
	_ int64,
	_ io.Seek_From,
) (count int64, err error) {
	if mode == io.STREAM_MODE_QUERY {
		return int64(
			1<<io.STREAM_MODE_QUERY | 1<<io.STREAM_MODE_READ,
		), nil
	}
	if mode != io.STREAM_MODE_READ {
		return 0, io.Stream_Empty
	}
	read, held := data.(standard_stream_read)
	if !held {
		panic("a standard input stream must hold its read function")
	}
	read_count, read_err := read(buffer)
	return int64(read_count), shared_stream_error(read_err)
}

func standard_output_stream_procedure(
	data any,
	mode io.Stream_Mode,
	buffer []byte,
	_ int64,
	_ io.Seek_From,
) (count int64, err error) {
	if mode == io.STREAM_MODE_QUERY {
		return int64(
			1<<io.STREAM_MODE_QUERY | 1<<io.STREAM_MODE_WRITE,
		), nil
	}
	if mode != io.STREAM_MODE_WRITE {
		return 0, io.Stream_Empty
	}
	write, held := data.(standard_stream_write)
	if !held {
		panic("a standard output stream must hold its write function")
	}
	write_count, write_err := write(buffer)
	return int64(write_count), shared_stream_error(write_err)
}

func shared_stream_error(err error) (stream_err error) {
	if err == standard_io.EOF {
		return io.Stream_EOF
	}
	if err == standard_io.ErrUnexpectedEOF {
		return io.Stream_Unexpected_EOF
	}
	if err == standard_io.ErrShortWrite {
		return io.Stream_Short_Write
	}
	if err == standard_io.ErrNoProgress {
		return io.Stream_No_Progress
	}
	return err
}

func standard_stream_error(err error) (standard_err error) {
	if err == io.Stream_EOF {
		return standard_io.EOF
	}
	if err == io.Stream_Unexpected_EOF {
		return standard_io.ErrUnexpectedEOF
	}
	if err == io.Stream_Short_Write {
		return standard_io.ErrShortWrite
	}
	if err == io.Stream_No_Progress {
		return standard_io.ErrNoProgress
	}
	return err
}

func standard_unread_size(subject any) (size int) {
	switch value := any(subject).(type) {
	case *standard_buffer:
		return int(shared_bytes.Buffer_Size(standard_buffer_port(value)))
	case *standard_reader:
		return int(shared_bytes.Reader_Unread_Size(standard_reader_port(value)))
	}
	panic("unsupported standard unread subject")
}

func standard_read_byte(subject any) (value byte, err error) {
	switch source := any(subject).(type) {
	case *standard_buffer:
		read_value, read_err := shared_bytes.Buffer_Read_Byte(standard_buffer_port(source))
		return byte(read_value), standard_stream_error(read_err)
	case *standard_reader:
		read_value, read_err := shared_bytes.Reader_Read_Byte(standard_reader_port(source))
		return byte(read_value), standard_stream_error(read_err)
	}
	panic("unsupported standard read subject")
}

func standard_read_character(
	subject any,
) (character rune, size int, err error) {
	switch source := any(subject).(type) {
	case *standard_buffer:
		value, width, read_err := shared_bytes.Buffer_Read_Character(
			standard_buffer_port(source),
		)
		return rune(value), int(width), standard_stream_error(read_err)
	case *standard_reader:
		value, width, read_err := shared_bytes.Reader_Read_Character(
			standard_reader_port(source),
		)
		return rune(value), int(width), standard_stream_error(read_err)
	}
	panic("unsupported standard read subject")
}

func standard_unread_character(subject any) (err error) {
	switch source := any(subject).(type) {
	case *standard_buffer:
		return shared_bytes.Buffer_Unread_Character(standard_buffer_port(source))
	case *standard_reader:
		return shared_bytes.Reader_Unread_Character(standard_reader_port(source))
	}
	panic("unsupported standard unread subject")
}

func standard_unread_byte(subject any) (err error) {
	switch source := any(subject).(type) {
	case *standard_buffer:
		return shared_bytes.Buffer_Unread_Byte(standard_buffer_port(source))
	case *standard_reader:
		return shared_bytes.Reader_Unread_Byte(standard_reader_port(source))
	}
	panic("unsupported standard unread subject")
}

func standard_clone(source []byte) (clone []byte) {
	return []byte(shared_bytes.Clone(source))
}

func standard_compare(left []byte, right []byte) (order int) {
	return int(shared_bytes.Compare(left, right))
}

func standard_contains(source []byte, separator []byte) (contained bool) {
	return bool(shared_bytes.Contains(source, separator))
}

func standard_contains_any(source []byte, characters string) (contained bool) {
	return bool(shared_bytes.Contains_Any(source, shared_bytes.Text(characters)))
}

func standard_contains_function(
	source []byte, predicate func(rune) (result_1 bool),
) (contained bool) {
	return bool(shared_bytes.Contains_Function(source, predicate))
}

func standard_contains_rune(source []byte, character rune) (contained bool) {
	return bool(shared_bytes.Contains_Rune(source, shared_bytes.Character(character)))
}

func standard_count(source []byte, separator []byte) (count int) {
	return int(shared_bytes.Count(source, separator))
}

func standard_cut(
	source []byte, separator []byte,
) (before []byte, after []byte, found bool) {
	port_before, port_after, port_found := shared_bytes.Cut(source, separator)
	return []byte(port_before), []byte(port_after), bool(port_found)
}

func standard_cut_prefix(
	source []byte, prefix []byte,
) (after []byte, found bool) {
	port_after, port_found := shared_bytes.Cut_Prefix(source, prefix)
	return []byte(port_after), bool(port_found)
}

func standard_cut_suffix(
	source []byte, suffix []byte,
) (before []byte, found bool) {
	port_before, port_found := shared_bytes.Cut_Suffix(source, suffix)
	return []byte(port_before), bool(port_found)
}

func standard_equal(left []byte, right []byte) (equal bool) {
	return bool(shared_bytes.Equal(left, right))
}

func standard_equal_fold(left []byte, right []byte) (equal bool) {
	return bool(shared_bytes.Equal_Fold(left, right))
}

func standard_fields(source []byte) (fields [][]byte) {
	return [][]byte(shared_bytes.Fields(source))
}

func standard_fields_function(
	source []byte, predicate func(rune) (result_2 bool),
) (fields [][]byte) {
	return [][]byte(shared_bytes.Fields_Function(source, predicate))
}

func standard_has_prefix(source []byte, prefix []byte) (present bool) {
	return bool(shared_bytes.Has_Prefix(source, prefix))
}

func standard_has_suffix(source []byte, suffix []byte) (present bool) {
	return bool(shared_bytes.Has_Suffix(source, suffix))
}

func standard_index(source []byte, separator []byte) (index int) {
	return int(shared_bytes.Index(source, separator))
}

func standard_index_any(source []byte, characters string) (index int) {
	return int(shared_bytes.Index_Any(source, shared_bytes.Text(characters)))
}

func standard_index_byte(source []byte, value byte) (index int) {
	return int(shared_bytes.Index_Byte(source, shared_bytes.Byte(value)))
}

func standard_index_function(source []byte, predicate func(rune) (result_3 bool)) (index int) {
	return int(shared_bytes.Index_Function(source, predicate))
}

func standard_index_rune(source []byte, character rune) (index int) {
	return int(shared_bytes.Index_Rune(source, shared_bytes.Character(character)))
}

func standard_join(parts [][]byte, separator []byte) (joined []byte) {
	return []byte(shared_bytes.Join(shared_bytes.Slices(parts), separator))
}

func standard_last_index(source []byte, separator []byte) (index int) {
	return int(shared_bytes.Last_Index(source, separator))
}

func standard_last_index_any(source []byte, characters string) (index int) {
	return int(shared_bytes.Last_Index_Any(source, shared_bytes.Text(characters)))
}

func standard_last_index_byte(source []byte, value byte) (index int) {
	return int(shared_bytes.Last_Index_Byte(source, shared_bytes.Byte(value)))
}

func standard_last_index_function(
	source []byte, predicate func(rune) (result_4 bool),
) (index int) {
	return int(shared_bytes.Last_Index_Function(source, predicate))
}

func standard_lines(source []byte) (sequence iter.Seq[[]byte]) {
	return standard_sequence(shared_bytes.Lines(source))
}

func standard_map(mapping func(rune) (result_5 rune), source []byte) (mapped []byte) {
	return []byte(shared_bytes.Map(mapping, source))
}

func standard_repeat(source []byte, count int) (repeated []byte) {
	return []byte(shared_bytes.Repeat(source, shared_bytes.Repeat_Count(count)))
}

func standard_replace(
	source []byte, old []byte, replacement []byte, count int,
) (replaced []byte) {
	return []byte(shared_bytes.Replace(
		source, old, replacement, shared_bytes.Replacement_Count(count),
	))
}

func standard_replace_all(
	source []byte, old []byte, replacement []byte,
) (replaced []byte) {
	return []byte(shared_bytes.Replace_All(source, old, replacement))
}

func standard_runes(source []byte) (characters []rune) {
	return []rune(shared_bytes.Runes(source))
}

func standard_split(source []byte, separator []byte) (parts [][]byte) {
	return [][]byte(shared_bytes.Split(source, separator))
}

func standard_split_after(source []byte, separator []byte) (parts [][]byte) {
	return [][]byte(shared_bytes.Split_After(source, separator))
}

func standard_split_after_n(
	source []byte, separator []byte, limit int,
) (parts [][]byte) {
	if limit > shared_bytes.LIMIT_MAXIMUM {
		limit = shared_bytes.LIMIT_MAXIMUM
	}
	return [][]byte(shared_bytes.Split_After_N(
		source, separator, shared_bytes.Limit(limit),
	))
}

func standard_split_n(
	source []byte, separator []byte, limit int,
) (parts [][]byte) {
	if limit > shared_bytes.LIMIT_MAXIMUM {
		limit = shared_bytes.LIMIT_MAXIMUM
	}
	return [][]byte(shared_bytes.Split_N(source, separator, shared_bytes.Limit(limit)))
}

func standard_title(source []byte) (title []byte) {
	return []byte(shared_bytes.Title(source))
}

func standard_to_lower(source []byte) (lower []byte) {
	return []byte(shared_bytes.To_Lower(source))
}

func standard_to_lower_special(
	special ucd.Special_Case, source []byte,
) (lower []byte) {
	return []byte(shared_bytes.To_Lower_Special(special, source))
}

func standard_to_title(source []byte) (title []byte) {
	return []byte(shared_bytes.To_Title(source))
}

func standard_to_title_special(
	special ucd.Special_Case, source []byte,
) (title []byte) {
	return []byte(shared_bytes.To_Title_Special(special, source))
}

func standard_to_upper(source []byte) (upper []byte) {
	return []byte(shared_bytes.To_Upper(source))
}

func standard_to_upper_special(
	special ucd.Special_Case, source []byte,
) (upper []byte) {
	return []byte(shared_bytes.To_Upper_Special(special, source))
}

func standard_to_valid_utf8(source []byte, replacement []byte) (valid []byte) {
	return []byte(shared_bytes.To_Valid_UTF8(source, replacement))
}

func standard_trim(source []byte, cutset string) (trimmed []byte) {
	return []byte(shared_bytes.Trim(source, shared_bytes.Text(cutset)))
}

func standard_trim_function(source []byte, predicate func(rune) (result_6 bool)) (trimmed []byte) {
	return []byte(shared_bytes.Trim_Function(source, predicate))
}

func standard_trim_left(source []byte, cutset string) (trimmed []byte) {
	return []byte(shared_bytes.Trim_Left(source, shared_bytes.Text(cutset)))
}

func standard_trim_left_function(
	source []byte, predicate func(rune) (result_7 bool),
) (trimmed []byte) {
	return []byte(shared_bytes.Trim_Left_Function(source, predicate))
}

func standard_trim_prefix(source []byte, prefix []byte) (trimmed []byte) {
	return []byte(shared_bytes.Trim_Prefix(source, prefix))
}

func standard_trim_right(source []byte, cutset string) (trimmed []byte) {
	return []byte(shared_bytes.Trim_Right(source, shared_bytes.Text(cutset)))
}

func standard_trim_right_function(
	source []byte, predicate func(rune) (result_8 bool),
) (trimmed []byte) {
	return []byte(shared_bytes.Trim_Right_Function(source, predicate))
}

func standard_trim_space(source []byte) (trimmed []byte) {
	return []byte(shared_bytes.Trim_Space(source))
}

func standard_trim_suffix(source []byte, suffix []byte) (trimmed []byte) {
	return []byte(shared_bytes.Trim_Suffix(source, suffix))
}

func standard_split_sequence(
	source []byte, separator []byte,
) (sequence iter.Seq[[]byte]) {
	return standard_sequence(shared_bytes.Split_Sequence(source, separator))
}

func standard_split_after_sequence(
	source []byte, separator []byte,
) (sequence iter.Seq[[]byte]) {
	return standard_sequence(shared_bytes.Split_After_Sequence(source, separator))
}

func standard_fields_sequence(source []byte) (sequence iter.Seq[[]byte]) {
	return standard_sequence(shared_bytes.Fields_Sequence(source))
}

func standard_fields_function_sequence(
	source []byte, predicate func(rune) (result_9 bool),
) (sequence iter.Seq[[]byte]) {
	return standard_sequence(shared_bytes.Fields_Function_Sequence(source, predicate))
}

func standard_sequence(source iter.Seq[shared_bytes.Slice]) (sequence iter.Seq[[]byte]) {
	return func(yield func([]byte) (result_10 bool)) {
		for value := range source {
			if !yield([]byte(value)) {
				return
			}
		}
	}
}

const TEST_CONTENT_SIZE = 512 // The bound keeps five repeated writes below one Slice.

func negative_reader_stream() (stream io.Stream) {
	return standard_input_stream(func([]byte) (count int, err error) {
		return -1, nil
	})
}

func test_bytes() (content []byte) {
	content = make([]byte, TEST_CONTENT_SIZE)
	for i_index := 0; i_index < TEST_CONTENT_SIZE; i_index++ {
		content[i_index] = 'a' + byte(i_index%26)
	}
	return content
}

func test_string() (content string) {
	return string(test_bytes())
}

// Verify that contents of buf match the string s.
func check(t *testing.T, testname string, buffer *standard_buffer, s string) {
	bytes := standard_buffer_bytes(buffer)
	string_value := standard_buffer_string(buffer)
	if standard_unread_size(buffer) != len(bytes) {
		t.Errorf(
			"%s: buf.Len() == %d, len(buf.Bytes()) == %d",
			testname,
			standard_unread_size(buffer),
			len(bytes),
		)
	}

	if standard_unread_size(buffer) != len(string_value) {
		t.Errorf(
			"%s: buf.Len() == %d, len(buf.String()) == %d",
			testname,
			standard_unread_size(buffer),
			len(string_value),
		)
	}

	if standard_unread_size(buffer) != len(s) {
		t.Errorf(
			"%s: buf.Len() == %d, len(s) == %d",
			testname,
			standard_unread_size(buffer),
			len(s),
		)
	}

	if string(bytes) != s {
		t.Errorf("%s: string(buf.Bytes()) == %q, s == %q", testname, string(bytes), s)
	}
}

// Fill buf through n writes of string fus.
// The initial contents of buf corresponds to the string s;
// the result is the final contents of buf returned as a string.
func fill_string(
	t *testing.T,
	testname string,
	buffer *standard_buffer,
	s string,
	n int,
	fus string,
) (result_13 string) {
	check(t, testname+" (fill 1)", buffer, s)
	for ; n > 0; n-- {
		m, err := standard_buffer_write_text(buffer, fus)
		if m != len(fus) {
			t.Errorf(testname+" (fill 2): m == %d, expected %d", m, len(fus))
		}
		if err != nil {
			t.Errorf(
				testname+" (fill 3): err should always be nil, found err == %s",
				err,
			)
		}
		s += fus
		check(t, testname+" (fill 4)", buffer, s)
	}
	return s
}

// Fill buf through n writes of byte slice fub.
// The initial contents of buf corresponds to the string s;
// the result is the final contents of buf returned as a string.
func fill_bytes(
	t *testing.T,
	testname string,
	buffer *standard_buffer,
	s string,
	n int,
	fub []byte,
) (result_14 string) {
	check(t, testname+" (fill 1)", buffer, s)
	for ; n > 0; n-- {
		m, err := standard_buffer_write(buffer, fub)
		if m != len(fub) {
			t.Errorf(testname+" (fill 2): m == %d, expected %d", m, len(fub))
		}
		if err != nil {
			t.Errorf(
				testname+" (fill 3): err should always be nil, found err == %s",
				err,
			)
		}
		s += string(fub)
		check(t, testname+" (fill 4)", buffer, s)
	}
	return s
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_New_Buffer(t *testing.T) {
	buffer := standard_new_buffer(test_bytes())
	check(t, "standard_new_buffer", buffer, test_string())
}

// The shallow copy resets the underlying Slice of an existing standard_buffer.
func Test_Standard_Library_New_Buffer_Shallow(t *testing.T) {
	shallow_buffer := *standard_new_buffer(test_bytes())
	check(t, "standard_new_buffer", &shallow_buffer, test_string())
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_New_Buffer_String(t *testing.T) {
	buffer := standard_new_buffer_text(test_string())
	check(t, "standard_new_buffer_text", buffer, test_string())
}

// Empty buf through repeated reads into fub.
// The initial contents of buf corresponds to the string s.
func empty(t *testing.T, testname string, buffer *standard_buffer, s string, fub []byte) {
	check(t, testname+" (empty 1)", buffer, s)

	for standard_unread_size(buffer) > 0 {
		n, err := standard_buffer_read(buffer, fub)
		if n == 0 {
			break
		}
		if err != nil {
			t.Errorf(
				testname+" (empty 2): err should always be nil, found err == %s",
				err,
			)
		}
		s = s[n:]
		check(t, testname+" (empty 3)", buffer, s)
	}

	check(t, testname+" (empty 4)", buffer, "")
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Basic_Operations(t *testing.T) {
	var buffer standard_buffer

	for i_index := 0; i_index < 5; i_index++ {
		check(t, "TestBasicOperations (1)", &buffer, "")

		standard_buffer_reset(&buffer)
		check(t, "TestBasicOperations (2)", &buffer, "")

		standard_buffer_truncate(&buffer, 0)
		check(t, "TestBasicOperations (3)", &buffer, "")

		n, err := standard_buffer_write(&buffer, test_bytes()[0:1])
		if want := 1; err != nil {
			t.Errorf("Write: got (%d, %v), want (%d, %v)", n, err, want, nil)
		} else if n != want {
			t.Errorf("Write: got (%d, %v), want (%d, %v)", n, err, want, nil)
		}
		check(t, "TestBasicOperations (4)", &buffer, "a")

		standard_buffer_write_byte(&buffer, test_string()[1])
		check(t, "TestBasicOperations (5)", &buffer, "ab")

		n, err = standard_buffer_write(&buffer, test_bytes()[2:26])
		if want := 24; err != nil {
			t.Errorf("Write: got (%d, %v), want (%d, %v)", n, err, want, nil)
		} else if n != want {
			t.Errorf("Write: got (%d, %v), want (%d, %v)", n, err, want, nil)
		}
		check(t, "TestBasicOperations (6)", &buffer, test_string()[0:26])

		standard_buffer_truncate(&buffer, 26)
		check(t, "TestBasicOperations (7)", &buffer, test_string()[0:26])

		standard_buffer_truncate(&buffer, 20)
		check(t, "TestBasicOperations (8)", &buffer, test_string()[0:20])

		empty(t, "TestBasicOperations (9)", &buffer, test_string()[0:20], make([]byte, 5))
		empty(t, "TestBasicOperations (10)", &buffer, "", make([]byte, 100))

		standard_buffer_write_byte(&buffer, test_string()[1])
		c, err := standard_read_byte(&buffer)
		if want := test_string()[1]; err != nil {
			t.Errorf("ReadByte: got (%q, %v), want (%q, %v)", c, err, want, nil)
		} else if c != want {
			t.Errorf("ReadByte: got (%q, %v), want (%q, %v)", c, err, want, nil)
		}
		c, err = standard_read_byte(&buffer)
		if err != standard_io.EOF {
			t.Errorf(
				"ReadByte: got (%q, %v), want (%q, %v)",
				c, err, byte(0), standard_io.EOF,
			)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Large_String_Writes(t *testing.T) {
	var buffer standard_buffer
	limit := 30
	if testing.Short() {
		limit = 9
	}
	for i := 3; i < limit; i += 3 {
		s := fill_string(t, "TestLargeWrites (1)", &buffer, "", 5, test_string())
		empty(
			t,
			"TestLargeStringWrites (2)",
			&buffer,
			s,
			make([]byte, len(test_string())/i),
		)
	}
	check(t, "TestLargeStringWrites (3)", &buffer, "")
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Large_Byte_Writes(t *testing.T) {
	var buffer standard_buffer
	limit := 30
	if testing.Short() {
		limit = 9
	}
	for i := 3; i < limit; i += 3 {
		s := fill_bytes(t, "TestLargeWrites (1)", &buffer, "", 5, test_bytes())
		empty(t, "TestLargeByteWrites (2)", &buffer, s, make([]byte, len(test_string())/i))
	}
	check(t, "TestLargeByteWrites (3)", &buffer, "")
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Large_String_Reads(t *testing.T) {
	var buffer standard_buffer
	for i := 3; i < 30; i += 3 {
		s := fill_string(
			t,
			"TestLargeReads (1)",
			&buffer,
			"",
			5,
			test_string()[:len(test_string())/i],
		)
		empty(t, "TestLargeReads (2)", &buffer, s, make([]byte, len(test_string())))
	}
	check(t, "TestLargeStringReads (3)", &buffer, "")
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Large_Byte_Reads(t *testing.T) {
	var buffer standard_buffer
	for i := 3; i < 30; i += 3 {
		s := fill_bytes(
			t,
			"TestLargeReads (1)",
			&buffer,
			"",
			5,
			test_bytes()[:len(test_bytes())/i],
		)
		empty(t, "TestLargeReads (2)", &buffer, s, make([]byte, len(test_string())))
	}
	check(t, "TestLargeByteReads (3)", &buffer, "")
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Mixed_Reads_And_Writes(t *testing.T) {
	var buffer standard_buffer
	s := ""
	for i_index := 0; i_index < 50; i_index++ {
		wlen := rand.Intn(len(test_string()))
		if i_index%2 == 0 {
			s = fill_string(
				t,
				"TestMixedReadsAndWrites (1)",
				&buffer,
				s,
				1,
				test_string()[0:wlen],
			)
		} else {
			s = fill_bytes(
				t,
				"TestMixedReadsAndWrites (1)",
				&buffer,
				s,
				1,
				test_bytes()[0:wlen],
			)
		}

		rlen_size := rand.Intn(len(test_string()))
		fub := make([]byte, rlen_size)
		n, _ := standard_buffer_read(&buffer, fub)
		s = s[n:]
	}
	empty(
		t,
		"TestMixedReadsAndWrites (2)",
		&buffer,
		s,
		make([]byte, standard_unread_size(&buffer)),
	)
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Cap_With_Preallocated_Slice(t *testing.T) {
	buffer := standard_new_buffer(make([]byte, 10))
	n_size := standard_buffer_capacity(buffer)
	if n_size != 10 {
		t.Errorf("expected 10, got %d", n_size)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Cap_With_Slice_And_Written_Data(t *testing.T) {
	buffer := standard_new_buffer(make([]byte, 0, 10))
	standard_buffer_write(buffer, []byte("test"))
	n_size := standard_buffer_capacity(buffer)
	if n_size != 10 {
		t.Errorf("expected 10, got %d", n_size)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Nil(t *testing.T) {
	var b *standard_buffer
	if standard_buffer_string(b) != "<nil>" {
		t.Errorf("expected <nil>; got %q", standard_buffer_string(b))
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Read_From(t *testing.T) {
	var buffer standard_buffer
	for i := 3; i < 30; i += 3 {
		s := fill_bytes(
			t,
			"TestReadFrom (1)",
			&buffer,
			"",
			5,
			test_bytes()[:len(test_bytes())/i],
		)
		var b standard_buffer
		standard_buffer_read_from(
			&b,
			shared_bytes.Buffer_To_Stream(standard_buffer_port(&buffer)),
		)
		empty(t, "TestReadFrom (2)", &b, s, make([]byte, len(test_string())))
	}
}

func panic_reader_stream(should_panic bool) (stream io.Stream) {
	return standard_input_stream(func([]byte) (count int, err error) {
		if should_panic {
			panic("oops")
		}
		return 0, standard_io.EOF
	})
}

// This regression keeps a failed Read from changing an empty Buffer.
func Test_Standard_Library_Read_From_Panic_Reader(t *testing.T) {

	// The first read establishes the state before the panic path.
	var buffer standard_buffer
	i, err := standard_buffer_read_from(&buffer, panic_reader_stream(false))
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Fatalf("unexpected return from bytes.ReadFrom (1): got: %d, want %d", i, 0)
	}
	check(t, "TestReadFromPanicReader (1)", &buffer, "")

	// The empty state must survive a Reader panic.
	var buf2 standard_buffer
	defer func() {
		recover()
		check(t, "TestReadFromPanicReader (2)", &buf2, "")
	}()
	standard_buffer_read_from(&buf2, panic_reader_stream(true))
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Read_From_Negative_Reader(t *testing.T) {
	var b standard_buffer
	defer func() {
		switch err := recover().(type) {
		case nil:
			t.Fatal("bytes.Buffer.ReadFrom didn't panic")
		case error:

			want_error := io.Stream_Negative_Read.Error()
			if err.Error() != want_error {
				t.Fatalf(
					"recovered panic: got %v, want %v",
					err.Error(),
					want_error,
				)
			}
		default:
			t.Fatalf("unexpected panic value: %#v", err)
		}
	}()

	standard_buffer_read_from(&b, negative_reader_stream())
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Write_To(t *testing.T) {
	var buffer standard_buffer
	for i := 3; i < 30; i += 3 {
		s := fill_bytes(
			t,
			"TestWriteTo (1)",
			&buffer,
			"",
			5,
			test_bytes()[:len(test_bytes())/i],
		)
		var b standard_buffer
		standard_buffer_write_to(
			&buffer,
			shared_bytes.Buffer_To_Stream(standard_buffer_port(&b)),
		)
		empty(t, "TestWriteTo (2)", &b, s, make([]byte, len(test_string())))
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Write_Append(t *testing.T) {
	var got standard_buffer
	var want []byte
	for i_index := 0; i_index < 1000; i_index++ {
		b := standard_buffer_available_slice(&got)
		b = strconv.AppendInt(b, int64(i_index), 10)
		want = strconv.AppendInt(want, int64(i_index), 10)
		standard_buffer_write(&got, b)
	}
	if !standard_equal(standard_buffer_bytes(&got), want) {
		t.Fatalf("Bytes() = %q, want %q", &got, want)
	}

	standard_buffer_reset(&got)
	for i_index := 0; i_index < 1000; i_index++ {
		b := standard_buffer_available_slice(&got)
		b = strconv.AppendInt(b, int64(i_index), 10)
		standard_buffer_write(&got, b)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Rune_IO(t *testing.T) {
	const N_RUNE = 1000

	b := make([]byte, utf8.UTFMax*N_RUNE)
	var buffer standard_buffer
	n := 0
	for r := rune(0); r < N_RUNE; r++ {
		size := utf8.EncodeRune(b[n:], r)
		nbytes, err := standard_buffer_write_character(&buffer, r)
		if err != nil {
			t.Fatalf("WriteRune(%U) error: %s", r, err)
		}
		if nbytes != size {
			t.Fatalf("WriteRune(%U) expected %d, got %d", r, size, nbytes)
		}
		n += size
	}
	b = b[0:n]

	if !standard_equal(standard_buffer_bytes(&buffer), b) {
		t.Fatalf(
			"incorrect result from WriteRune: %q not %q",
			standard_buffer_bytes(&buffer),
			b,
		)
	}

	assert_rune_reads(t, &buffer, N_RUNE)

	standard_buffer_reset(&buffer)

	if err := standard_unread_character(&buffer); err == nil {
		t.Fatal("UnreadRune at EOF: got no error")
	}
	if _, _, err := standard_read_character(&buffer); err == nil {
		t.Fatal("ReadRune at EOF: got no error")
	}
	if err := standard_unread_character(&buffer); err == nil {
		t.Fatal("UnreadRune after ReadRune at EOF: got no error")
	}

	standard_buffer_write(&buffer, b)
	assert_unread_rune_reads(t, &buffer, N_RUNE)
}

func assert_rune_reads(t *testing.T, buffer *standard_buffer, rune_count rune) {
	p := make([]byte, utf8.UTFMax)
	for r := rune(0); r < rune_count; r++ {
		size := utf8.EncodeRune(p, r)
		actual, actual_size, err := standard_read_character(buffer)
		if actual != r {
			fail_rune_read(t, r, actual, actual_size, size, err)
		}
		if actual_size != size {
			fail_rune_read(t, r, actual, actual_size, size, err)
		}
		if err != nil {
			fail_rune_read(t, r, actual, actual_size, size, err)
		}
	}
}

func fail_rune_read(
	t *testing.T,
	want rune,
	actual rune,
	actual_size int,
	want_size int,
	err error,
) {
	t.Fatalf(
		"ReadRune(%U) got %U,%d not %U,%d (err=%s)",
		want,
		actual,
		actual_size,
		want,
		want_size,
		err,
	)
}

func assert_unread_rune_reads(t *testing.T, buffer *standard_buffer, rune_count rune) {
	for r := rune(0); r < rune_count; r++ {
		first, size, _ := standard_read_character(buffer)
		if err := standard_unread_character(buffer); err != nil {
			t.Fatalf("UnreadRune(%U) got error %q", r, err)
		}
		actual, actual_size, err := standard_read_character(buffer)
		if first != actual {
			fail_unread_rune_read(t, r, actual, actual_size, size, err)
		}
		if first != r {
			fail_unread_rune_read(t, r, actual, actual_size, size, err)
		}
		if actual_size != size {
			fail_unread_rune_read(t, r, actual, actual_size, size, err)
		}
		if err != nil {
			fail_unread_rune_read(t, r, actual, actual_size, size, err)
		}
	}
}

func fail_unread_rune_read(
	t *testing.T,
	want rune,
	actual rune,
	actual_size int,
	want_size int,
	err error,
) {
	t.Fatalf(
		"ReadRune(%U) after UnreadRune got %U,%d not %U,%d (err=%s)",
		want,
		actual,
		actual_size,
		want,
		want_size,
		err,
	)
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Write_Invalid_Rune(t *testing.T) {

	for _, r := range []rune{-1, utf8.MaxRune + 1} {
		var buffer standard_buffer
		standard_buffer_write_character(&buffer, r)
		check(t, fmt.Sprintf("TestWriteInvalidRune (%d)", r), &buffer, "\uFFFD")
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Next(t *testing.T) {
	b := []byte{0, 1, 2, 3, 4}
	temporary := make([]byte, 5)
	for i := 0; i <= 5; i++ {
		for j := i; j <= 5; j++ {
			for k := 0; k <= 6; k++ {

				buffer := standard_new_buffer(b[0:j])
				n, _ := standard_buffer_read(buffer, temporary[0:i])
				if n != i {
					t.Fatalf("Read %d returned %d", i, n)
				}
				bb := standard_buffer_next(buffer, k)
				want := k
				if want > j-i {
					want = j - i
				}
				if len(bb) != want {
					t.Fatalf("in %d,%d: len(Next(%d)) == %d", i, j, k, len(bb))
				}
				for l, v := range bb {
					if v != byte(l+i) {
						t.Fatalf(
							"in %d,%d: Next(%d)[%d] = %d, want %d",
							i,
							j,
							k,
							l,
							v,
							l+i,
						)
					}
				}
			}
		}
	}
}

func read_bytes_tests() (tests []struct {
	Buffer   string
	Delim    byte
	Expected []string
	Err      error
}) {
	return []struct {
		Buffer   string
		Delim    byte
		Expected []string
		Err      error
	}{
		{"", 0, []string{""}, standard_io.EOF},
		{"a\x00", 0, []string{"a\x00"}, nil},
		{"abbbaaaba", 'b', []string{"ab", "b", "b", "aaab"}, nil},
		{"hello\x01world", 1, []string{"hello\x01"}, nil},
		{"foo\nbar", 0, []string{"foo\nbar"}, standard_io.EOF},
		{"alpha\nbeta\ngamma\n", '\n', []string{"alpha\n", "beta\n", "gamma\n"}, nil},
		{
			"alpha\nbeta\ngamma", '\n',
			[]string{"alpha\n", "beta\n", "gamma"}, standard_io.EOF,
		},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Read_Bytes(t *testing.T) {
	for _, test := range read_bytes_tests() {
		buffer := standard_new_buffer_text(test.Buffer)
		var err error
		for _, expected := range test.Expected {
			var bytes []byte
			bytes, err = standard_buffer_read_bytes(buffer, test.Delim)
			if string(bytes) != expected {
				t.Errorf("expected %q, got %q", expected, bytes)
			}
			if err != nil {
				break
			}
		}
		if err != test.Err {
			t.Errorf("expected error %v, got %v", test.Err, err)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Read_String(t *testing.T) {
	for _, test := range read_bytes_tests() {
		buffer := standard_new_buffer_text(test.Buffer)
		var err error
		for _, expected := range test.Expected {
			var s string
			s, err = standard_buffer_read_text(buffer, test.Delim)
			if s != expected {
				t.Errorf("expected %q, got %q", expected, s)
			}
			if err != nil {
				break
			}
		}
		if err != test.Err {
			t.Errorf("expected error %v, got %v", test.Err, err)
		}
	}
}

func peek_tests() (tests []struct {
	Buffer   string
	Skip     int
	N        int
	Expected string
	Err      error
}) {
	return []struct {
		Buffer   string
		Skip     int
		N        int
		Expected string
		Err      error
	}{
		{"", 0, 0, "", nil},
		{"aaa", 0, 3, "aaa", nil},
		{"foobar", 0, 2, "fo", nil},
		{"a", 0, 2, "a", standard_io.EOF},
		{"helloworld", 4, 3, "owo", nil},
		{"helloworld", 5, 5, "world", nil},
		{"helloworld", 5, 6, "world", standard_io.EOF},
		{"helloworld", 10, 1, "", standard_io.EOF},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Peek(t *testing.T) {
	for _, test := range peek_tests() {
		buffer := standard_new_buffer_text(test.Buffer)
		standard_buffer_next(buffer, test.Skip)
		bytes, err := standard_buffer_peek(buffer, test.N)
		if string(bytes) != test.Expected {
			t.Errorf("expected %q, got %q", test.Expected, bytes)
		}
		if err != test.Err {
			t.Errorf("expected error %v, got %v", test.Err, err)
		}
		if standard_unread_size(buffer) != len(test.Buffer)-test.Skip {
			t.Errorf(
				"bad length after peek: %d, want %d",
				standard_unread_size(buffer),
				len(test.Buffer)-test.Skip,
			)
		}
	}
}

func Benchmark_Standard_Library_Read_String(b *testing.B) {
	const N_SIZE = 4 << 10

	data := make([]byte, N_SIZE)
	data[N_SIZE-1] = 'x'
	b.SetBytes(int64(N_SIZE))
	for i_index := 0; i_index < b.N; i_index++ {
		buffer := standard_new_buffer(data)
		_, err := standard_buffer_read_text(buffer, 'x')
		if err != nil {
			b.Fatal(err)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Grow(t *testing.T) {
	x := []byte{'x'}
	y := []byte{'y'}
	temporary := make([]byte, 72)
	for _, grow_size := range []int{0, 100, 1000, 2000} {
		for _, start_size := range []int{0, 100, 1000, 2000} {
			x_bytes := standard_repeat(x, start_size)

			buffer := standard_new_buffer(x_bytes)

			read_bytes, _ := standard_buffer_read(buffer, temporary)
			y_bytes := standard_repeat(y, grow_size)
			standard_buffer_grow(buffer, grow_size)
			standard_buffer_write(buffer, y_bytes)

			if !standard_equal(
				standard_buffer_bytes(buffer)[0:start_size-read_bytes],
				x_bytes[read_bytes:],
			) {
				t.Errorf("bad initial data at %d %d", start_size, grow_size)
			}
			written_start := start_size - read_bytes
			written_end := written_start + grow_size
			if !standard_equal(
				standard_buffer_bytes(buffer)[written_start:written_end],
				y_bytes,
			) {
				t.Errorf("bad written data at %d %d", start_size, grow_size)
			}
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Grow_Overflow(t *testing.T) {
	defer func() {
		if err := recover(); err != shared_bytes.Error_Too_Large {
			t.Errorf(
				"after too-large Grow, recover() = %v; want %v",
				err,
				shared_bytes.Error_Too_Large,
			)
		}
	}()

	buffer := standard_new_buffer(make([]byte, 1))
	const INT_MAX = int(^uint(0) >> 1)
	standard_buffer_grow(buffer, INT_MAX)
}

// Was a bug: used to give EOF reading empty slice at EOF.
func Test_Standard_Library_Read_Empty_At_EOF(t *testing.T) {
	b := new(standard_buffer)
	destination := make([]byte, 0)
	n, err := standard_buffer_read(b, destination)
	if err != nil {
		t.Errorf("read error: %v", err)
	}
	if n != 0 {
		t.Errorf("wrong count; got %d want 0", n)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Unread_Byte(t *testing.T) {
	b := new(standard_buffer)

	if err := standard_unread_byte(b); err == nil {
		t.Fatal("UnreadByte at EOF: got no error")
	}
	if _, err := standard_read_byte(b); err == nil {
		t.Fatal("ReadByte at EOF: got no error")
	}
	if err := standard_unread_byte(b); err == nil {
		t.Fatal("UnreadByte after ReadByte at EOF: got no error")
	}

	standard_buffer_write_text(b, "abcdefghijklmnopqrstuvwxyz")

	if n, err := standard_buffer_read(b, nil); n != 0 {
		t.Fatalf("Read(nil) = %d,%v; want 0,nil", n, err)
	} else if err != nil {
		t.Fatalf("Read(nil) = %d,%v; want 0,nil", n, err)
	}
	if err := standard_unread_byte(b); err == nil {
		t.Fatal("UnreadByte after Read(nil): got no error")
	}

	if _, err := standard_buffer_read_bytes(b, 'm'); err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := standard_unread_byte(b); err != nil {
		t.Fatalf("UnreadByte: %v", err)
	}
	c, err := standard_read_byte(b)
	if err != nil {
		t.Fatalf("ReadByte: %v", err)
	}
	if c != 'm' {
		t.Errorf("ReadByte = %q; want %q", c, 'm')
	}
}

// Tests that we occasionally compact. Issue 5154.
func Test_Standard_Library_Buffer_Growth(t *testing.T) {
	var b standard_buffer
	buffer := make([]byte, 1024)
	standard_buffer_write(&b, buffer[0:1])
	var cap0 int
	for i_index := 0; i_index < 5<<10; i_index++ {
		standard_buffer_write(&b, buffer)
		standard_buffer_read(&b, buffer)
		if i_index == 0 {
			cap0 = standard_buffer_capacity(&b)
		}
	}
	cap1_size := standard_buffer_capacity(&b)

	if cap1_size > cap0*3 {
		t.Errorf("buffer cap = %d; too big (grew from %d)", cap1_size, cap0)
	}
}

func Benchmark_Standard_Library_Write_Byte(b *testing.B) {
	const N_SIZE = 4 << 10
	b.SetBytes(N_SIZE)
	buffer := standard_new_buffer(make([]byte, N_SIZE))
	for i_index := 0; i_index < b.N; i_index++ {
		standard_buffer_reset(buffer)
		for write_index := 0; write_index < N_SIZE; write_index++ {
			standard_buffer_write_byte(buffer, 'x')
		}
	}
}

func Benchmark_Standard_Library_Write_Rune(b *testing.B) {
	const RUNE_COUNT = 1 << 10
	const R = '☺'
	b.SetBytes(int64(RUNE_COUNT * utf8.RuneLen(R)))
	buffer := standard_new_buffer(make([]byte, RUNE_COUNT*utf8.UTFMax))
	for i_index := 0; i_index < b.N; i_index++ {
		standard_buffer_reset(buffer)
		for write_index := 0; write_index < RUNE_COUNT; write_index++ {
			standard_buffer_write_character(buffer, R)
		}
	}
}

// From Issue 5154.
func Benchmark_Standard_Library_Buffer_Not_Empty_Write_Read(b *testing.B) {
	buffer := make([]byte, 1024)
	for i_index := 0; i_index < b.N; i_index++ {
		var subject_buffer standard_buffer
		standard_buffer_write(&subject_buffer, buffer[0:1])
		for cycle_index := 0; cycle_index < 5<<10; cycle_index++ {
			standard_buffer_write(&subject_buffer, buffer)
			standard_buffer_read(&subject_buffer, buffer)
		}
	}
}

// Check that we don't compact too often. From Issue 5154.
func Benchmark_Standard_Library_Buffer_Full_Small_Reads(b *testing.B) {
	buffer := make([]byte, 1024)
	for i_index := 0; i_index < b.N; i_index++ {
		var subject_buffer standard_buffer
		standard_buffer_write(&subject_buffer, buffer)
		for standard_unread_size(&subject_buffer)+20 <
			standard_buffer_capacity(&subject_buffer) {
			standard_buffer_write(&subject_buffer, buffer[:10])
		}
		for cycle_index := 0; cycle_index < 5<<10; cycle_index++ {
			standard_buffer_read(&subject_buffer, buffer[:1])
			standard_buffer_write(&subject_buffer, buffer[:1])
		}
	}
}

func Benchmark_Standard_Library_Buffer_Write_Block(b *testing.B) {
	block := make([]byte, 1024)
	for _, n := range []int{1 << 10, 2 << 10, 4 << 10} {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i_index := 0; i_index < b.N; i_index++ {
				var bb standard_buffer
				for standard_unread_size(&bb) < n {
					standard_buffer_write(&bb, block)
				}
			}
		})
	}
}

func Benchmark_Standard_Library_Buffer_Append_No_Copy(b *testing.B) {
	var bb standard_buffer
	standard_buffer_grow(&bb, 4<<10)
	b.SetBytes(int64(standard_buffer_available(&bb)))
	b.ReportAllocs()
	for i_index := 0; i_index < b.N; i_index++ {
		standard_buffer_reset(&bb)
		available := standard_buffer_available_slice(&bb)
		available = available[:cap(available)]
		standard_buffer_write(&bb, available)
	}
}

func slice_of_string(s [][]byte) (result_17 []string) {
	result := make([]string, len(s))
	for i, v := range s {
		result[i] = string(v)
	}
	return result
}

func standard_collect(t *testing.T, sequence iter.Seq[[]byte]) (result_18 [][]byte) {
	output := slices.Collect(sequence)
	out1 := slices.Collect(sequence)
	if !slices.Equal(slice_of_string(output), slice_of_string(out1)) {
		t.Fatalf("inconsistent seq:\n%s\n%s", output, out1)
	}
	return output
}

type Lines_Test struct {
	A string
	B []string
}

func lines_tests() (tests []Lines_Test) {
	return []Lines_Test{
		{A: "abc\nabc\n", B: []string{"abc\n", "abc\n"}},
		{A: "abc\r\nabc", B: []string{"abc\r\n", "abc"}},
		{A: "abc\r\n", B: []string{"abc\r\n"}},
		{A: "\nabc", B: []string{"\n", "abc"}},
		{A: "\nabc\n\n", B: []string{"\n", "abc\n", "\n"}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Lines(t *testing.T) {
	for _, s := range lines_tests() {
		result := slice_of_string(slices.Collect(standard_lines([]byte(s.A))))
		if !slices.Equal(result, s.B) {
			t.Errorf(
				`slices.Collect(standard_lines(%q)) = %q; want %q`,
				s.A,
				result,
				s.B,
			)
		}
	}
}

const ABCD = "abcd"

const FACES = "☺☻☹"

const COMMAS = "1,2,3,4"

const DOTS = "1....2....3....4"

type Binary_Op_Test struct {
	A string
	B string
	I int
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Equal(t *testing.T) {
	for _, tt := range compare_tests() {
		eql := standard_equal(tt.A, tt.B)
		if eql != (tt.I == 0) {
			t.Errorf(`standard_equal(%q, %q) = %v`, tt.A, tt.B, eql)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Equal_Exhaustive(t *testing.T) {
	var size = 128
	if testing.Short() {
		size = 32
	}
	a := make([]byte, size)
	b := make([]byte, size)
	b_init := make([]byte, size)

	for i_index := 0; i_index < size; i_index++ {
		a[i_index] = byte(17 * i_index)
		b_init[i_index] = byte(23*i_index + 100)
	}

	for slice_size := 0; slice_size <= size; slice_size++ {
		for x := 0; x <= size-slice_size; x++ {
			for y := 0; y <= size-slice_size; y++ {
				copy(b, b_init)
				copy(b[y:y+slice_size], a[x:x+slice_size])
				if !standard_equal(
					a[x:x+slice_size],
					b[y:y+slice_size],
				) {
					t.Errorf(
						"standard_equal(%d, %d, %d) = false",
						slice_size,
						x,
						y,
					)
				} else if !standard_equal(
					b[y:y+slice_size],
					a[x:x+slice_size],
				) {
					t.Errorf(
						"standard_equal(%d, %d, %d) = false",
						slice_size,
						x,
						y,
					)
				}
			}
		}
	}
}

// A one-byte difference guards the shortest unequal path.
func Test_Standard_Library_Not_Equal(t *testing.T) {
	t.Parallel()
	var size = 128
	if testing.Short() {
		size = 32
	}
	a := make([]byte, size)
	b := make([]byte, size)

	for slice_size := 0; slice_size <= size; slice_size++ {
		for x := 0; x <= size-slice_size; x++ {
			for y := 0; y <= size-slice_size; y++ {
				for diffpos := x; diffpos < x+slice_size; diffpos++ {
					a[diffpos] = 1
					if standard_equal(a[x:x+slice_size], b[y:y+slice_size]) {
						t.Errorf(
							"NotEqual(%d, %d, %d, %d) = true",
							slice_size,
							x,
							y,
							diffpos,
						)
					} else if standard_equal(
						b[y:y+slice_size],
						a[x:x+slice_size],
					) {
						t.Errorf(
							"NotEqual(%d, %d, %d, %d) = true",
							slice_size,
							x,
							y,
							diffpos,
						)
					}
					a[diffpos] = 0
				}
			}
		}
	}
}

func index_tests() (tests []Binary_Op_Test) {
	return []Binary_Op_Test{
		{"", "", 0},
		{"", "a", -1},
		{"", "foo", -1},
		{"fo", "foo", -1},
		{"foo", "baz", -1},
		{"foo", "foo", 0},
		{"oofofoofooo", "f", 2},
		{"oofofoofooo", "foo", 4},
		{"barfoobarfoo", "foo", 3},
		{"foo", "", 0},
		{"foo", "o", 1},
		{"abcABCabc", "A", 3},
		{"", "a", -1},
		{"x", "a", -1},
		{"x", "x", 0},
		{"abc", "a", 0},
		{"abc", "b", 1},
		{"abc", "c", 2},
		{"abc", "x", -1},
		{"barfoobarfooyyyzzzyyyzzzyyyzzzyyyxxxzzzyyy", "x", 33},
		{"fofofofooofoboo", "oo", 7},
		{"fofofofofofoboo", "ob", 11},
		{"fofofofofofoboo", "boo", 12},
		{"fofofofofofoboo", "oboo", 11},
		{"fofofofofoooboo", "fooo", 8},
		{"fofofofofofoboo", "foboo", 10},
		{"fofofofofofoboo", "fofob", 8},
		{"fofofofofofofoffofoobarfoo", "foffof", 12},
		{"fofofofofoofofoffofoobarfoo", "foffof", 13},
		{"fofofofofofofoffofoobarfoo", "foffofo", 12},
		{"fofofofofoofofoffofoobarfoo", "foffofo", 13},
		{"fofofofofoofofoffofoobarfoo", "foffofoo", 13},
		{"fofofofofofofoffofoobarfoo", "foffofoo", 12},
		{"fofofofofoofofoffofoobarfoo", "foffofoob", 13},
		{"fofofofofofofoffofoobarfoo", "foffofoob", 12},
		{"fofofofofoofofoffofoobarfoo", "foffofooba", 13},
		{"fofofofofofofoffofoobarfoo", "foffofooba", 12},
		{"fofofofofoofofoffofoobarfoo", "foffofoobar", 13},
		{"fofofofofofofoffofoobarfoo", "foffofoobar", 12},
		{"fofofofofoofofoffofoobarfoo", "foffofoobarf", 13},
		{"fofofofofofofoffofoobarfoo", "foffofoobarf", 12},
		{"fofofofofoofofoffofoobarfoo", "foffofoobarfo", 13},
		{"fofofofofofofoffofoobarfoo", "foffofoobarfo", 12},
		{"fofofofofoofofoffofoobarfoo", "foffofoobarfoo", 13},
		{"fofofofofofofoffofoobarfoo", "foffofoobarfoo", 12},
		{"fofofofofoofofoffofoobarfoo", "ofoffofoobarfoo", 12},
		{"fofofofofofofoffofoobarfoo", "ofoffofoobarfoo", 11},
		{"fofofofofoofofoffofoobarfoo", "fofoffofoobarfoo", 11},
		{"fofofofofofofoffofoobarfoo", "fofoffofoobarfoo", 10},
		{"fofofofofoofofoffofoobarfoo", "foobars", -1},
		{"foofyfoobarfoobar", "y", 4},
		{"oooooooooooooooooooooo", "r", -1},
		{"oxoxoxoxoxoxoxoxoxoxoxoy", "oy", 22},
		{"oxoxoxoxoxoxoxoxoxoxoxox", "oy", -1},
		{
			standard_strings.Repeat("0", 71) + "1",
			standard_strings.Repeat("0", 66) + "1",
			5,
		},
		{"oxoxoxoxoxoxoxoxoxoxox☺", "☺", 22},
		{
			"xx012345678901234567890123456789012345678901234567890123456789012" +
				"0123456789012345678901234567890123456xxx\xed\x9f\xc0",
			"\xed\x9f\xc0",
			105,
		},
	}
}

func last_index_tests() (tests []Binary_Op_Test) {
	return []Binary_Op_Test{
		{"", "", 0},
		{"", "a", -1},
		{"", "foo", -1},
		{"fo", "foo", -1},
		{"foo", "foo", 0},
		{"foo", "f", 0},
		{"oofofoofooo", "f", 7},
		{"oofofoofooo", "foo", 7},
		{"barfoobarfoo", "foo", 9},
		{"foo", "", 3},
		{"foo", "o", 2},
		{"abcABCabc", "A", 3},
		{"abcABCabc", "a", 6},
	}
}

func index_any_tests() (tests []Binary_Op_Test) {
	return []Binary_Op_Test{
		{"", "", -1},
		{"", "a", -1},
		{"", "abc", -1},
		{"a", "", -1},
		{"a", "a", 0},
		{"\x80", "\xffb", 0},
		{"aaa", "a", 0},
		{"abc", "xyz", -1},
		{"abc", "xcz", 2},
		{"ab☺c", "x☺yz", 2},
		{"a☺b☻c☹d", "cx", len("a☺b☻")},
		{"a☺b☻c☹d", "uvw☻xyz", len("a☺b")},
		{"aRegExp*", ".(|)*+?^$[]", 7},
		{DOTS + DOTS + DOTS, " ", -1},
		{"012abcba210", "\xffb", 4},
		{"012\x80bcb\x80210", "\xffb", 3},
		{"0123456\xcf\x80abc", "\xcfb\x80", 10},
	}
}

func last_index_any_tests() (tests []Binary_Op_Test) {
	return []Binary_Op_Test{
		{"", "", -1},
		{"", "a", -1},
		{"", "abc", -1},
		{"a", "", -1},
		{"a", "a", 0},
		{"\x80", "\xffb", 0},
		{"aaa", "a", 2},
		{"abc", "xyz", -1},
		{"abc", "ab", 1},
		{"ab☺c", "x☺yz", 2},
		{"a☺b☻c☹d", "cx", len("a☺b☻")},
		{"a☺b☻c☹d", "uvw☻xyz", len("a☺b")},
		{"a.RegExp*", ".(|)*+?^$[]", 8},
		{DOTS + DOTS + DOTS, " ", -1},
		{"012abcba210", "\xffb", 6},
		{"012\x80bcb\x80210", "\xffb", 7},
		{"0123456\xcf\x80abc", "\xcfb\x80", 10},
	}
}

// Execute f on each test case.  funcName should be the name of f; it's used
// in failure reports.
func run_index_tests(
	t *testing.T,
	f func(s, sep []byte) (result_19 int),
	function_name string,
	test_cases []Binary_Op_Test,
) {
	for _, test := range test_cases {
		a := []byte(test.A)
		b := []byte(test.B)
		actual := f(a, b)
		if actual != test.I {
			t.Errorf("%s(%q,%q) = %v; want %v", function_name, a, b, actual, test.I)
		}
	}
	var alloc_tests = []struct {
		A []byte
		B []byte
		I int
	}{

		{
			[]byte(standard_strings.Repeat("0", 71) + "1"),
			[]byte(standard_strings.Repeat("0", 66) + "1"),
			5,
		},

		{
			[]byte(standard_strings.Repeat("0", 64) + "10000"),
			[]byte(standard_strings.Repeat("0", 61) + "1"),
			3,
		},
	}
	if i := standard_index(alloc_tests[1].A, alloc_tests[1].B); i != alloc_tests[1].I {
		t.Errorf(
			"standard_index([]byte(%q), []byte(%q)) = %v; want %v",
			alloc_tests[1].A,
			alloc_tests[1].B,
			i,
			alloc_tests[1].I,
		)
	}
	if i := standard_last_index(alloc_tests[0].A, alloc_tests[0].B); i != alloc_tests[0].I {
		t.Errorf(
			"standard_last_index([]byte(%q), []byte(%q)) = %v; want %v",
			alloc_tests[0].A,
			alloc_tests[0].B,
			i,
			alloc_tests[0].I,
		)
	}
}

func run_index_any_tests(
	t *testing.T,
	f func(s []byte, chars string) (result_20 int),
	function_name string,
	test_cases []Binary_Op_Test,
) {
	for _, test := range test_cases {
		a := []byte(test.A)
		actual := f(a, test.B)
		if actual != test.I {
			t.Errorf(
				"%s(%q,%q) = %v; want %v",
				function_name,
				a,
				test.B,
				actual,
				test.I,
			)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Index(t *testing.T) {
	run_index_tests(t, standard_index, "standard_index", index_tests())
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Last_Index(t *testing.T) {
	run_index_tests(
		t,
		standard_last_index,
		"standard_last_index",
		last_index_tests(),
	)
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Index_Any(t *testing.T) {
	run_index_any_tests(
		t,
		standard_index_any,
		"standard_index_any",
		index_any_tests(),
	)
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Last_Index_Any(t *testing.T) {
	run_index_any_tests(
		t,
		standard_last_index_any,
		"standard_last_index_any",
		last_index_any_tests(),
	)
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Index_Byte(t *testing.T) {
	for _, tt := range index_tests() {
		if len(tt.B) != 1 {
			continue
		}
		a := []byte(tt.A)
		b := tt.B[0]
		position := standard_index_byte(a, b)
		if position != tt.I {
			t.Errorf(`standard_index_byte(%q, '%c') = %v`, tt.A, b, position)
		}
		posp := standard_index_byte(a, b)
		if posp != tt.I {
			t.Errorf(`indexBytePortable(%q, '%c') = %v`, tt.A, b, posp)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Last_Index_Byte(t *testing.T) {
	test_cases := []Binary_Op_Test{
		{"", "q", -1},
		{"abcdef", "q", -1},
		{"abcdefabcdef", "a", len("abcdef")},
		{"abcdefabcdef", "f", len("abcdefabcde")},
		{"zabcdefabcdef", "z", 0},
		{"a☺b☻c☹d", "b", len("a☺")},
	}
	for _, test := range test_cases {
		actual := standard_last_index_byte([]byte(test.A), test.B[0])
		if actual != test.I {
			t.Errorf(
				"standard_last_index_byte(%q,%c) = %v; want %v",
				test.A,
				test.B[0],
				actual,
				test.I,
			)
		}
	}
}

// Different alignments expose reads outside a large requested window.
func Test_Standard_Library_Index_Byte_Big(t *testing.T) {
	t.Parallel()
	var n_size = 1024
	if testing.Short() {
		n_size = 128
	}
	b := make([]byte, n_size)
	for i_index := 0; i_index < n_size; i_index++ {

		b1 := b[i_index:]
		for j_index := 0; j_index < len(b1); j_index++ {
			b1[j_index] = 'x'
			position := standard_index_byte(b1, 'x')
			if position != j_index {
				t.Errorf("standard_index_byte(%q, 'x') = %v", b1, position)
			}
			b1[j_index] = 0
			position = standard_index_byte(b1, 'x')
			if position != -1 {
				t.Errorf("standard_index_byte(%q, 'x') = %v", b1, position)
			}
		}

		b1 = b[:i_index]
		for j_index := 0; j_index < len(b1); j_index++ {
			b1[j_index] = 'x'
			position := standard_index_byte(b1, 'x')
			if position != j_index {
				t.Errorf("standard_index_byte(%q, 'x') = %v", b1, position)
			}
			b1[j_index] = 0
			position = standard_index_byte(b1, 'x')
			if position != -1 {
				t.Errorf("standard_index_byte(%q, 'x') = %v", b1, position)
			}
		}

		b1 = b[i_index/2 : n_size-(i_index+1)/2]
		for j_index := 0; j_index < len(b1); j_index++ {
			b1[j_index] = 'x'
			position := standard_index_byte(b1, 'x')
			if position != j_index {
				t.Errorf("standard_index_byte(%q, 'x') = %v", b1, position)
			}
			b1[j_index] = 0
			position = standard_index_byte(b1, 'x')
			if position != -1 {
				t.Errorf("standard_index_byte(%q, 'x') = %v", b1, position)
			}
		}
	}
}

// Every page offset exposes an alignment-specific small-index regression.
func Test_Standard_Library_Index_Byte_Small(t *testing.T) {
	b := make([]byte, 5015)

	for i := 0; i <= len(b)-15; i++ {
		for j_index := 0; j_index < 15; j_index++ {
			b[i+j_index] = byte(100 + j_index)
		}
		for j_index := 0; j_index < 15; j_index++ {
			p := standard_index_byte(b[i:i+15], byte(100+j_index))
			if p != j_index {
				t.Errorf(
					"standard_index_byte(%q, %d) = %d",
					b[i:i+15],
					100+j_index,
					p,
				)
			}
		}
		for j_index := 0; j_index < 15; j_index++ {
			b[i+j_index] = 0
		}
	}

	for i := 0; i <= len(b)-15; i++ {
		for j_index := 0; j_index < 15; j_index++ {
			b[i+j_index] = 1
		}
		for j_index := 0; j_index < 15; j_index++ {
			p := standard_index_byte(b[i:i+15], byte(0))
			if p != -1 {
				t.Errorf("standard_index_byte(%q, %d) = %d", b[i:i+15], 0, p)
			}
		}
		for j_index := 0; j_index < 15; j_index++ {
			b[i+j_index] = 0
		}
	}
}

type Index_Rune_Test struct {
	In   string
	Rune rune
	Want int
}

func index_rune_tests() (tests []Index_Rune_Test) {
	return []Index_Rune_Test{
		{"", 'a', -1},
		{"", '☺', -1},
		{"foo", '☹', -1},
		{"foo", 'o', 1},
		{"foo☺bar", '☺', 3},
		{"foo☺☻☹bar", '☹', 9},
		{"a A x", 'A', 2},
		{"some_text=some_value", '=', 9},
		{"☺a", 'a', 3},
		{"a☻☺b", '☺', 4},
		{"𠀳𠀗𠀾𠁄𠀧𠁆𠁂𠀫𠀖𠀪𠀲𠀴𠁀𠀨𠀿", '𠀿', 56},

		{"ӆ", 'ӆ', 0},
		{"a", 'ӆ', -1},
		{"  ӆ", 'ӆ', 2},
		{"  a", 'ӆ', -1},
		{standard_strings.Repeat("ц", 64) + "ӆ", 'ӆ', 128},
		{standard_strings.Repeat("ц", 64), 'ӆ', -1},

		{"Ꚁ", 'Ꚁ', 0},
		{"a", 'Ꚁ', -1},
		{"  Ꚁ", 'Ꚁ', 2},
		{"  a", 'Ꚁ', -1},
		{standard_strings.Repeat("Ꙁ", 64) + "Ꚁ", 'Ꚁ', 192},
		{standard_strings.Repeat("Ꙁ", 64) + "Ꚁ", '䚀', -1},

		{"𡌀", '𡌀', 0},
		{"a", '𡌀', -1},
		{"  𡌀", '𡌀', 2},
		{"  a", '𡌀', -1},
		{standard_strings.Repeat("𡋀", 64) + "𡌀", '𡌀', 256},
		{standard_strings.Repeat("𡋀", 64) + "𡌀", '𣌀', -1},

		{"�", '�', 0},
		{"\xff", '�', 0},
		{"☻x�", '�', len("☻x")},
		{"☻x\xe2\x98", '�', len("☻x")},
		{"☻x\xe2\x98�", '�', len("☻x")},
		{"☻x\xe2\x98x", '�', len("☻x")},

		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", -1, -1},
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", 0xD800, -1},
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", utf8.MaxRune + 1, -1},

		{"aaaaaKKKK\U000bc104", '\U000bc104', 17},
		{"aaaaaKKKK鄄", '鄄', 17},
		{"aaKKKKKa\U000bc104", '\U000bc104', 18},
		{"aaKKKKKa鄄", '鄄', 18},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Index_Rune(t *testing.T) {
	for _, tt := range index_rune_tests() {
		if got := standard_index_rune([]byte(tt.In), tt.Rune); got != tt.Want {
			t.Errorf(
				"standard_index_rune(%q, %d) = %v; want %v",
				tt.In,
				tt.Rune,
				got,
				tt.Want,
			)
		}
	}

	haystack := []byte("test世界")
	if i := standard_index_rune(haystack, 's'); i != 2 {
		t.Fatalf("'s' at %d; want 2", i)
	}
	if i := standard_index_rune(haystack, '世'); i != 4 {
		t.Fatalf("'世' at %d; want 4", i)
	}
}

// Every page offset exposes an alignment-specific Count regression.
func Test_Standard_Library_Count_Byte(t *testing.T) {
	b := make([]byte, 5015)
	windows := []int{1, 2, 3, 4, 15, 16, 17, 31, 32, 33, 63, 64, 65, 128}
	test_count_window := func(i, window int) {
		for j_index := 0; j_index < window; j_index++ {
			b[i+j_index] = byte(100)
			p := standard_count(b[i:i+window], []byte{100})
			if p != j_index+1 {
				t.Errorf(
					"TestCountByte.standard_count(%q, 100) = %d",
					b[i:i+window],
					p,
				)
			}
		}
	}

	wnd_max := windows[len(windows)-1]

	for i := 0; i <= 2*wnd_max; i++ {
		for _, window := range windows {
			if window > len(b[i:]) {
				window = len(b[i:])
			}
			test_count_window(i, window)
			for j_index := 0; j_index < window; j_index++ {
				b[i+j_index] = byte(0)
			}
		}
	}
	for i := 4096 - (wnd_max + 1); i < len(b); i++ {
		for _, window := range windows {
			if window > len(b[i:]) {
				window = len(b[i:])
			}
			test_count_window(i, window)
			for j_index := 0; j_index < window; j_index++ {
				b[i+j_index] = byte(0)
			}
		}
	}
}

// Guard bytes expose Count reads outside the requested window.
func Test_Standard_Library_Count_Byte_No_Match(t *testing.T) {
	b := make([]byte, 5015)
	windows := []int{1, 2, 3, 4, 15, 16, 17, 31, 32, 33, 63, 64, 65, 128}
	for i := 0; i <= len(b); i++ {
		for _, window := range windows {
			if window > len(b[i:]) {
				window = len(b[i:])
			}

			for j_index := 0; j_index < window; j_index++ {
				b[i+j_index] = byte(100)
			}

			p := standard_count(b[i:i+window], []byte{0})
			if p != 0 {
				t.Errorf("TestCountByteNoMatch(%q, 0) = %d", b[i:i+window], p)
			}
			for j_index := 0; j_index < window; j_index++ {
				b[i+j_index] = byte(0)
			}
		}
	}
}

func value_name(x int) (result_21 string) {
	if s := x >> 20; s<<20 == x {
		return fmt.Sprintf("%dM", s)
	}
	if s := x >> 10; s<<10 == x {
		return fmt.Sprintf("%dK", s)
	}
	return fmt.Sprint(x)
}

func bench_bytes(b *testing.B, sizes []int, f func(b *testing.B, n int)) {
	for _, n_size := range sizes {
		if IS_RACE_BUILDER {
			if n_size > 4<<10 {
				continue
			}
		}
		b.Run(value_name(n_size), func(b *testing.B) {
			b.SetBytes(int64(n_size))
			f(b, n_size)
		})
	}
}

func index_sizes() (sizes []int) {
	return []int{10, 32, 1 << 10, 2 << 10, 4 << 10}
}

const IS_RACE_BUILDER = false

func Benchmark_Standard_Library_Index_Byte(b *testing.B) {
	bench_bytes(b, index_sizes(), bm_index_byte(standard_index_byte))
}

func Benchmark_Standard_Library_Index_Byte_Portable(b *testing.B) {
	bench_bytes(b, index_sizes(), bm_index_byte(standard_index_byte))
}

func bm_index_byte(index func([]byte, byte) (result_23 int)) (result_22 func(b *testing.B, n int)) {
	return func(b *testing.B, n int) {
		buffer := make([]byte, n)
		buffer[n-1] = 'x'
		for i_index := 0; i_index < b.N; i_index++ {
			j := index(buffer, 'x')
			if j != n-1 {
				b.Fatal("bad index", j)
			}
		}
		buffer[n-1] = '\x00'
	}
}

func Benchmark_Standard_Library_Index_Rune(b *testing.B) {
	bench_bytes(b, index_sizes(), bm_index_rune(standard_index_rune))
}

func Benchmark_Standard_Library_Index_Rune_ASCII(b *testing.B) {
	bench_bytes(b, index_sizes(), bm_index_rune_ascii(standard_index_rune))
}

func Benchmark_Standard_Library_Index_Rune_Unicode(b *testing.B) {
	b.Run("Latin", func(b *testing.B) {

		bench_bytes(b, index_sizes(), bm_index_rune_unicode(unicode.Latin, 'é'))
	})
	b.Run("Cyrillic", func(b *testing.B) {

		bench_bytes(b, index_sizes(), bm_index_rune_unicode(unicode.Cyrillic, 'Ꙁ'))
	})
	b.Run("Han", func(b *testing.B) {

		bench_bytes(b, index_sizes(), bm_index_rune_unicode(unicode.Han, '𠀿'))
	})
}

func bm_index_rune_ascii(
	index func([]byte, rune) (result_25 int),
) (result_24 func(b *testing.B, n int)) {
	return func(b *testing.B, n int) {
		buffer := make([]byte, n)
		buffer[n-1] = 'x'
		for i_index := 0; i_index < b.N; i_index++ {
			j := index(buffer, 'x')
			if j != n-1 {
				b.Fatal("bad index", j)
			}
		}
		buffer[n-1] = '\x00'
	}
}

func bm_index_rune(
	index func([]byte, rune) (result_27 int),
) (result_26 func(b *testing.B, n int)) {
	return func(b *testing.B, n int) {
		buffer := make([]byte, n)
		utf8.EncodeRune(buffer[n-3:], '世')
		for i_index := 0; i_index < b.N; i_index++ {
			j := index(buffer, '世')
			if j != n-3 {
				b.Fatal("bad index", j)
			}
		}
		buffer[n-3] = '\x00'
		buffer[n-2] = '\x00'
		buffer[n-1] = '\x00'
	}
}

func bm_index_rune_unicode(
	rt *unicode.RangeTable,
	needle rune,
) (result_28 func(b *testing.B, n int)) {
	var rs []rune
	for _, r16 := range rt.R16 {
		for r := rune(r16.Lo); r <= rune(r16.Hi); r += rune(r16.Stride) {
			if r != needle {
				rs = append(rs, r)
			}
		}
	}
	for _, r32 := range rt.R32 {
		for r := rune(r32.Lo); r <= rune(r32.Hi); r += rune(r32.Stride) {
			if r != needle {
				rs = append(rs, r)
			}
		}
	}

	rr := rand.New(rand.NewSource(1))
	rr.Shuffle(len(rs), func(i, j int) {
		rs[i], rs[j] = rs[j], rs[i]
	})
	uchars := string(rs)

	return func(b *testing.B, n int) {
		buffer := make([]byte, n)
		o := copy(buffer, uchars)
		for o < len(buffer) {
			o += copy(buffer[o:], uchars)
		}

		m := utf8.RuneLen(needle)
		for o := m; o > 0; {
			_, sz := utf8.DecodeLastRune(buffer)
			copy(buffer[len(buffer)-sz:], "\x00\x00\x00\x00")
			buffer = buffer[:len(buffer)-sz]
			o -= sz
		}
		buffer = utf8.AppendRune(buffer[:n-m], needle)

		n -= m
		for i_index := 0; i_index < b.N; i_index++ {
			j := standard_index_rune(buffer, needle)
			if j != n {
				b.Fatal("bad index", j)
			}
		}
		for i := range buffer {
			buffer[i] = '\x00'
		}
	}
}

func Benchmark_Standard_Library_Equal(b *testing.B) {
	b.Run("0", func(b *testing.B) {
		const EQUAL_BUFFER_SIZE = 4
		var buffer [EQUAL_BUFFER_SIZE]byte
		buf1 := buffer[0:0]
		buf2 := buffer[1:1]
		for i_index := 0; i_index < b.N; i_index++ {
			eq := standard_equal(buf1, buf2)
			if !eq {
				b.Fatal("bad equal")
			}
		}
	})

	sizes := []int{1, 6, 9, 15, 16, 20, 32, 1 << 10, 2 << 10, 4 << 10}

	b.Run("same", func(b *testing.B) {
		bench_bytes(
			b,
			sizes,
			bm_equal(func(a, b []byte) (result_29 bool) {
				return standard_equal(a, a)
			}),
		)
	})

	bench_bytes(b, sizes, bm_equal(standard_equal))
}

func bm_equal(equal func([]byte, []byte) (result_31 bool)) (result_30 func(b *testing.B, n int)) {
	return func(b *testing.B, n int) {
		buffer := make([]byte, 2*n)
		buf1 := buffer[0:n]
		buf2 := buffer[n : 2*n]
		buf1[n-1] = 'x'
		buf2[n-1] = 'x'
		for i_index := 0; i_index < b.N; i_index++ {
			eq := equal(buf1, buf2)
			if !eq {
				b.Fatal("bad equal")
			}
		}
		buf1[n-1] = '\x00'
		buf2[n-1] = '\x00'
	}
}

func Benchmark_Standard_Library_Equal_Both_Unaligned(b *testing.B) {
	sizes := []int{64, 4 << 10}
	benchmark_buffer_size := 2 * (sizes[len(sizes)-1] + 8)
	buffer := make([]byte, benchmark_buffer_size)

	for _, n := range sizes {
		for _, off := range []int{0, 1, 4, 7} {
			buf1 := buffer[off : off+n]
			buf2_start := (len(buffer) / 2) + off
			buf2 := buffer[buf2_start : buf2_start+n]
			buf1[n-1] = 'x'
			buf2[n-1] = 'x'
			b.Run(fmt.Sprint(n, off), func(b *testing.B) {
				b.SetBytes(int64(n))
				for i_index := 0; i_index < b.N; i_index++ {
					eq := standard_equal(buf1, buf2)
					if !eq {
						b.Fatal("bad equal")
					}
				}
			})
			buf1[n-1] = '\x00'
			buf2[n-1] = '\x00'
		}
	}
}

func Benchmark_Standard_Library_Index(b *testing.B) {
	bench_bytes(b, index_sizes(), func(b *testing.B, n int) {
		buffer := make([]byte, n)
		buffer[n-1] = 'x'
		for i_index := 0; i_index < b.N; i_index++ {
			j := standard_index(buffer, buffer[n-7:])
			if j != n-7 {
				b.Fatal("bad index", j)
			}
		}
		buffer[n-1] = '\x00'
	})
}

func Benchmark_Standard_Library_Index_Easy(b *testing.B) {
	bench_bytes(b, index_sizes(), func(b *testing.B, n int) {
		buffer := make([]byte, n)
		buffer[n-1] = 'x'
		buffer[n-7] = 'x'
		for i_index := 0; i_index < b.N; i_index++ {
			j := standard_index(buffer, buffer[n-7:])
			if j != n-7 {
				b.Fatal("bad index", j)
			}
		}
		buffer[n-1] = '\x00'
		buffer[n-7] = '\x00'
	})
}

func Benchmark_Standard_Library_Count(b *testing.B) {
	bench_bytes(b, index_sizes(), func(b *testing.B, n int) {
		buffer := make([]byte, n)
		buffer[n-1] = 'x'
		for i_index := 0; i_index < b.N; i_index++ {
			j := standard_count(buffer, buffer[n-7:])
			if j != 1 {
				b.Fatal("bad count", j)
			}
		}
		buffer[n-1] = '\x00'
	})
}

func Benchmark_Standard_Library_Count_Easy(b *testing.B) {
	bench_bytes(b, index_sizes(), func(b *testing.B, n int) {
		buffer := make([]byte, n)
		buffer[n-1] = 'x'
		buffer[n-7] = 'x'
		for i_index := 0; i_index < b.N; i_index++ {
			j := standard_count(buffer, buffer[n-7:])
			if j != 1 {
				b.Fatal("bad count", j)
			}
		}
		buffer[n-1] = '\x00'
		buffer[n-7] = '\x00'
	})
}

func Benchmark_Standard_Library_Count_Single(b *testing.B) {
	bench_bytes(b, index_sizes(), func(b *testing.B, n int) {
		buffer := make([]byte, n)
		step := 8
		for i := 0; i < len(buffer); i += step {
			buffer[i] = 1
		}
		expect := (len(buffer) + (step - 1)) / step
		for i_index := 0; i_index < b.N; i_index++ {
			j := standard_count(buffer, []byte{1})
			if j != expect {
				b.Fatal("bad count", j, expect)
			}
		}
		clear(buffer)
	})
}

type Split_Test struct {
	S         string
	Separator string
	N         int
	A         []string
}

func splittests() (tests []Split_Test) {
	return []Split_Test{
		{"", "", -1, []string{}},
		{ABCD, "a", 0, nil},
		{ABCD, "", 2, []string{"a", "bcd"}},
		{ABCD, "a", -1, []string{"", "bcd"}},
		{ABCD, "z", -1, []string{"abcd"}},
		{ABCD, "", -1, []string{"a", "b", "c", "d"}},
		{COMMAS, ",", -1, []string{"1", "2", "3", "4"}},
		{DOTS, "...", -1, []string{"1", ".2", ".3", ".4"}},
		{FACES, "☹", -1, []string{"☺☻", ""}},
		{FACES, "~", -1, []string{FACES}},
		{FACES, "", -1, []string{"☺", "☻", "☹"}},
		{"1 2 3 4", " ", 3, []string{"1", "2", "3 4"}},
		{"1 2", " ", 3, []string{"1", "2"}},
		{"123", "", 2, []string{"1", "23"}},
		{"123", "", 17, []string{"1", "2", "3"}},
		{"bT", "T", math.MaxInt / 4, []string{"b", ""}},
		{"\xff-\xff", "", -1, []string{"\xff", "-", "\xff"}},
		{"\xff-\xff", "-", -1, []string{"\xff", "\xff"}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Split(t *testing.T) {
	for _, tt := range splittests() {
		check_split_case(t, tt)
	}
}

func check_split_case(t *testing.T, tt Split_Test) {
	a := standard_split_n([]byte(tt.S), []byte(tt.Separator), tt.N)

	// Appending to the results should not change future results.
	var x []byte
	for _, v := range a {
		x = append(v, 'z')
	}

	result := slice_of_string(a)
	if !slices.Equal(result, tt.A) {
		t.Errorf(
			`standard_split(%q, %q, %d) = %v; want %v`,
			tt.S,
			tt.Separator,
			tt.N,
			result,
			tt.A,
		)
		return
	}

	if tt.N < 0 {
		check_split_sequence(t, tt)
	}

	if tt.N == 0 {
		return
	}
	if len(a) == 0 {
		return
	}

	if want := tt.A[len(tt.A)-1] + "z"; string(x) != want {
		t.Errorf("last appended result was %s; want %s", x, want)
	}

	s := standard_join(a, []byte(tt.Separator))
	if string(s) != tt.S {
		t.Errorf(
			`standard_join(standard_split(%q, %q, %d), %q) = %q`,
			tt.S,
			tt.Separator,
			tt.N,
			tt.Separator,
			s,
		)
	}
	if tt.N < 0 {
		b := slice_of_string(standard_split([]byte(tt.S), []byte(tt.Separator)))
		if !slices.Equal(result, b) {
			t.Errorf(
				"split disagrees with split_n(%q, %q, %d) = %v; want %v",
				tt.S,
				tt.Separator,
				tt.N,
				b,
				a,
			)
		}
	}
	check_split_join_copy(t, tt, a, s)
}

func check_split_sequence(t *testing.T, tt Split_Test) {
	actual := slice_of_string(slices.Collect(standard_split_sequence(
		[]byte(tt.S),
		[]byte(tt.Separator),
	)))
	if !slices.Equal(actual, tt.A) {
		t.Errorf(
			`collect(split_sequence(%q, %q)) = %v; want %v`,
			tt.S,
			tt.Separator,
			actual,
			tt.A,
		)
	}
}

func check_split_join_copy(
	t *testing.T,
	tt Split_Test,
	parts [][]byte,
	joined []byte,
) {
	in := parts[0]
	if cap(in) != cap(joined) {
		return
	}
	if &in[:1][0] == &joined[:1][0] {
		t.Errorf(
			"standard_join(%#v, %q) didn't copy",
			parts,
			tt.Separator,
		)
	}
}

func splitaftertests() (tests []Split_Test) {
	return []Split_Test{
		{ABCD, "a", -1, []string{"a", "bcd"}},
		{ABCD, "z", -1, []string{"abcd"}},
		{ABCD, "", -1, []string{"a", "b", "c", "d"}},
		{COMMAS, ",", -1, []string{"1,", "2,", "3,", "4"}},
		{DOTS, "...", -1, []string{"1...", ".2...", ".3...", ".4"}},
		{FACES, "☹", -1, []string{"☺☻☹", ""}},
		{FACES, "~", -1, []string{FACES}},
		{FACES, "", -1, []string{"☺", "☻", "☹"}},
		{"1 2 3 4", " ", 3, []string{"1 ", "2 ", "3 4"}},
		{"1 2 3", " ", 3, []string{"1 ", "2 ", "3"}},
		{"1 2", " ", 3, []string{"1 ", "2"}},
		{"123", "", 2, []string{"1", "23"}},
		{"123", "", 17, []string{"1", "2", "3"}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Split_After(t *testing.T) {
	for _, tt := range splitaftertests() {
		a := standard_split_after_n([]byte(tt.S), []byte(tt.Separator), tt.N)

		// Appending to the results should not change future results.
		var x []byte
		for _, v := range a {
			x = append(v, 'z')
		}

		result := slice_of_string(a)
		if !slices.Equal(result, tt.A) {
			t.Errorf(
				`standard_split(%q, %q, %d) = %v; want %v`,
				tt.S,
				tt.Separator,
				tt.N,
				result,
				tt.A,
			)
			continue
		}

		if tt.N < 0 {
			check_split_after_sequence(t, tt)
		}

		if want := tt.A[len(tt.A)-1] + "z"; string(x) != want {
			t.Errorf("last appended result was %s; want %s", x, want)
		}

		s := standard_join(a, nil)
		if string(s) != tt.S {
			t.Errorf(
				`standard_join(standard_split(%q, %q, %d), %q) = %q`,
				tt.S,
				tt.Separator,
				tt.N,
				tt.Separator,
				s,
			)
		}
		if tt.N < 0 {
			b := slice_of_string(standard_split_after(
				[]byte(tt.S),
				[]byte(tt.Separator),
			))
			if !slices.Equal(result, b) {
				t.Errorf(
					"split_after != split_after_n(%q, %q, %d): %v; want %v",
					tt.S,
					tt.Separator,
					tt.N,
					b,
					a,
				)
			}
		}
	}
}

func check_split_after_sequence(t *testing.T, tt Split_Test) {
	actual := slice_of_string(slices.Collect(standard_split_after_sequence(
		[]byte(tt.S),
		[]byte(tt.Separator),
	)))
	if !slices.Equal(actual, tt.A) {
		t.Errorf(
			`collect(split_after_sequence(%q, %q)) = %v; want %v`,
			tt.S,
			tt.Separator,
			actual,
			tt.A,
		)
	}
}

type Fields_Test struct {
	S string
	A []string
}

func fieldstests() (tests []Fields_Test) {
	return []Fields_Test{
		{"", []string{}},
		{" ", []string{}},
		{" \t ", []string{}},
		{"  abc  ", []string{"abc"}},
		{"1 2 3 4", []string{"1", "2", "3", "4"}},
		{"1  2  3  4", []string{"1", "2", "3", "4"}},
		{"1\t\t2\t\t3\t4", []string{"1", "2", "3", "4"}},
		{"1\u20002\u20013\u20024", []string{"1", "2", "3", "4"}},
		{"\u2000\u2001\u2002", []string{}},
		{"\n™\t™\n", []string{"™", "™"}},
		{FACES, []string{FACES}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Fields(t *testing.T) {
	for _, tt := range fieldstests() {
		b := []byte(tt.S)
		a := standard_fields(b)

		// Appending to the results should not change future results.
		var x []byte
		for _, v := range a {
			x = append(v, 'z')
		}

		result := slice_of_string(a)
		if !slices.Equal(result, tt.A) {
			t.Errorf("standard_fields(%q) = %v; want %v", tt.S, a, tt.A)
			continue
		}

		result2 := slice_of_string(standard_collect(
			t,
			standard_fields_sequence([]byte(tt.S)),
		))
		if !slices.Equal(result2, tt.A) {
			t.Errorf(
				`standard_collect(standard_fields_sequence(%q)) = %v; want %v`,
				tt.S,
				result2,
				tt.A,
			)
		}

		if string(b) != tt.S {
			t.Errorf("slice changed to %s; want %s", string(b), tt.S)
		}
		if len(tt.A) > 0 {
			if want := tt.A[len(tt.A)-1] + "z"; string(x) != want {
				t.Errorf("last appended result was %s; want %s", x, want)
			}
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Fields_Function(t *testing.T) {
	for _, tt := range fieldstests() {
		a := standard_fields_function([]byte(tt.S), unicode.IsSpace)
		result := slice_of_string(a)
		if !slices.Equal(result, tt.A) {
			t.Errorf(
				"standard_fields_function(%q, unicode.IsSpace) = %v; want %v",
				tt.S,
				a,
				tt.A,
			)
			continue
		}
	}
	match_x := func(c rune) (result_32 bool) { return c == 'X' }
	var fields_function_tests = []Fields_Test{
		{"", []string{}},
		{"XX", []string{}},
		{"XXhiXXX", []string{"hi"}},
		{"aXXbXXXcX", []string{"a", "b", "c"}},
	}
	for _, tt := range fields_function_tests {
		b := []byte(tt.S)
		a := standard_fields_function(b, match_x)

		// Appending to the results should not change future results.
		var x []byte
		for _, v := range a {
			x = append(v, 'z')
		}

		result := slice_of_string(a)
		if !slices.Equal(result, tt.A) {
			t.Errorf("standard_fields_function(%q) = %v, want %v", tt.S, a, tt.A)
		}

		result2 := slice_of_string(standard_collect(
			t,
			standard_fields_function_sequence([]byte(tt.S), match_x),
		))
		if !slices.Equal(result2, tt.A) {
			t.Errorf(
				`collect(fields_function_sequence(%q)) = %v; want %v`,
				tt.S,
				result2,
				tt.A,
			)
		}

		if string(b) != tt.S {
			t.Errorf("slice changed to %s; want %s", b, tt.S)
		}
		if len(tt.A) > 0 {
			if want := tt.A[len(tt.A)-1] + "z"; string(x) != want {
				t.Errorf("last appended result was %s; want %s", x, want)
			}
		}
	}
}

// Test case for any function which accepts and returns a byte slice.
// For ease of creation, we write the input byte slice as a string.
type String_Test struct {
	In     string
	Output []byte
}

func upper_tests() (tests []String_Test) {
	return []String_Test{
		{"", []byte("")},
		{"ONLYUPPER", []byte("ONLYUPPER")},
		{"abc", []byte("ABC")},
		{"AbC123", []byte("ABC123")},
		{"azAZ09_", []byte("AZAZ09_")},
		{"longStrinGwitHmixofsmaLLandcAps", []byte("LONGSTRINGWITHMIXOFSMALLANDCAPS")},
		{
			"long\u0250string\u0250with\u0250nonascii\u2C6Fchars",
			[]byte("LONG\u2C6FSTRING\u2C6FWITH\u2C6FNONASCII\u2C6FCHARS"),
		},
		{"\u0250\u0250\u0250\u0250\u0250", []byte("\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F")},
		{"a\u0080\U0010FFFF", []byte("A\u0080\U0010FFFF")},
	}
}

func lower_tests() (tests []String_Test) {
	return []String_Test{
		{"", []byte("")},
		{"abc", []byte("abc")},
		{"AbC123", []byte("abc123")},
		{"azAZ09_", []byte("azaz09_")},
		{"longStrinGwitHmixofsmaLLandcAps", []byte("longstringwithmixofsmallandcaps")},
		{
			"LONG\u2C6FSTRING\u2C6FWITH\u2C6FNONASCII\u2C6FCHARS",
			[]byte("long\u0250string\u0250with\u0250nonascii\u0250chars"),
		},
		{"\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", []byte("\u0251\u0251\u0251\u0251\u0251")},
		{"A\u0080\U0010FFFF", []byte("a\u0080\U0010FFFF")},
	}
}

const SPACE = "\t\v\r\f\n\u0085\u00a0\u2000\u3000"

func trim_space_tests() (tests []String_Test) {
	return []String_Test{
		{"", nil},
		{"  a", []byte("a")},
		{"b  ", []byte("b")},
		{"abc", []byte("abc")},
		{SPACE + "abc" + SPACE, []byte("abc")},
		{" ", nil},
		{"\u3000 ", nil},
		{" \u3000", nil},
		{" \t\r\n \t\t\r\r\n\n ", nil},
		{" \t\r\n x\t\t\r\r\n\n ", []byte("x")},
		{" \u2000\t\r\n x\t\t\r\r\ny\n \u3000", []byte("x\t\t\r\r\ny")},
		{"1 \t\r\n2", []byte("1 \t\r\n2")},
		{" x\x80", []byte("x\x80")},
		{" x\xc0", []byte("x\xc0")},
		{"x \xc0\xc0 ", []byte("x \xc0\xc0")},
		{"x \xc0", []byte("x \xc0")},
		{"x \xc0 ", []byte("x \xc0")},
		{"x \xc0\xc0 ", []byte("x \xc0\xc0")},
		{"x ☺\xc0\xc0 ", []byte("x ☺\xc0\xc0")},
		{"x ☺ ", []byte("x ☺")},
	}
}

// Execute f on each test case.  funcName should be the name of f; it's used
// in failure reports.
func run_string_tests(
	t *testing.T,
	f func([]byte) (result_33 []byte),
	function_name string,
	test_cases []String_Test,
) {
	for _, tc := range test_cases {
		actual := f([]byte(tc.In))
		if actual == nil {
			if tc.Output != nil {
				t.Errorf("%s(%q) = nil; want %q", function_name, tc.In, tc.Output)
			}
		}
		if actual != nil {
			if tc.Output == nil {
				t.Errorf("%s(%q) = %q; want nil", function_name, tc.In, actual)
			}
		}
		if !standard_equal(actual, tc.Output) {
			t.Errorf("%s(%q) = %q; want %q", function_name, tc.In, actual, tc.Output)
		}
	}
}

func ten_runes(r rune) (result_34 string) {
	runes := make([]rune, 10)
	for i := range runes {
		runes[i] = r
	}
	return string(runes)
}

// A self-inverse mapping lets the case table verify both transformations.
func rot13(r rune) (result_35 rune) {
	const STEP = 13
	if r >= 'a' {
		if r <= 'z' {
			return ((r - 'a' + STEP) % 26) + 'a'
		}
	}
	if r >= 'A' {
		if r <= 'Z' {
			return ((r - 'A' + STEP) % 26) + 'A'
		}
	}
	return r
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Map(t *testing.T) {

	a := ten_runes('a')

	rune_max := func(r rune) (result_36 rune) { return unicode.MaxRune }
	m := standard_map(rune_max, []byte(a))
	expect := ten_runes(unicode.MaxRune)
	if string(m) != expect {
		t.Errorf("growing: expected %q got %q", expect, m)
	}

	rune_min := func(r rune) (result_37 rune) { return 'a' }
	m = standard_map(rune_min, []byte(ten_runes(unicode.MaxRune)))
	expect = a
	if string(m) != expect {
		t.Errorf("shrinking: expected %q got %q", expect, m)
	}

	m = standard_map(rot13, []byte("a to zed"))
	expect = "n gb mrq"
	if string(m) != expect {
		t.Errorf("rot13: expected %q got %q", expect, m)
	}

	m = standard_map(rot13, standard_map(rot13, []byte("a to zed")))
	expect = "a to zed"
	if string(m) != expect {
		t.Errorf("rot13: expected %q got %q", expect, m)
	}

	drop_not_latin := func(r rune) (result_38 rune) {
		if unicode.Is(unicode.Latin, r) {
			return r
		}
		return -1
	}
	m = standard_map(drop_not_latin, []byte("Hello, 세계"))
	expect = "Hello"
	if string(m) != expect {
		t.Errorf("drop: expected %q got %q", expect, m)
	}

	invalid_rune := func(r rune) (result_39 rune) {
		return utf8.MaxRune + 1
	}
	m = standard_map(invalid_rune, []byte("x"))
	expect = "\uFFFD"
	if string(m) != expect {
		t.Errorf("invalidRune: expected %q got %q", expect, m)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_To_Upper(t *testing.T) {
	run_string_tests(t, standard_to_upper, "standard_to_upper", upper_tests())
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_To_Lower(t *testing.T) {
	run_string_tests(t, standard_to_lower, "standard_to_lower", lower_tests())
}

func Benchmark_Standard_Library_To_Upper(b *testing.B) {
	for _, tc := range upper_tests() {
		tin := []byte(tc.In)
		b.Run(tc.In, func(b *testing.B) {
			for i_index := 0; i_index < b.N; i_index++ {
				actual := standard_to_upper(tin)
				if !standard_equal(actual, tc.Output) {
					b.Errorf(
						"standard_to_upper(%q) = %q; want %q",
						tc.In,
						actual,
						tc.Output,
					)
				}
			}
		})
	}
}

func Benchmark_Standard_Library_To_Lower(b *testing.B) {
	for _, tc := range lower_tests() {
		tin := []byte(tc.In)
		b.Run(tc.In, func(b *testing.B) {
			for i_index := 0; i_index < b.N; i_index++ {
				actual := standard_to_lower(tin)
				if !standard_equal(actual, tc.Output) {
					b.Errorf(
						"standard_to_lower(%q) = %q; want %q",
						tc.In,
						actual,
						tc.Output,
					)
				}
			}
		})
	}
}

func to_valid_utf8_tests() (tests []struct {
	In     string
	Repl   string
	Output string
}) {
	return []struct {
		In     string
		Repl   string
		Output string
	}{
		{"", "\uFFFD", ""},
		{"abc", "\uFFFD", "abc"},
		{"\uFDDD", "\uFFFD", "\uFDDD"},
		{"a\xffb", "\uFFFD", "a\uFFFDb"},
		{"a\xffb\uFFFD", "X", "aXb\uFFFD"},
		{"a☺\xffb☺\xC0\xAFc☺\xff", "", "a☺b☺c☺"},
		{"a☺\xffb☺\xC0\xAFc☺\xff", "日本語", "a☺日本語b☺日本語c☺日本語"},
		{"\xC0\xAF", "\uFFFD", "\uFFFD"},
		{"\xE0\x80\xAF", "\uFFFD", "\uFFFD"},
		{"\xed\xa0\x80", "abc", "abc"},
		{"\xed\xbf\xbf", "\uFFFD", "\uFFFD"},
		{"\xF0\x80\x80\xaf", "☺", "☺"},
		{"\xF8\x80\x80\x80\xAF", "\uFFFD", "\uFFFD"},
		{"\xFC\x80\x80\x80\x80\xAF", "\uFFFD", "\uFFFD"},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_To_Valid_UTF8(t *testing.T) {
	for _, tc := range to_valid_utf8_tests() {
		got := standard_to_valid_utf8([]byte(tc.In), []byte(tc.Repl))
		if !standard_equal(got, []byte(tc.Output)) {
			t.Errorf(
				"standard_to_valid_utf8(%q, %q) = %q; want %q",
				tc.In,
				tc.Repl,
				got,
				tc.Output,
			)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Trim_Space(t *testing.T) {
	run_string_tests(t, standard_trim_space, "standard_trim_space", trim_space_tests())
}

type Repeat_Test struct {
	In, Output string
	Count      int
}

func long_string() (value string) {
	return "a" + string(make([]byte, 2046)) + "z"
}

func Repeat_Tests() (tests []Repeat_Test) {
	return []Repeat_Test{
		{"", "", 0},
		{"", "", 1},
		{"", "", 2},
		{"-", "", 0},
		{"-", "-", 1},
		{"-", "----------", 10},
		{"abc ", "abc abc abc ", 3},

		{string(rune(0)), string(make([]byte, 4096)), 4096},
		{long_string(), long_string() + long_string(), 2},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Repeat(t *testing.T) {
	for _, tt := range Repeat_Tests() {
		tin := []byte(tt.In)
		tout := []byte(tt.Output)
		a := standard_repeat(tin, tt.Count)
		if !standard_equal(a, tout) {
			t.Errorf("standard_repeat(%q, %d) = %q; want %q", tin, tt.Count, a, tout)
			continue
		}
	}
}

func repeat(b []byte, count int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("%s", v)
			}
		}
	}()

	standard_repeat(b, count)

	return err
}

// This regression preserves the overflow panic from Go issue 16237.
func Test_Standard_Library_Repeat_Catches_Overflow(t *testing.T) {
	type test_case struct {
		S          string
		Count      int
		Must_Panic bool
	}

	run_test_cases := func(prefix string, tests []test_case) {
		for i, tt := range tests {
			err := repeat([]byte(tt.S), tt.Count)
			if !tt.Must_Panic {
				if err != nil {
					t.Errorf("#%d panicked %v", i, err)
				}
				continue
			}
			if err == nil {
				t.Errorf("%s#%d did not panic", prefix, i)
			}
		}
	}

	const INT_MAX = int(^uint(0) >> 1)

	run_test_cases("", []test_case{
		0: {"--", -2147483647, true},
		1: {"", INT_MAX, true},
		2: {"-", 10, false},
		3: {"gopher", 0, false},
		4: {"-", -1, true},
		5: {"--", -102, true},
		6: {string(make([]byte, 255)), int((^uint(0))/255 + 1), true},
	})

	const IS64_BIT = 1<<(^uintptr(0)>>63)/2 != 0
	if !IS64_BIT {
		return
	}

	run_test_cases("64-bit", []test_case{
		0: {"-", INT_MAX, true},
	})
}

type Runes_Test struct {
	In     string
	Output []rune
	Lossy  bool
}

func Runes_Tests() (tests []Runes_Test) {
	return []Runes_Test{
		{"", []rune{}, false},
		{" ", []rune{32}, false},
		{"ABC", []rune{65, 66, 67}, false},
		{"abc", []rune{97, 98, 99}, false},
		{"\u65e5\u672c\u8a9e", []rune{26085, 26412, 35486}, false},
		{"ab\x80c", []rune{97, 98, 0xFFFD, 99}, true},
		{"ab\xc0c", []rune{97, 98, 0xFFFD, 99}, true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Runes(t *testing.T) {
	for _, tt := range Runes_Tests() {
		tin := []byte(tt.In)
		a := standard_runes(tin)
		if !slices.Equal(a, tt.Output) {
			t.Errorf("standard_runes(%q) = %v; want %v", tin, a, tt.Output)
			continue
		}
		if !tt.Lossy {

			s := string(a)
			if s != tt.In {
				t.Errorf("string(standard_runes(%q)) = %x; want %x", tin, s, tin)
			}
		}
	}
}

type Trim_Test struct {
	F                    string
	In, Argument, Output string
}

func trim_tests() (tests []Trim_Test) {
	return []Trim_Test{
		{"standard_trim", "abba", "a", "bb"},
		{"standard_trim", "abba", "ab", ""},
		{"standard_trim_left", "abba", "ab", ""},
		{"standard_trim_right", "abba", "ab", ""},
		{"standard_trim_left", "abba", "a", "bba"},
		{"standard_trim_left", "abba", "b", "abba"},
		{"standard_trim_right", "abba", "a", "abb"},
		{"standard_trim_right", "abba", "b", "abba"},
		{"standard_trim", "<tag>", "<>", "tag"},
		{"standard_trim", "* listitem", " *", "listitem"},
		{"standard_trim", `"quote"`, `"`, "quote"},
		{"standard_trim", "\u2C6F\u2C6F\u0250\u0250\u2C6F\u2C6F", "\u2C6F", "\u0250\u0250"},
		{"standard_trim", "\x80test\xff", "\xff", "test"},
		{"standard_trim", " Ġ ", " ", "Ġ"},
		{"standard_trim", " Ġİ0", "0 ", "Ġİ"},

		{"standard_trim", "abba", "", "abba"},
		{"standard_trim", "", "123", ""},
		{"standard_trim", "", "", ""},
		{"standard_trim_left", "abba", "", "abba"},
		{"standard_trim_left", "", "123", ""},
		{"standard_trim_left", "", "", ""},
		{"standard_trim_right", "abba", "", "abba"},
		{"standard_trim_right", "", "123", ""},
		{"standard_trim_right", "", "", ""},
		{"standard_trim_right", "☺\xc0", "☺", "☺\xc0"},
		{"standard_trim_prefix", "aabb", "a", "abb"},
		{"standard_trim_prefix", "aabb", "b", "aabb"},
		{"standard_trim_suffix", "aabb", "a", "aabb"},
		{"standard_trim_suffix", "aabb", "b", "aab"},
	}
}

type Trim_Nil_Test struct {
	F        string
	In       []byte
	Argument string
	Output   []byte
}

func trim_nil_tests() (tests []Trim_Nil_Test) {
	return []Trim_Nil_Test{
		{"standard_trim", nil, "", nil},
		{"standard_trim", []byte{}, "", nil},
		{"standard_trim", []byte{'a'}, "a", nil},
		{"standard_trim", []byte{'a', 'a'}, "a", nil},
		{"standard_trim", []byte{'a'}, "ab", nil},
		{"standard_trim", []byte{'a', 'b'}, "ab", nil},
		{"standard_trim", []byte("☺"), "☺", nil},
		{"standard_trim_left", nil, "", nil},
		{"standard_trim_left", []byte{}, "", nil},
		{"standard_trim_left", []byte{'a'}, "a", nil},
		{"standard_trim_left", []byte{'a', 'a'}, "a", nil},
		{"standard_trim_left", []byte{'a'}, "ab", nil},
		{"standard_trim_left", []byte{'a', 'b'}, "ab", nil},
		{"standard_trim_left", []byte("☺"), "☺", nil},
		{"standard_trim_right", nil, "", nil},
		{"standard_trim_right", []byte{}, "", []byte{}},
		{"standard_trim_right", []byte{'a'}, "a", []byte{}},
		{"standard_trim_right", []byte{'a', 'a'}, "a", []byte{}},
		{"standard_trim_right", []byte{'a'}, "ab", []byte{}},
		{"standard_trim_right", []byte{'a', 'b'}, "ab", []byte{}},
		{"standard_trim_right", []byte("☺"), "☺", []byte{}},
		{"standard_trim_prefix", nil, "", nil},
		{"standard_trim_prefix", []byte{}, "", []byte{}},
		{"standard_trim_prefix", []byte{'a'}, "a", []byte{}},
		{"standard_trim_prefix", []byte("☺"), "☺", []byte{}},
		{"standard_trim_suffix", nil, "", nil},
		{"standard_trim_suffix", []byte{}, "", []byte{}},
		{"standard_trim_suffix", []byte{'a'}, "a", []byte{}},
		{"standard_trim_suffix", []byte("☺"), "☺", []byte{}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Trim(t *testing.T) {
	for _, tc := range trim_tests() {
		check_trim_case(t, tc)
	}

	for _, tc := range trim_nil_tests() {
		check_trim_nil_case(t, tc)
	}
}

type trim_set_function func([]byte, string) (result []byte)

type trim_affix_function func([]byte, []byte) (result []byte)

func trim_function(
	t *testing.T,
	name string,
) (set trim_set_function, affix trim_affix_function) {
	switch name {
	case "standard_trim":
		return standard_trim, nil
	case "standard_trim_left":
		return standard_trim_left, nil
	case "standard_trim_right":
		return standard_trim_right, nil
	case "standard_trim_prefix":
		return nil, standard_trim_prefix
	case "standard_trim_suffix":
		return nil, standard_trim_suffix
	default:
		t.Fatalf("undefined trim function %s", name)
		return nil, nil
	}
}

func check_trim_case(t *testing.T, tc Trim_Test) {
	set, affix := trim_function(t, tc.F)
	var actual string
	if set != nil {
		actual = string(set([]byte(tc.In), tc.Argument))
	} else {
		actual = string(affix([]byte(tc.In), []byte(tc.Argument)))
	}
	if actual != tc.Output {
		t.Errorf(
			"%s(%q, %q) = %q; want %q",
			tc.F,
			tc.In,
			tc.Argument,
			actual,
			tc.Output,
		)
	}
}

func check_trim_nil_case(t *testing.T, tc Trim_Nil_Test) {
	set, affix := trim_function(t, tc.F)
	var actual []byte
	if set != nil {
		actual = set(tc.In, tc.Argument)
	} else {
		actual = affix(tc.In, []byte(tc.Argument))
	}
	if len(actual) != 0 {
		t.Errorf(
			"%s(%s, %q) returned non-empty value",
			tc.F,
			report_slice(tc.In),
			tc.Argument,
		)
		return
	}
	actual_nil := actual == nil
	output_nil := tc.Output == nil
	if actual_nil != output_nil {
		t.Errorf(
			"%s(%s, %q) got nil %t; want nil %t",
			tc.F,
			report_slice(tc.In),
			tc.Argument,
			actual_nil,
			output_nil,
		)
	}
}

func report_slice(s []byte) (result string) {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("%q", s)
}

type predicate struct {
	F    func(r rune) (result_45 bool)
	Name string
}

func is_space() (value predicate) {
	return predicate{F: unicode.IsSpace, Name: "IsSpace"}
}

func is_digit() (value predicate) {
	return predicate{F: unicode.IsDigit, Name: "IsDigit"}
}

func is_upper() (value predicate) {
	return predicate{F: unicode.IsUpper, Name: "IsUpper"}
}

func is_valid_rune() (value predicate) {
	return predicate{
		F: func(r rune) (result_46 bool) {
			return r != utf8.RuneError
		},
		Name: "IsValidRune",
	}
}

type Trim_Function_Test struct {
	F            predicate
	In           string
	Trim_Output  []byte
	Left_Output  []byte
	Right_Output []byte
}

func not(p predicate) (result_47 predicate) {
	return predicate{
		F: func(r rune) (result_48 bool) {
			return !p.F(r)
		},
		Name: "not " + p.Name,
	}
}

func trim_function_tests() (tests []Trim_Function_Test) {
	return []Trim_Function_Test{
		{is_space(), SPACE + " hello " + SPACE,
			[]byte("hello"),
			[]byte("hello " + SPACE),
			[]byte(SPACE + " hello")},
		{is_digit(), "\u0e50\u0e5212hello34\u0e50\u0e51",
			[]byte("hello"),
			[]byte("hello34\u0e50\u0e51"),
			[]byte("\u0e50\u0e5212hello")},
		{is_upper(), "\u2C6F\u2C6F\u2C6F\u2C6FABCDhelloEF\u2C6F\u2C6FGH\u2C6F\u2C6F",
			[]byte("hello"),
			[]byte("helloEF\u2C6F\u2C6FGH\u2C6F\u2C6F"),
			[]byte("\u2C6F\u2C6F\u2C6F\u2C6FABCDhello")},
		{not(is_space()), "hello" + SPACE + "hello",
			[]byte(SPACE),
			[]byte(SPACE + "hello"),
			[]byte("hello" + SPACE)},
		{not(is_digit()), "hello\u0e50\u0e521234\u0e50\u0e51helo",
			[]byte("\u0e50\u0e521234\u0e50\u0e51"),
			[]byte("\u0e50\u0e521234\u0e50\u0e51helo"),
			[]byte("hello\u0e50\u0e521234\u0e50\u0e51")},
		{is_valid_rune(), "ab\xc0a\xc0cd",
			[]byte("\xc0a\xc0"),
			[]byte("\xc0a\xc0cd"),
			[]byte("ab\xc0a\xc0")},
		{not(is_valid_rune()), "\xc0a\xc0",
			[]byte("a"),
			[]byte("a\xc0"),
			[]byte("\xc0a")},

		{is_space(), "",
			nil,
			nil,
			[]byte("")},
		{is_space(), " ",
			nil,
			nil,
			[]byte("")},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Trim_Function(t *testing.T) {
	for _, tc := range trim_function_tests() {
		trimmers := []struct {
			Name   string
			Trim   func(s []byte, f func(r rune) (result_50 bool)) (result_49 []byte)
			Output []byte
		}{
			{"standard_trim_function", standard_trim_function, tc.Trim_Output},
			{
				"standard_trim_left_function",
				standard_trim_left_function,
				tc.Left_Output,
			},
			{
				"standard_trim_right_function",
				standard_trim_right_function,
				tc.Right_Output,
			},
		}
		for _, trimmer := range trimmers {
			actual := trimmer.Trim([]byte(tc.In), tc.F.F)
			if actual == nil {
				if trimmer.Output != nil {
					t.Errorf(
						"%s(%q, %q) = nil; want %q",
						trimmer.Name,
						tc.In,
						tc.F.Name,
						trimmer.Output,
					)
				}
			}
			if actual != nil {
				if trimmer.Output == nil {
					t.Errorf(
						"%s(%q, %q) = %q; want nil",
						trimmer.Name,
						tc.In,
						tc.F.Name,
						actual,
					)
				}
			}
			if !standard_equal(actual, trimmer.Output) {
				t.Errorf(
					"%s(%q, %q) = %q; want %q",
					trimmer.Name,
					tc.In,
					tc.F.Name,
					actual,
					trimmer.Output,
				)
			}
		}
	}
}

type Index_Function_Test struct {
	In          string
	F           predicate
	First, Last int
}

func index_function_tests() (tests []Index_Function_Test) {
	return []Index_Function_Test{
		{"", is_valid_rune(), -1, -1},
		{"abc", is_digit(), -1, -1},
		{"0123", is_digit(), 0, 3},
		{"a1b", is_digit(), 1, 1},
		{SPACE, is_space(), 0, len(SPACE) - 3},
		{"\u0e50\u0e5212hello34\u0e50\u0e51", is_digit(), 0, 18},
		{
			"\u2C6F\u2C6F\u2C6F\u2C6FABCDhelloEF\u2C6F\u2C6FGH\u2C6F\u2C6F",
			is_upper(),
			0,
			34,
		},
		{"12\u0e50\u0e52hello34\u0e50\u0e51", not(is_digit()), 8, 12},

		{"\x801", is_digit(), 1, 1},
		{"\x80abc", is_digit(), -1, -1},
		{"\xc0a\xc0", is_valid_rune(), 1, 1},
		{"\xc0a\xc0", not(is_valid_rune()), 0, 2},
		{"\xc0☺\xc0", not(is_valid_rune()), 0, 4},
		{"\xc0☺\xc0\xc0", not(is_valid_rune()), 0, 5},
		{"ab\xc0a\xc0cd", not(is_valid_rune()), 2, 4},
		{"a\xe0\x80cd", not(is_valid_rune()), 1, 2},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Index_Function(t *testing.T) {
	for _, tc := range index_function_tests() {
		first := standard_index_function([]byte(tc.In), tc.F.F)
		if first != tc.First {
			t.Errorf(
				"standard_index_function(%q, %s) = %d; want %d",
				tc.In,
				tc.F.Name,
				first,
				tc.First,
			)
		}
		last := standard_last_index_function([]byte(tc.In), tc.F.F)
		if last != tc.Last {
			t.Errorf(
				"standard_last_index_function(%q, %s) = %d; want %d",
				tc.In,
				tc.F.Name,
				last,
				tc.Last,
			)
		}
	}
}

type Replace_Test struct {
	In       string
	Old, New string
	N        int
	Output   string
}

func Replace_Tests() (tests []Replace_Test) {
	return []Replace_Test{
		{"hello", "l", "L", 0, "hello"},
		{"hello", "l", "L", -1, "heLLo"},
		{"hello", "x", "X", -1, "hello"},
		{"", "x", "X", -1, ""},
		{"radar", "r", "<r>", -1, "<r>ada<r>"},
		{"", "", "<>", -1, "<>"},
		{"banana", "a", "<>", -1, "b<>n<>n<>"},
		{"banana", "a", "<>", 1, "b<>nana"},
		{"banana", "a", "<>", 1000, "b<>n<>n<>"},
		{"banana", "an", "<>", -1, "b<><>a"},
		{"banana", "ana", "<>", -1, "b<>na"},
		{"banana", "", "<>", -1, "<>b<>a<>n<>a<>n<>a<>"},
		{"banana", "", "<>", 10, "<>b<>a<>n<>a<>n<>a<>"},
		{"banana", "", "<>", 6, "<>b<>a<>n<>a<>n<>a"},
		{"banana", "", "<>", 5, "<>b<>a<>n<>a<>na"},
		{"banana", "", "<>", 1, "<>banana"},
		{"banana", "a", "a", -1, "banana"},
		{"banana", "a", "a", 1, "banana"},
		{"☺☻☹", "", "<>", -1, "<>☺<>☻<>☹<>"},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Replace(t *testing.T) {
	for _, tt := range Replace_Tests() {
		var (
			in  = []byte(tt.In)
			old = []byte(tt.Old)
			new = []byte(tt.New)
		)
		standard_replace(in, old, new, tt.N)
		in = append(in, "<spare>"...)
		in = in[:len(tt.In)]
		output := standard_replace(in, old, new, tt.N)
		if s := string(output); s != tt.Output {
			t.Errorf(
				"standard_replace(%q, %q, %q, %d) = %q, want %q",
				tt.In,
				tt.Old,
				tt.New,
				tt.N,
				s,
				tt.Output,
			)
		}
		if cap(in) == cap(output) {
			if &in[:1][0] == &output[:1][0] {
				t.Errorf(
					"standard_replace(%q, %q, %q, %d) didn't copy",
					tt.In,
					tt.Old,
					tt.New,
					tt.N,
				)
			}
		}
		if tt.N == -1 {
			all_output := standard_replace_all(in, old, new)
			if s := string(all_output); s != tt.Output {
				t.Errorf(
					"standard_replace_all(%q, %q, %q) = %q, want %q",
					tt.In,
					tt.Old,
					tt.New,
					s,
					tt.Output,
				)
			}
		}
	}
}

func Fuzz_Standard_Library_Replace(f *testing.F) {
	for _, tt := range Replace_Tests() {
		f.Add([]byte(tt.In), []byte(tt.Old), []byte(tt.New), tt.N)
	}
	f.Fuzz(func(t *testing.T, in, old, new []byte, n int) {
		different_implementation := func(
			in []byte,
			old []byte,
			new []byte,
			n int,
		) (result_51 []byte) {
			var output standard_buffer
			if n < 0 {
				n = math.MaxInt
			}
			for i := 0; i < len(in); {
				if n == 0 {
					standard_buffer_write(&output, in[i:])
					break
				}
				if standard_has_prefix(in[i:], old) {
					standard_buffer_write(&output, new)
					i += len(old)
					n--
					if len(old) != 0 {
						continue
					}
					if i == len(in) {
						break
					}
				}
				if len(old) == 0 {
					_, rune_size := utf8.DecodeRune(in[i:])
					standard_buffer_write(&output, in[i:i+rune_size])
					i += rune_size
				} else {
					standard_buffer_write_byte(&output, in[i])
					i++
				}
			}
			if len(old) == 0 {
				if n != 0 {
					standard_buffer_write(&output, new)
				}
			}
			return standard_buffer_bytes(&output)
		}
		simple := different_implementation(in, old, new, n)
		replaced := standard_replace(in, old, new, n)
		if !slices.Equal(simple, replaced) {
			t.Errorf(
				"different implementations: %q != %q for replace(%q, %q, %q, %d)",
				simple,
				replaced,
				in,
				old,
				new,
				n,
			)
		}
	})
}

func Benchmark_Standard_Library_Replace(b *testing.B) {
	for _, tt := range Replace_Tests() {
		description := fmt.Sprintf("%q %q %q %d", tt.In, tt.Old, tt.New, tt.N)
		var (
			in  = []byte(tt.In)
			old = []byte(tt.Old)
			new = []byte(tt.New)
		)
		b.Run(description, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				standard_replace(in, old, new, tt.N)
			}
		})
	}
}

type Title_Test struct {
	In, Output string
}

func Title_Tests() (tests []Title_Test) {
	return []Title_Test{
		{"", ""},
		{"a", "A"},
		{" aaa aaa aaa ", " Aaa Aaa Aaa "},
		{" Aaa Aaa Aaa ", " Aaa Aaa Aaa "},
		{"123a456", "123a456"},
		{"double-blind", "Double-Blind"},
		{"ÿøû", "Ÿøû"},
		{"with_underscore", "With_underscore"},
		{"unicode \xe2\x80\xa8 line separator", "Unicode \xe2\x80\xa8 Line Separator"},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Title(t *testing.T) {
	for _, tt := range Title_Tests() {
		if s := string(standard_title([]byte(tt.In))); s != tt.Output {
			t.Errorf("standard_title(%q) = %q, want %q", tt.In, s, tt.Output)
		}
	}
}

func To_Title_Tests() (tests []Title_Test) {
	return []Title_Test{
		{"", ""},
		{"a", "A"},
		{" aaa aaa aaa ", " AAA AAA AAA "},
		{" Aaa Aaa Aaa ", " AAA AAA AAA "},
		{"123a456", "123A456"},
		{"double-blind", "DOUBLE-BLIND"},
		{"ÿøû", "ŸØÛ"},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_To_Title(t *testing.T) {
	for _, tt := range To_Title_Tests() {
		if s := string(standard_to_title([]byte(tt.In))); s != tt.Output {
			t.Errorf("standard_to_title(%q) = %q, want %q", tt.In, s, tt.Output)
		}
	}
}

func Equal_Fold_Tests() (tests []struct {
	S, T   string
	Output bool
}) {
	return []struct {
		S, T   string
		Output bool
	}{
		{"abc", "abc", true},
		{"ABcd", "ABcd", true},
		{"123abc", "123ABC", true},
		{"αβδ", "ΑΒΔ", true},
		{"abc", "xyz", false},
		{"abc", "XYZ", false},
		{"abcdefghijk", "abcdefghijX", false},
		{"abcdefghijk", "abcdefghij\u212A", true},
		{"abcdefghijK", "abcdefghij\u212A", true},
		{"abcdefghijkz", "abcdefghij\u212Ay", false},
		{"abcdefghijKz", "abcdefghij\u212Ay", false},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Equal_Fold(t *testing.T) {
	for _, tt := range Equal_Fold_Tests() {
		if output := standard_equal_fold([]byte(tt.S), []byte(tt.T)); output != tt.Output {
			t.Errorf(
				"standard_equal_fold(%#q, %#q) = %v, want %v",
				tt.S,
				tt.T,
				output,
				tt.Output,
			)
		}
		if output := standard_equal_fold([]byte(tt.T), []byte(tt.S)); output != tt.Output {
			t.Errorf(
				"standard_equal_fold(%#q, %#q) = %v, want %v",
				tt.T,
				tt.S,
				output,
				tt.Output,
			)
		}
	}
}

func cut_tests() (tests []struct {
	S, Separator  string
	Before, After string
	Found         bool
}) {
	return []struct {
		S, Separator  string
		Before, After string
		Found         bool
	}{
		{"abc", "b", "a", "c", true},
		{"abc", "a", "", "bc", true},
		{"abc", "c", "ab", "", true},
		{"abc", "abc", "", "", true},
		{"abc", "", "", "abc", true},
		{"abc", "d", "abc", "", false},
		{"", "d", "", "", false},
		{"", "", "", "", true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Cut(t *testing.T) {
	for _, tt := range cut_tests() {
		before, after, found := standard_cut([]byte(tt.S), []byte(tt.Separator))
		mismatch := string(before) != tt.Before
		if !mismatch {
			mismatch = string(after) != tt.After
		}
		if !mismatch {
			mismatch = found != tt.Found
		}
		if mismatch {
			t.Errorf(
				"standard_cut(%q, %q) = %q, %q, %v, want %q, %q, %v",
				tt.S,
				tt.Separator,
				before,
				after,
				found,
				tt.Before,
				tt.After,
				tt.Found,
			)
		}
	}
}

func cut_prefix_tests() (tests []struct {
	S, Separator string
	After        string
	Found        bool
}) {
	return []struct {
		S, Separator string
		After        string
		Found        bool
	}{
		{"abc", "a", "bc", true},
		{"abc", "abc", "", true},
		{"abc", "", "abc", true},
		{"abc", "d", "abc", false},
		{"", "d", "", false},
		{"", "", "", true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Cut_Prefix(t *testing.T) {
	for _, tt := range cut_prefix_tests() {
		after, found := standard_cut_prefix([]byte(tt.S), []byte(tt.Separator))
		mismatch := string(after) != tt.After
		if !mismatch {
			mismatch = found != tt.Found
		}
		if mismatch {
			t.Errorf(
				"standard_cut_prefix(%q, %q) = %q, %v, want %q, %v",
				tt.S,
				tt.Separator,
				after,
				found,
				tt.After,
				tt.Found,
			)
		}
	}
}

func cut_suffix_tests() (tests []struct {
	S, Separator string
	Before       string
	Found        bool
}) {
	return []struct {
		S, Separator string
		Before       string
		Found        bool
	}{
		{"abc", "bc", "a", true},
		{"abc", "abc", "", true},
		{"abc", "", "abc", true},
		{"abc", "d", "abc", false},
		{"", "d", "", false},
		{"", "", "", true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Cut_Suffix(t *testing.T) {
	for _, tt := range cut_suffix_tests() {
		before, found := standard_cut_suffix([]byte(tt.S), []byte(tt.Separator))
		mismatch := string(before) != tt.Before
		if !mismatch {
			mismatch = found != tt.Found
		}
		if mismatch {
			t.Errorf(
				"standard_cut_suffix(%q, %q) = %q, %v, want %q, %v",
				tt.S,
				tt.Separator,
				before,
				found,
				tt.Before,
				tt.Found,
			)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Buffer_Grow_Negative(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("Grow(-1) should have panicked")
		}
	}()
	var b standard_buffer
	standard_buffer_grow(&b, -1)
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Buffer_Truncate_Negative(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("Truncate(-1) should have panicked")
		}
	}()
	var b standard_buffer
	standard_buffer_truncate(&b, -1)
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Buffer_Truncate_Output_Of_Range(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("Truncate(20) should have panicked")
		}
	}()
	var b standard_buffer
	standard_buffer_write(&b, make([]byte, 10))
	standard_buffer_truncate(&b, 20)
}

func contains_tests() (tests []struct {
	B, Subslice []byte
	Want        bool
}) {
	return []struct {
		B, Subslice []byte
		Want        bool
	}{
		{[]byte("hello"), []byte("hel"), true},
		{[]byte("日本語"), []byte("日本"), true},
		{[]byte("hello"), []byte("Hello, world"), false},
		{[]byte("東京"), []byte("京東"), false},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Contains(t *testing.T) {
	for _, tt := range contains_tests() {
		if got := standard_contains(tt.B, tt.Subslice); got != tt.Want {
			t.Errorf(
				"standard_contains(%q, %q) = %v, want %v",
				tt.B,
				tt.Subslice,
				got,
				tt.Want,
			)
		}
	}
}

func Contains_Any_Tests() (tests []struct {
	B        []byte
	Substr   string
	Expected bool
}) {
	return []struct {
		B        []byte
		Substr   string
		Expected bool
	}{
		{[]byte(""), "", false},
		{[]byte(""), "a", false},
		{[]byte(""), "abc", false},
		{[]byte("a"), "", false},
		{[]byte("a"), "a", true},
		{[]byte("aaa"), "a", true},
		{[]byte("abc"), "xyz", false},
		{[]byte("abc"), "xcz", true},
		{[]byte("a☺b☻c☹d"), "uvw☻xyz", true},
		{[]byte("aRegExp*"), ".(|)*+?^$[]", true},
		{[]byte(DOTS + DOTS + DOTS), " ", false},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Contains_Any(t *testing.T) {
	for _, ct := range Contains_Any_Tests() {
		if standard_contains_any(ct.B, ct.Substr) != ct.Expected {
			t.Errorf("standard_contains_any(%s, %s) = %v, want %v",
				ct.B, ct.Substr, !ct.Expected, ct.Expected)
		}
	}
}

func Contains_Rune_Tests() (tests []struct {
	B        []byte
	R        rune
	Expected bool
}) {
	return []struct {
		B        []byte
		R        rune
		Expected bool
	}{
		{[]byte(""), 'a', false},
		{[]byte("a"), 'a', true},
		{[]byte("aaa"), 'a', true},
		{[]byte("abc"), 'y', false},
		{[]byte("abc"), 'c', true},
		{[]byte("a☺b☻c☹d"), 'x', false},
		{[]byte("a☺b☻c☹d"), '☻', true},
		{[]byte("aRegExp*"), '*', true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Contains_Rune(t *testing.T) {
	for _, ct := range Contains_Rune_Tests() {
		if standard_contains_rune(ct.B, ct.R) != ct.Expected {
			t.Errorf("standard_contains_rune(%q, %q) = %v, want %v",
				ct.B, ct.R, !ct.Expected, ct.Expected)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Contains_Function(t *testing.T) {
	for _, ct := range Contains_Rune_Tests() {
		if standard_contains_function(ct.B, func(r rune) (result_52 bool) {
			return ct.R == r
		}) != ct.Expected {
			t.Errorf("standard_contains_function(%q, func(%q)) = %v, want %v",
				ct.B, ct.R, !ct.Expected, ct.Expected)
		}
	}
}

func make_fields_input() (result_53 []byte) {
	x := make([]byte, 4<<10)

	r := rand.New(rand.NewSource(99))
	for i := range x {
		switch r.Intn(10) {
		case 0:
			x[i] = ' '
		case 1:
			if i > 0 {
				if x[i-1] == 'x' {
					copy(x[i-1:], "χ")
					break
				}
			}
			x[i] = 'x'
		default:
			x[i] = 'x'
		}
	}
	return x
}

func make_fields_input_ascii() (result_54 []byte) {
	x := make([]byte, 4<<10)

	r := rand.New(rand.NewSource(99))
	for i := range x {
		if r.Intn(10) == 0 {
			x[i] = ' '
		} else {
			x[i] = 'x'
		}
	}
	return x
}

func bytesdata() (data []struct {
	Name string
	Data []byte
}) {
	return []struct {
		Name string
		Data []byte
	}{
		{"ASCII", make_fields_input_ascii()},
		{"Mixed", make_fields_input()},
	}
}

func Benchmark_Standard_Library_Fields(b *testing.B) {
	for _, sd := range bytesdata() {
		b.Run(sd.Name, func(b *testing.B) {
			for j := 1 << 4; j <= 4<<10; j <<= 4 {
				b.Run(fmt.Sprintf("%d", j), func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(j))
					data := sd.Data[:j]
					for i_index := 0; i_index < b.N; i_index++ {
						standard_fields(data)
					}
				})
			}
		})
	}
}

func Benchmark_Standard_Library_Fields_Function(b *testing.B) {
	for _, sd := range bytesdata() {
		b.Run(sd.Name, func(b *testing.B) {
			for j := 1 << 4; j <= 4<<10; j <<= 4 {
				b.Run(fmt.Sprintf("%d", j), func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(j))
					data := sd.Data[:j]
					for i_index := 0; i_index < b.N; i_index++ {
						standard_fields_function(data, unicode.IsSpace)
					}
				})
			}
		})
	}
}

func Benchmark_Standard_Library_Trim_Space(b *testing.B) {
	tests := []struct {
		Name  string
		Input []byte
	}{
		{"NoTrim", []byte("typical")},
		{"ASCII", []byte("  foo bar  ")},
		{"SomeNonASCII", []byte("    \u2000\t\r\n x\t\t\r\r\ny\n \u3000    ")},
		{"JustNonASCII", []byte("\u2000\u2000\u2000☺☺☺☺\u3000\u3000\u3000")},
	}
	for _, test := range tests {
		b.Run(test.Name, func(b *testing.B) {
			for i_index := 0; i_index < b.N; i_index++ {
				standard_trim_space(test.Input)
			}
		})
	}
}

func Benchmark_Standard_Library_To_Valid_UTF8(b *testing.B) {
	tests := []struct {
		Name  string
		Input []byte
	}{
		{"Valid", []byte("typical")},
		{"InvalidASCII", []byte("foo\xffbar")},
		{"InvalidNonASCII", []byte("日本語\xff日本語")},
	}
	replacement := []byte("\uFFFD")
	b.ResetTimer()
	for _, test := range tests {
		b.Run(test.Name, func(b *testing.B) {
			for i_index := 0; i_index < b.N; i_index++ {
				standard_to_valid_utf8(test.Input, replacement)
			}
		})
	}
}

func make_bench_input_hard() (result_55 []byte) {
	tokens := [...]string{
		"<a>", "<p>", "<b>", "<strong>",
		"</a>", "</p>", "</b>", "</strong>",
		"hello", "world",
	}
	x := make([]byte, 0, 4<<10)
	r := rand.New(rand.NewSource(99))
	complete := false
	for !complete {
		i := r.Intn(len(tokens))
		if len(x)+len(tokens[i]) >= 4<<10 {
			complete = true
			continue
		}
		x = append(x, tokens[i]...)
	}
	return x
}

func benchmark_index_hard(b *testing.B, separator []byte) {
	bench_input_hard := make_bench_input_hard()
	n := standard_index(bench_input_hard, separator)
	if n < 0 {
		n = len(bench_input_hard)
	}
	b.SetBytes(int64(n))
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index(bench_input_hard, separator)
	}
}

func benchmark_last_index_hard(b *testing.B, separator []byte) {
	bench_input_hard := make_bench_input_hard()
	for i_index := 0; i_index < b.N; i_index++ {
		standard_last_index(bench_input_hard, separator)
	}
}

func benchmark_count_hard(b *testing.B, separator []byte) {
	bench_input_hard := make_bench_input_hard()
	for i_index := 0; i_index < b.N; i_index++ {
		standard_count(bench_input_hard, separator)
	}
}

func Benchmark_Standard_Library_Index_Hard1(b *testing.B) { benchmark_index_hard(b, []byte("<>")) }

func Benchmark_Standard_Library_Index_Hard2(b *testing.B) {
	benchmark_index_hard(b, []byte("</pre>"))
}

func Benchmark_Standard_Library_Index_Hard3(b *testing.B) {
	benchmark_index_hard(b, []byte("<b>hello world</b>"))
}

func Benchmark_Standard_Library_Index_Hard4(b *testing.B) {
	benchmark_index_hard(b, []byte("<pre><b>hello</b><strong>world</strong></pre>"))
}

func Benchmark_Standard_Library_Last_Index_Hard1(b *testing.B) {
	benchmark_last_index_hard(b, []byte("<>"))
}

func Benchmark_Standard_Library_Last_Index_Hard2(b *testing.B) {
	benchmark_last_index_hard(b, []byte("</pre>"))
}

func Benchmark_Standard_Library_Last_Index_Hard3(b *testing.B) {
	benchmark_last_index_hard(b, []byte("<b>hello world</b>"))
}

func Benchmark_Standard_Library_Count_Hard1(b *testing.B) { benchmark_count_hard(b, []byte("<>")) }

func Benchmark_Standard_Library_Count_Hard2(b *testing.B) {
	benchmark_count_hard(b, []byte("</pre>"))
}

func Benchmark_Standard_Library_Count_Hard3(b *testing.B) {
	benchmark_count_hard(b, []byte("<b>hello world</b>"))
}

func Benchmark_Standard_Library_Split_Empty_Separator(b *testing.B) {
	bench_input_hard := make_bench_input_hard()
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split(bench_input_hard, nil)
	}
}

func Benchmark_Standard_Library_Split_Single_Byte_Separator(b *testing.B) {
	bench_input_hard := make_bench_input_hard()
	separator := []byte("/")
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split(bench_input_hard, separator)
	}
}

func Benchmark_Standard_Library_Split_Multi_Byte_Separator(b *testing.B) {
	bench_input_hard := make_bench_input_hard()
	separator := []byte("hello")
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split(bench_input_hard, separator)
	}
}

func Benchmark_Standard_Library_Split_N_Single_Byte_Separator(b *testing.B) {
	bench_input_hard := make_bench_input_hard()
	separator := []byte("/")
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split_n(bench_input_hard, separator, 10)
	}
}

func Benchmark_Standard_Library_Split_N_Multi_Byte_Separator(b *testing.B) {
	bench_input_hard := make_bench_input_hard()
	separator := []byte("hello")
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split_n(bench_input_hard, separator, 10)
	}
}

func Benchmark_Standard_Library_Repeat(b *testing.B) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_repeat([]byte("-"), 80)
	}
}

func Benchmark_Standard_Library_Repeat_Large(b *testing.B) {
	s := standard_repeat([]byte("@"), 4<<10)
	for j := 8; j <= 12; j++ {
		for _, k := range []int{1, 16, 4 << 10} {
			window := s[:k]
			n := (1 << j) / k
			if n == 0 {
				continue
			}
			b.Run(fmt.Sprintf("%d/%d", 1<<j, k), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_repeat(window, n)
				}
				b.SetBytes(int64(n * len(window)))
			})
		}
	}
}

func Benchmark_Standard_Library_Bytes_Compare(b *testing.B) {
	for n_size := 1; n_size <= 2048; n_size <<= 1 {
		b.Run(fmt.Sprint(n_size), func(b *testing.B) {
			var x = make([]byte, n_size)
			var y = make([]byte, n_size)

			for i_index := 0; i_index < n_size; i_index++ {
				x[i_index] = 'a'
			}

			for i_index := 0; i_index < n_size; i_index++ {
				y[i_index] = 'a'
			}

			b.ResetTimer()
			for i_index := 0; i_index < b.N; i_index++ {
				standard_compare(x, y)
			}
		})
	}
}

func Benchmark_Standard_Library_Index_Any_ASCII(b *testing.B) {
	x := standard_repeat([]byte{'#'}, 2048)
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_index_any(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Index_Any_UTF8(b *testing.B) {
	x := standard_repeat([]byte{'#'}, 2048)
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_index_any(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Last_Index_Any_ASCII(b *testing.B) {
	x := standard_repeat([]byte{'#'}, 2048)
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_last_index_any(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Last_Index_Any_UTF8(b *testing.B) {
	x := standard_repeat([]byte{'#'}, 2048)
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_last_index_any(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Trim_ASCII(b *testing.B) {
	cs := "0123456789abcdef"
	for k := 1; k <= 4096; k <<= 4 {
		for j := 1; j <= 16; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				x := []byte(standard_strings.Repeat(cs[:j], k))[:k]
				for i_index := 0; i_index < b.N; i_index++ {
					standard_trim(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Trim_Byte(b *testing.B) {
	x := []byte("  the quick brown fox   ")
	for i_index := 0; i_index < b.N; i_index++ {
		standard_trim(x, " ")
	}
}

func Benchmark_Standard_Library_Index_Periodic(b *testing.B) {
	key := []byte{1, 1}
	for _, skip := range [...]int{2, 4, 8, 16, 32, 64} {
		b.Run(fmt.Sprintf("IndexPeriodic%d", skip), func(b *testing.B) {
			buffer := make([]byte, 4<<10)
			for i := 0; i < len(buffer); i += skip {
				buffer[i] = 1
			}
			for i_index := 0; i_index < b.N; i_index++ {
				standard_index(buffer, key)
			}
		})
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Clone(t *testing.T) {
	var clone_tests = [][]byte{
		[]byte(nil),
		[]byte{},
		standard_clone([]byte{}),
		[]byte(standard_strings.Repeat("a", 42))[:0],
		[]byte(standard_strings.Repeat("a", 42))[:0:0],
		[]byte("short"),
		[]byte(standard_strings.Repeat("a", 42)),
	}
	for _, input := range clone_tests {
		clone := standard_clone(input)
		if !standard_equal(clone, input) {
			t.Errorf("standard_clone(%q) = %q; want %q", input, clone, input)
		}

		if input == nil {
			if clone != nil {
				t.Errorf(
					"standard_clone(%#v) return value "+
						"should be equal to nil slice.",
					input,
				)
			}
		}

		if input != nil {
			if clone == nil {
				t.Errorf(
					"standard_clone(%#v) return value "+
						"should not be equal to nil slice.",
					input,
				)
			}
		}

		if cap(input) != 0 {
			if unsafe.SliceData(input) == unsafe.SliceData(clone) {
				t.Errorf(
					"standard_clone(%q) must not reference "+
						"the input backing memory.",
					input,
				)
			}
		}
	}
}

func compare_tests() (tests []struct {
	A, B []byte
	I    int
}) {
	return []struct {
		A, B []byte
		I    int
	}{
		{[]byte(""), []byte(""), 0},
		{[]byte("a"), []byte(""), 1},
		{[]byte(""), []byte("a"), -1},
		{[]byte("abc"), []byte("abc"), 0},
		{[]byte("abd"), []byte("abc"), 1},
		{[]byte("abc"), []byte("abd"), -1},
		{[]byte("ab"), []byte("abc"), -1},
		{[]byte("abc"), []byte("ab"), 1},
		{[]byte("x"), []byte("ab"), 1},
		{[]byte("ab"), []byte("x"), -1},
		{[]byte("x"), []byte("a"), 1},
		{[]byte("b"), []byte("x"), -1},

		{[]byte("abcdefgh"), []byte("abcdefgh"), 0},
		{[]byte("abcdefghi"), []byte("abcdefghi"), 0},
		{[]byte("abcdefghi"), []byte("abcdefghj"), -1},
		{[]byte("abcdefghj"), []byte("abcdefghi"), 1},

		{nil, nil, 0},
		{[]byte(""), nil, 0},
		{nil, []byte(""), 0},
		{[]byte("a"), nil, 1},
		{nil, []byte("a"), -1},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Compare(t *testing.T) {
	for _, tt := range compare_tests() {
		number_shifts := 16
		buffer := make([]byte, len(tt.B)+number_shifts)

		for offset := 0; offset <= number_shifts; offset++ {
			shifted_b := buffer[offset : len(tt.B)+offset]
			copy(shifted_b, tt.B)
			cmp := standard_compare(tt.A, shifted_b)
			if cmp != tt.I {
				t.Errorf(
					`standard_compare(%q, %q), offset %d = %v; want %v`,
					tt.A,
					tt.B,
					offset,
					cmp,
					tt.I,
				)
			}
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Compare_Identical_Slice(t *testing.T) {
	var b = []byte("Hello Gophers!")
	if standard_compare(b, b) != 0 {
		t.Error("b != b")
	}
	if standard_compare(b, b[:1]) != 1 {
		t.Error("b > b[:1] failed")
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Compare_Bytes(t *testing.T) {
	lengths := make([]int, 0)
	for i := 0; i <= 128; i++ {
		lengths = append(lengths, i)
	}
	lengths = append(lengths, 256, 512, 1024, 1333, 4095, 4096)

	n := lengths[len(lengths)-1]
	a := make([]byte, n+1)
	b := make([]byte, n+1)
	for _, slice_size := range lengths {

		for i_index := 0; i_index < slice_size; i_index++ {
			a[i_index] = byte(1 + 31*i_index%254)
			b[i_index] = byte(1 + 31*i_index%254)
		}

		for i := slice_size; i <= n; i++ {
			a[i] = 8
			b[i] = 9
		}
		cmp := standard_compare(a[:slice_size], b[:slice_size])
		if cmp != 0 {
			t.Errorf(`CompareIdentical(%d) = %d`, slice_size, cmp)
		}
		if slice_size > 0 {
			cmp = standard_compare(a[:slice_size-1], b[:slice_size])
			if cmp != -1 {
				t.Errorf(`CompareAshorter(%d) = %d`, slice_size, cmp)
			}
			cmp = standard_compare(a[:slice_size], b[:slice_size-1])
			if cmp != 1 {
				t.Errorf(`CompareBshorter(%d) = %d`, slice_size, cmp)
			}
		}
		for k_index := 0; k_index < slice_size; k_index++ {
			b[k_index] = a[k_index] - 1
			cmp = standard_compare(a[:slice_size], b[:slice_size])
			if cmp != 1 {
				t.Errorf(`CompareAbigger(%d,%d) = %d`, slice_size, k_index, cmp)
			}
			b[k_index] = a[k_index] + 1
			cmp = standard_compare(a[:slice_size], b[:slice_size])
			if cmp != -1 {
				t.Errorf(`CompareBbigger(%d,%d) = %d`, slice_size, k_index, cmp)
			}
			b[k_index] = a[k_index]
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Endian_Base_Compare(t *testing.T) {
	// This test compares byte slices that are almost identical, except one
	// difference that for some j, a[j]>b[j] and a[j+1]<b[j+1]. If the implementation
	// compares large chunks with wrong endianness, it gets wrong result.
	// This size covers the widest vector register used by the implementation.
	const COMPARE_BUFFER_SIZE = 512
	a := make([]byte, COMPARE_BUFFER_SIZE)
	b := make([]byte, COMPARE_BUFFER_SIZE)

	for i_index := 0; i_index < COMPARE_BUFFER_SIZE; i_index++ {
		a[i_index] = byte(1 + 31*i_index%254)
		b[i_index] = byte(1 + 31*i_index%254)
	}
	for compare_size := 2; compare_size <= COMPARE_BUFFER_SIZE; compare_size <<= 1 {
		for j_index := 0; j_index < compare_size-1; j_index++ {
			a[j_index] = b[j_index] - 1
			a[j_index+1] = b[j_index+1] + 1
			cmp := standard_compare(a[:compare_size], b[:compare_size])
			if cmp != -1 {
				t.Errorf(`CompareBbigger(%d,%d) = %d`, compare_size, j_index, cmp)
			}
			a[j_index] = b[j_index] + 1
			a[j_index+1] = b[j_index+1] - 1
			cmp = standard_compare(a[:compare_size], b[:compare_size])
			if cmp != 1 {
				t.Errorf(`CompareAbigger(%d,%d) = %d`, compare_size, j_index, cmp)
			}
			a[j_index] = b[j_index]
			a[j_index+1] = b[j_index+1]
		}
	}
}

func Benchmark_Standard_Library_Compare_Bytes_Equal(b *testing.B) {
	b1 := []byte("Hello Gophers!")
	b2 := []byte("Hello Gophers!")
	for i_index := 0; i_index < b.N; i_index++ {
		if standard_compare(b1, b2) != 0 {
			b.Fatal("b1 != b2")
		}
	}
}

func Benchmark_Standard_Library_Compare_Bytes_To_Nil(b *testing.B) {
	b1 := []byte("Hello Gophers!")
	var b2 []byte
	for i_index := 0; i_index < b.N; i_index++ {
		if standard_compare(b1, b2) != 1 {
			b.Fatal("b1 > b2 failed")
		}
	}
}

func Benchmark_Standard_Library_Compare_Bytes_Empty(b *testing.B) {
	b1 := []byte("")
	b2 := b1
	for i_index := 0; i_index < b.N; i_index++ {
		if standard_compare(b1, b2) != 0 {
			b.Fatal("b1 != b2")
		}
	}
}

func Benchmark_Standard_Library_Compare_Bytes_Identical(b *testing.B) {
	b1 := []byte("Hello Gophers!")
	b2 := b1
	for i_index := 0; i_index < b.N; i_index++ {
		if standard_compare(b1, b2) != 0 {
			b.Fatal("b1 != b2")
		}
	}
}

func Benchmark_Standard_Library_Compare_Bytes_Same_Size(b *testing.B) {
	b1 := []byte("Hello Gophers!")
	b2 := []byte("Hello, Gophers")
	for i_index := 0; i_index < b.N; i_index++ {
		if standard_compare(b1, b2) != -1 {
			b.Fatal("b1 < b2 failed")
		}
	}
}

func Benchmark_Standard_Library_Compare_Bytes_Different_Size(b *testing.B) {
	b1 := []byte("Hello Gophers!")
	b2 := []byte("Hello, Gophers!")
	for i_index := 0; i_index < b.N; i_index++ {
		if standard_compare(b1, b2) != -1 {
			b.Fatal("b1 < b2 failed")
		}
	}
}

func benchmark_compare_bytes_big_unaligned(b *testing.B, offset int) {
	b.StopTimer()
	b1 := make([]byte, 0, 4<<10)
	for len(b1) < 4<<10 {
		b1 = append(b1, "Hello Gophers!"...)
	}
	b1 = b1[:4<<10]
	b2 := append([]byte("12345678")[:offset], b1...)
	b.StartTimer()
	for j_index := 0; j_index < b.N; j_index++ {
		if standard_compare(b1, b2[offset:]) != 0 {
			b.Fatal("b1 != b2")
		}
	}
	b.SetBytes(int64(len(b1)))
}

func Benchmark_Standard_Library_Compare_Bytes_Big_Unaligned(b *testing.B) {
	for i_index := 1; i_index < 8; i_index++ {
		b.Run(fmt.Sprintf("offset=%d", i_index), func(b *testing.B) {
			benchmark_compare_bytes_big_unaligned(b, i_index)
		})
	}
}

func benchmark_compare_bytes_big_both_unaligned(b *testing.B, offset int) {
	b.StopTimer()
	pattern := []byte("Hello Gophers!")
	b1 := make([]byte, 0, 4<<10+len(pattern))
	for len(b1) < 4<<10 {
		b1 = append(b1, pattern...)
	}
	b1 = b1[:4<<10]
	b2 := make([]byte, len(b1))
	copy(b2, b1)
	b.StartTimer()
	for j_index := 0; j_index < b.N; j_index++ {
		if standard_compare(b1[offset:], b2[offset:]) != 0 {
			b.Fatal("b1 != b2")
		}
	}
	b.SetBytes(int64(len(b1[offset:])))
}

func Benchmark_Standard_Library_Compare_Bytes_Big_Both_Unaligned(b *testing.B) {
	for i_index := 0; i_index < 8; i_index++ {
		b.Run(fmt.Sprintf("offset=%d", i_index), func(b *testing.B) {
			benchmark_compare_bytes_big_both_unaligned(b, i_index)
		})
	}
}

func Benchmark_Standard_Library_Compare_Bytes_Big(b *testing.B) {
	b.StopTimer()
	b1 := make([]byte, 0, 4<<10)
	for len(b1) < 4<<10 {
		b1 = append(b1, "Hello Gophers!"...)
	}
	b1 = b1[:4<<10]
	b2 := append([]byte{}, b1...)
	b.StartTimer()
	for i_index := 0; i_index < b.N; i_index++ {
		if standard_compare(b1, b2) != 0 {
			b.Fatal("b1 != b2")
		}
	}
	b.SetBytes(int64(len(b1)))
}

func Benchmark_Standard_Library_Compare_Bytes_Big_Identical(b *testing.B) {
	b.StopTimer()
	b1 := make([]byte, 0, 4<<10)
	for len(b1) < 4<<10 {
		b1 = append(b1, "Hello Gophers!"...)
	}
	b1 = b1[:4<<10]
	b2 := b1
	b.StartTimer()
	for i_index := 0; i_index < b.N; i_index++ {
		if standard_compare(b1, b2) != 0 {
			b.Fatal("b1 != b2")
		}
	}
	b.SetBytes(int64(len(b1)))
}

// Test_Standard_Library_Example_Buffer preserves the upstream example.
func Test_Standard_Library_Example_Buffer(t *testing.T) {
	snap.Run(t, func() {
		var b standard_buffer // A Buffer needs no initialization.
		standard_buffer_write(&b, []byte("Hello "))
		standard_buffer_write_text(&b, fmt.Sprintf("%s", "world!"))
		standard_buffer_write_to(
			&b,
			standard_output_stream(snap.Default.Stdout.Write),
		)

	}, snap.Init(`
STDOUT:
Hello world!
`))
}

// Test_Standard_Library_Example_Buffer_Reader preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Reader(t *testing.T) {
	snap.Run(t, func() {

		buffer := standard_new_buffer_text("R29waGVycyBydWxlIQ==")
		decoded, err := base64.StdEncoding.DecodeString(standard_buffer_string(buffer))
		if err != nil {
			panic(err)
		}
		snap.Default.Stdout.Write(decoded)

	}, snap.Init(`
STDOUT:
Gophers rule!
`))
}

// Test_Standard_Library_Example_Buffer_Bytes preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Bytes(t *testing.T) {
	snap.Run(t, func() {
		buffer := standard_buffer{}
		standard_buffer_write(
			&buffer,
			[]byte{'h', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd'},
		)
		snap.Default.Stdout.Write(standard_buffer_bytes(&buffer))

	}, snap.Init(`
STDOUT:
hello world
`))
}

// Test_Standard_Library_Example_Buffer_Available_Buffer preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Available_Buffer(t *testing.T) {
	// The example intentionally leaves a trailing space after the final number.
	snapshot := `
STDOUT:
0 1 2 3` + " \n"
	snap.Run(t, func() {
		var buffer standard_buffer
		for i_index := 0; i_index < 4; i_index++ {
			b := standard_buffer_available_slice(&buffer)
			b = strconv.AppendInt(b, int64(i_index), 10)
			b = append(b, ' ')
			standard_buffer_write(&buffer, b)
		}
		snap.Default.Stdout.Write(standard_buffer_bytes(&buffer))

	}, snap.Init(snapshot))
}

// Test_Standard_Library_Example_Buffer_Cap preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Cap(t *testing.T) {
	snap.Run(t, func() {
		buf1 := standard_new_buffer(make([]byte, 10))
		buf2 := standard_new_buffer(make([]byte, 0, 10))
		fmt.Fprintln(snap.Default.Stdout, standard_buffer_capacity(buf1))
		fmt.Fprintln(snap.Default.Stdout, standard_buffer_capacity(buf2))

	}, snap.Init(`
STDOUT:
10
10

`))
}

// Test_Standard_Library_Example_Buffer_Grow preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Grow(t *testing.T) {
	snap.Run(t, func() {
		var b standard_buffer
		standard_buffer_grow(&b, 64)
		bb := standard_buffer_bytes(&b)
		standard_buffer_write(&b, []byte("64 bytes or fewer"))
		fmt.Fprintf(snap.Default.Stdout, "%q", bb[:standard_unread_size(&b)])

	}, snap.Init(`
STDOUT:
"64 bytes or fewer"
`))
}

// Test_Standard_Library_Example_Buffer_Size preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Size(t *testing.T) {
	snap.Run(t, func() {
		var b standard_buffer
		standard_buffer_grow(&b, 64)
		standard_buffer_write(&b, []byte("abcde"))
		fmt.Fprintf(snap.Default.Stdout, "%d", standard_unread_size(&b))

	}, snap.Init(`
STDOUT:
5
`))
}

// Test_Standard_Library_Example_Buffer_Next preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Next(t *testing.T) {
	snap.Run(t, func() {
		var b standard_buffer
		standard_buffer_grow(&b, 64)
		standard_buffer_write(&b, []byte("abcde"))
		fmt.Fprintf(snap.Default.Stdout, "%s\n", standard_buffer_next(&b, 2))
		fmt.Fprintf(snap.Default.Stdout, "%s\n", standard_buffer_next(&b, 2))
		fmt.Fprintf(snap.Default.Stdout, "%s", standard_buffer_next(&b, 2))

	}, snap.Init(`
STDOUT:
ab
cd
e
`))
}

// Test_Standard_Library_Example_Buffer_Read preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Read(t *testing.T) {
	snap.Run(t, func() {
		var b standard_buffer
		standard_buffer_grow(&b, 64)
		standard_buffer_write(&b, []byte("abcde"))
		rdbuf := make([]byte, 1)
		n, err := standard_buffer_read(&b, rdbuf)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(snap.Default.Stdout, n)
		fmt.Fprintln(snap.Default.Stdout, standard_buffer_string(&b))
		fmt.Fprintln(snap.Default.Stdout, string(rdbuf))

	}, snap.Init(`
STDOUT:
1
bcde
a

`))
}

// Test_Standard_Library_Example_Buffer_Read_Byte preserves the upstream example.
func Test_Standard_Library_Example_Buffer_Read_Byte(t *testing.T) {
	snap.Run(t, func() {
		var b standard_buffer
		standard_buffer_grow(&b, 64)
		standard_buffer_write(&b, []byte("abcde"))
		c, err := standard_read_byte(&b)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(snap.Default.Stdout, c)
		fmt.Fprintln(snap.Default.Stdout, standard_buffer_string(&b))

	}, snap.Init(`
STDOUT:
97
bcde

`))
}

// Test_Standard_Library_Example_Clone preserves the upstream example.
func Test_Standard_Library_Example_Clone(t *testing.T) {
	snap.Run(t, func() {
		b := []byte("abc")
		clone := standard_clone(b)
		fmt.Fprintf(snap.Default.Stdout, "%s\n", clone)
		clone[0] = 'd'
		fmt.Fprintf(snap.Default.Stdout, "%s\n", b)
		fmt.Fprintf(snap.Default.Stdout, "%s\n", clone)

	}, snap.Init(`
STDOUT:
abc
abc
dbc

`))
}

// Test_Standard_Library_Example_Compare preserves the upstream example.
func Test_Standard_Library_Example_Compare(t *testing.T) {
	snap.Run(t, func() {
		// Interpret Compare's result by comparing it to zero.
		var a, b []byte
		if standard_compare(a, b) < 0 {

		}
		if standard_compare(a, b) <= 0 {

		}
		if standard_compare(a, b) > 0 {

		}
		if standard_compare(a, b) >= 0 {

		}

		if standard_equal(a, b) {

		}
		if !standard_equal(a, b) {

		}
	}, snap.Init(`snap.Run: no output`))
}

// Test_Standard_Library_Example_Compare_Search preserves the upstream example.
func Test_Standard_Library_Example_Compare_Search(t *testing.T) {
	snap.Run(t, func() {
		// Binary search to find a matching byte slice.
		var needle []byte
		var haystack [][]byte // Assume sorted
		_, found := slices.BinarySearchFunc(haystack, needle, standard_compare)
		if found {

		}
	}, snap.Init(`snap.Run: no output`))
}

// Test_Standard_Library_Example_Contains preserves the upstream example.
func Test_Standard_Library_Example_Contains(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains([]byte("seafood"), []byte("foo")),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains([]byte("seafood"), []byte("bar")),
		)
		fmt.Fprintln(snap.Default.Stdout, standard_contains([]byte("seafood"), []byte("")))
		fmt.Fprintln(snap.Default.Stdout, standard_contains([]byte(""), []byte("")))

	}, snap.Init(`
STDOUT:
true
false
true
true

`))
}

// Test_Standard_Library_Example_Contains_Any preserves the upstream example.
func Test_Standard_Library_Example_Contains_Any(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains_any([]byte("I like seafood."), "fÄo!"),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains_any([]byte("I like seafood."), "去是伟大的."),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains_any([]byte("I like seafood."), ""),
		)
		fmt.Fprintln(snap.Default.Stdout, standard_contains_any([]byte(""), ""))

	}, snap.Init(`
STDOUT:
true
true
false
false

`))
}

// Test_Standard_Library_Example_Contains_Rune preserves the upstream example.
func Test_Standard_Library_Example_Contains_Rune(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains_rune([]byte("I like seafood."), 'f'),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains_rune([]byte("I like seafood."), 'ö'),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains_rune([]byte("去是伟大的!"), '大'),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_contains_rune([]byte("去是伟大的!"), '!'),
		)
		fmt.Fprintln(snap.Default.Stdout, standard_contains_rune([]byte(""), '@'))

	}, snap.Init(`
STDOUT:
true
false
true
true
false

`))
}

// Test_Standard_Library_Example_Contains_Function preserves the upstream example.
func Test_Standard_Library_Example_Contains_Function(t *testing.T) {
	snap.Run(t, func() {
		f := func(r rune) (result_56 bool) {
			return r >= 'a' && r <= 'z'
		}
		fmt.Fprintln(snap.Default.Stdout, standard_contains_function([]byte("HELLO"), f))
		fmt.Fprintln(snap.Default.Stdout, standard_contains_function([]byte("World"), f))

	}, snap.Init(`
STDOUT:
false
true

`))
}

// Test_Standard_Library_Example_Count preserves the upstream example.
func Test_Standard_Library_Example_Count(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(snap.Default.Stdout, standard_count([]byte("cheese"), []byte("e")))
		fmt.Fprintln(snap.Default.Stdout, standard_count([]byte("five"), []byte("")))

	}, snap.Init(`
STDOUT:
3
5

`))
}

// Test_Standard_Library_Example_Cut preserves the upstream example.
func Test_Standard_Library_Example_Cut(t *testing.T) {
	snap.Run(t, func() {
		show := func(s, sep string) {
			before, after, found := standard_cut([]byte(s), []byte(sep))
			fmt.Fprintf(
				snap.Default.Stdout,
				"Cut(%q, %q) = %q, %q, %v\n",
				s,
				sep,
				before,
				after,
				found,
			)
		}
		show("Gopher", "Go")
		show("Gopher", "ph")
		show("Gopher", "er")
		show("Gopher", "Badger")

	}, snap.Init(`
STDOUT:
Cut("Gopher", "Go") = "", "pher", true
Cut("Gopher", "ph") = "Go", "er", true
Cut("Gopher", "er") = "Goph", "", true
Cut("Gopher", "Badger") = "Gopher", "", false

`))
}

// Test_Standard_Library_Example_Cut_Prefix preserves the upstream example.
func Test_Standard_Library_Example_Cut_Prefix(t *testing.T) {
	snap.Run(t, func() {
		show := func(s, prefix string) {
			after, found := standard_cut_prefix([]byte(s), []byte(prefix))
			fmt.Fprintf(
				snap.Default.Stdout,
				"CutPrefix(%q, %q) = %q, %v\n",
				s,
				prefix,
				after,
				found,
			)
		}
		show("Gopher", "Go")
		show("Gopher", "ph")

	}, snap.Init(`
STDOUT:
CutPrefix("Gopher", "Go") = "pher", true
CutPrefix("Gopher", "ph") = "Gopher", false

`))
}

// Test_Standard_Library_Example_Cut_Suffix preserves the upstream example.
func Test_Standard_Library_Example_Cut_Suffix(t *testing.T) {
	snap.Run(t, func() {
		show := func(s, suffix string) {
			before, found := standard_cut_suffix([]byte(s), []byte(suffix))
			fmt.Fprintf(
				snap.Default.Stdout,
				"CutSuffix(%q, %q) = %q, %v\n",
				s,
				suffix,
				before,
				found,
			)
		}
		show("Gopher", "Go")
		show("Gopher", "er")

	}, snap.Init(`
STDOUT:
CutSuffix("Gopher", "Go") = "Gopher", false
CutSuffix("Gopher", "er") = "Goph", true

`))
}

// Test_Standard_Library_Example_Equal preserves the upstream example.
func Test_Standard_Library_Example_Equal(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(snap.Default.Stdout, standard_equal([]byte("Go"), []byte("Go")))
		fmt.Fprintln(snap.Default.Stdout, standard_equal([]byte("Go"), []byte("C++")))

	}, snap.Init(`
STDOUT:
true
false

`))
}

// Test_Standard_Library_Example_Equal_Fold preserves the upstream example.
func Test_Standard_Library_Example_Equal_Fold(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(snap.Default.Stdout, standard_equal_fold([]byte("Go"), []byte("go")))

	}, snap.Init(`
STDOUT:
true

`))
}

// Test_Standard_Library_Example_Fields preserves the upstream example.
func Test_Standard_Library_Example_Fields(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"Fields are: %q",
			standard_fields([]byte("  foo bar  baz   ")),
		)

	}, snap.Init(`
STDOUT:
Fields are: ["foo" "bar" "baz"]
`))
}

// Test_Standard_Library_Example_Fields_Function preserves the upstream example.
func Test_Standard_Library_Example_Fields_Function(t *testing.T) {
	snap.Run(t, func() {
		f := func(c rune) (result_57 bool) {
			return !unicode.IsLetter(c) && !unicode.IsNumber(c)
		}
		fmt.Fprintf(
			snap.Default.Stdout,
			"Fields are: %q",
			standard_fields_function([]byte("  foo1;bar2,baz3..."), f),
		)

	}, snap.Init(`
STDOUT:
Fields are: ["foo1" "bar2" "baz3"]
`))
}

// Test_Standard_Library_Example_Has_Prefix preserves the upstream example.
func Test_Standard_Library_Example_Has_Prefix(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_has_prefix([]byte("Gopher"), []byte("Go")),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_has_prefix([]byte("Gopher"), []byte("C")),
		)
		fmt.Fprintln(snap.Default.Stdout, standard_has_prefix([]byte("Gopher"), []byte("")))

	}, snap.Init(`
STDOUT:
true
false
true

`))
}

// Test_Standard_Library_Example_Has_Suffix preserves the upstream example.
func Test_Standard_Library_Example_Has_Suffix(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_has_suffix([]byte("Amigo"), []byte("go")),
		)
		fmt.Fprintln(snap.Default.Stdout, standard_has_suffix([]byte("Amigo"), []byte("O")))
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_has_suffix([]byte("Amigo"), []byte("Ami")),
		)
		fmt.Fprintln(snap.Default.Stdout, standard_has_suffix([]byte("Amigo"), []byte("")))

	}, snap.Init(`
STDOUT:
true
false
false
true

`))
}

// Test_Standard_Library_Example_Index preserves the upstream example.
func Test_Standard_Library_Example_Index(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(snap.Default.Stdout, standard_index([]byte("chicken"), []byte("ken")))
		fmt.Fprintln(snap.Default.Stdout, standard_index([]byte("chicken"), []byte("dmr")))

	}, snap.Init(`
STDOUT:
4
-1

`))
}

// Test_Standard_Library_Example_Index_Byte preserves the upstream example.
func Test_Standard_Library_Example_Index_Byte(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(snap.Default.Stdout, standard_index_byte([]byte("chicken"), byte('k')))
		fmt.Fprintln(snap.Default.Stdout, standard_index_byte([]byte("chicken"), byte('g')))

	}, snap.Init(`
STDOUT:
4
-1

`))
}

// Test_Standard_Library_Example_Index_Function preserves the upstream example.
func Test_Standard_Library_Example_Index_Function(t *testing.T) {
	snap.Run(t, func() {
		f := func(c rune) (result_58 bool) {
			return unicode.Is(unicode.Han, c)
		}
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_index_function([]byte("Hello, 世界"), f),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_index_function([]byte("Hello, world"), f),
		)

	}, snap.Init(`
STDOUT:
7
-1

`))
}

// Test_Standard_Library_Example_Index_Any preserves the upstream example.
func Test_Standard_Library_Example_Index_Any(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(snap.Default.Stdout, standard_index_any([]byte("chicken"), "aeiouy"))
		fmt.Fprintln(snap.Default.Stdout, standard_index_any([]byte("crwth"), "aeiouy"))

	}, snap.Init(`
STDOUT:
2
-1

`))
}

// Test_Standard_Library_Example_Index_Rune preserves the upstream example.
func Test_Standard_Library_Example_Index_Rune(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(snap.Default.Stdout, standard_index_rune([]byte("chicken"), 'k'))
		fmt.Fprintln(snap.Default.Stdout, standard_index_rune([]byte("chicken"), 'd'))

	}, snap.Init(`
STDOUT:
4
-1

`))
}

// Test_Standard_Library_Example_Join preserves the upstream example.
func Test_Standard_Library_Example_Join(t *testing.T) {
	snap.Run(t, func() {
		s := [][]byte{[]byte("foo"), []byte("bar"), []byte("baz")}
		fmt.Fprintf(snap.Default.Stdout, "%s", standard_join(s, []byte(", ")))

	}, snap.Init(`
STDOUT:
foo, bar, baz
`))
}

// Test_Standard_Library_Example_Last_Index preserves the upstream example.
func Test_Standard_Library_Example_Last_Index(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(snap.Default.Stdout, standard_index([]byte("go gopher"), []byte("go")))
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index([]byte("go gopher"), []byte("go")),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index([]byte("go gopher"), []byte("rodent")),
		)

	}, snap.Init(`
STDOUT:
0
3
-1

`))
}

// Test_Standard_Library_Example_Last_Index_Any preserves the upstream example.
func Test_Standard_Library_Example_Last_Index_Any(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_any([]byte("go gopher"), "MüQp"),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_any([]byte("go 地鼠"), "地大"),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_any([]byte("go gopher"), "z,!."),
		)

	}, snap.Init(`
STDOUT:
5
3
-1

`))
}

// Test_Standard_Library_Example_Last_Index_Byte preserves the upstream example.
func Test_Standard_Library_Example_Last_Index_Byte(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_byte([]byte("go gopher"), byte('g')),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_byte([]byte("go gopher"), byte('r')),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_byte([]byte("go gopher"), byte('z')),
		)

	}, snap.Init(`
STDOUT:
3
8
-1

`))
}

// Test_Standard_Library_Example_Last_Index_Function preserves the upstream example.
func Test_Standard_Library_Example_Last_Index_Function(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_function([]byte("go gopher!"), unicode.IsLetter),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_function([]byte("go gopher!"), unicode.IsPunct),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_last_index_function([]byte("go gopher!"), unicode.IsNumber),
		)

	}, snap.Init(`
STDOUT:
8
9
-1

`))
}

// Test_Standard_Library_Example_Map preserves the upstream example.
func Test_Standard_Library_Example_Map(t *testing.T) {
	snap.Run(t, func() {
		rot13_mapping := func(r rune) (result_59 rune) {
			switch {
			case r >= 'A' && r <= 'Z':
				return 'A' + (r-'A'+13)%26
			case r >= 'a' && r <= 'z':
				return 'a' + (r-'a'+13)%26
			}
			return r
		}
		fmt.Fprintf(snap.Default.Stdout,
			"%s\n",
			standard_map(
				rot13_mapping,
				[]byte("'Twas brillig and the slithy gopher..."),
			),
		)

	}, snap.Init(`
STDOUT:
'Gjnf oevyyvt naq gur fyvgul tbcure...

`))
}

// Test_Standard_Library_Example_Reader_Size preserves the upstream example.
func Test_Standard_Library_Example_Reader_Size(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_unread_size(standard_new_reader([]byte("Hi!"))),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			standard_unread_size(standard_new_reader([]byte("こんにちは!"))),
		)

	}, snap.Init(`
STDOUT:
3
16

`))
}

// Test_Standard_Library_Example_Repeat preserves the upstream example.
func Test_Standard_Library_Example_Repeat(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(snap.Default.Stdout, "ba%s", standard_repeat([]byte("na"), 2))

	}, snap.Init(`
STDOUT:
banana
`))
}

// Test_Standard_Library_Example_Replace preserves the upstream example.
func Test_Standard_Library_Example_Replace(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"%s\n",
			standard_replace(
				[]byte("oink oink oink"),
				[]byte("k"),
				[]byte("ky"),
				2,
			),
		)
		fmt.Fprintf(snap.Default.Stdout, "%s\n", standard_replace(
			[]byte("oink oink oink"),
			[]byte("oink"),
			[]byte("moo"),
			-1,
		))

	}, snap.Init(`
STDOUT:
oinky oinky oink
moo moo moo

`))
}

// Test_Standard_Library_Example_Replace_All preserves the upstream example.
func Test_Standard_Library_Example_Replace_All(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(snap.Default.Stdout, "%s\n", standard_replace_all(
			[]byte("oink oink oink"),
			[]byte("oink"),
			[]byte("moo"),
		))

	}, snap.Init(`
STDOUT:
moo moo moo

`))
}

// Test_Standard_Library_Example_Runes preserves the upstream example.
func Test_Standard_Library_Example_Runes(t *testing.T) {
	snap.Run(t, func() {
		rs := standard_runes([]byte("go gopher"))
		for _, r := range rs {
			fmt.Fprintf(snap.Default.Stdout, "%#U\n", r)
		}

	}, snap.Init(`
STDOUT:
U+0067 'g'
U+006F 'o'
U+0020 ' '
U+0067 'g'
U+006F 'o'
U+0070 'p'
U+0068 'h'
U+0065 'e'
U+0072 'r'

`))
}

// Test_Standard_Library_Example_Split preserves the upstream example.
func Test_Standard_Library_Example_Split(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"%q\n",
			standard_split([]byte("a,b,c"), []byte(",")),
		)
		fmt.Fprintf(
			snap.Default.Stdout,
			"%q\n",
			standard_split(
				[]byte("a man a plan a canal panama"),
				[]byte("a "),
			),
		)
		fmt.Fprintf(
			snap.Default.Stdout,
			"%q\n",
			standard_split([]byte(" xyz "), []byte("")),
		)
		fmt.Fprintf(
			snap.Default.Stdout,
			"%q\n",
			standard_split([]byte(""), []byte("Bernardo O'Higgins")),
		)

	}, snap.Init(`
STDOUT:
["a" "b" "c"]
["" "man " "plan " "canal panama"]
[" " "x" "y" "z" " "]
[""]

`))
}

// Test_Standard_Library_Example_Split_N preserves the upstream example.
func Test_Standard_Library_Example_Split_N(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"%q\n",
			standard_split_n([]byte("a,b,c"), []byte(","), 2),
		)
		z := standard_split_n([]byte("a,b,c"), []byte(","), 0)
		fmt.Fprintf(snap.Default.Stdout, "%q (nil = %v)\n", z, z == nil)

	}, snap.Init(`
STDOUT:
["a" "b,c"]
[] (nil = true)

`))
}

// Test_Standard_Library_Example_Split_After preserves the upstream example.
func Test_Standard_Library_Example_Split_After(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"%q\n",
			standard_split_after([]byte("a,b,c"), []byte(",")),
		)

	}, snap.Init(`
STDOUT:
["a," "b," "c"]

`))
}

// Test_Standard_Library_Example_Split_After_N preserves the upstream example.
func Test_Standard_Library_Example_Split_After_N(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"%q\n",
			standard_split_after_n([]byte("a,b,c"), []byte(","), 2),
		)

	}, snap.Init(`
STDOUT:
["a," "b,c"]

`))
}

// Test_Standard_Library_Example_Title preserves the upstream example.
func Test_Standard_Library_Example_Title(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(snap.Default.Stdout, "%s", standard_title([]byte("her royal highness")))

	}, snap.Init(`
STDOUT:
Her Royal Highness
`))
}

// Test_Standard_Library_Example_To_Title preserves the upstream example.
func Test_Standard_Library_Example_To_Title(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(snap.Default.Stdout, "%s\n", standard_to_title([]byte("loud noises")))
		fmt.Fprintf(snap.Default.Stdout, "%s\n", standard_to_title([]byte("брат")))

	}, snap.Init(`
STDOUT:
LOUD NOISES
БРАТ

`))
}

// Test_Standard_Library_Example_To_Title_Special preserves the upstream example.
func Test_Standard_Library_Example_To_Title_Special(t *testing.T) {
	snap.Run(t, func() {
		string_value := []byte("ahoj vývojári golang")
		totitle := standard_to_title_special(
			ucd.Special_Case(ucd.Azeri_Case()), string_value,
		)
		fmt.Fprintln(snap.Default.Stdout, "Original : "+string(string_value))
		fmt.Fprintln(snap.Default.Stdout, "ToTitle : "+string(totitle))

	}, snap.Init(`
STDOUT:
Original : ahoj vývojári golang
ToTitle : AHOJ VÝVOJÁRİ GOLANG

`))
}

// Test_Standard_Library_Example_To_Valid_Utf8 preserves the upstream example.
func Test_Standard_Library_Example_To_Valid_Utf8(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"%s\n",
			standard_to_valid_utf8([]byte("abc"), []byte("\uFFFD")),
		)
		fmt.Fprintf(
			snap.Default.Stdout,
			"%s\n",
			standard_to_valid_utf8([]byte("a\xffb\xC0\xAFc\xff"), []byte("")),
		)
		fmt.Fprintf(
			snap.Default.Stdout,
			"%s\n",
			standard_to_valid_utf8([]byte("\xed\xa0\x80"), []byte("abc")),
		)

	}, snap.Init(`
STDOUT:
abc
abc
abc

`))
}

// Test_Standard_Library_Example_Trim preserves the upstream example.
func Test_Standard_Library_Example_Trim(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"[%q]",
			standard_trim([]byte(" !!! Achtung! Achtung! !!! "), "! "),
		)

	}, snap.Init(`
STDOUT:
["Achtung! Achtung"]
`))
}

// Test_Standard_Library_Example_Trim_Function preserves the upstream example.
func Test_Standard_Library_Example_Trim_Function(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			string(standard_trim_function([]byte("go-gopher!"), unicode.IsLetter)),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			string(standard_trim_function(
				[]byte("\"go-gopher!\""),
				unicode.IsLetter,
			)),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			string(standard_trim_function([]byte("go-gopher!"), unicode.IsPunct)),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			string(standard_trim_function(
				[]byte("1234go-gopher!567"),
				unicode.IsNumber,
			)),
		)

	}, snap.Init(`
STDOUT:
-gopher!
"go-gopher!"
go-gopher
go-gopher!

`))
}

// Test_Standard_Library_Example_Trim_Left preserves the upstream example.
func Test_Standard_Library_Example_Trim_Left(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprint(
			snap.Default.Stdout,
			string(standard_trim_left([]byte("453gopher8257"), "0123456789")),
		)

	}, snap.Init(`
STDOUT:
gopher8257
`))
}

// Test_Standard_Library_Example_Trim_Left_Function preserves the upstream example.
func Test_Standard_Library_Example_Trim_Left_Function(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			string(standard_trim_left_function([]byte("go-gopher"), unicode.IsLetter)),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			string(standard_trim_left_function([]byte("go-gopher!"), unicode.IsPunct)),
		)
		fmt.Fprintln(snap.Default.Stdout, string(standard_trim_left_function(
			[]byte("1234go-gopher!567"),
			unicode.IsNumber,
		)))

	}, snap.Init(`
STDOUT:
-gopher
go-gopher!
go-gopher!567

`))
}

// Test_Standard_Library_Example_Trim_Prefix preserves the upstream example.
func Test_Standard_Library_Example_Trim_Prefix(t *testing.T) {
	snap.Run(t, func() {
		var b = []byte("Goodbye,, world!")
		b = standard_trim_prefix(b, []byte("Goodbye,"))
		b = standard_trim_prefix(b, []byte("See ya,"))
		fmt.Fprintf(snap.Default.Stdout, "Hello%s", b)

	}, snap.Init(`
STDOUT:
Hello, world!
`))
}

// Test_Standard_Library_Example_Trim_Space preserves the upstream example.
func Test_Standard_Library_Example_Trim_Space(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(
			snap.Default.Stdout,
			"%s",
			standard_trim_space([]byte(" \t\n a lone gopher \n\t\r\n")),
		)

	}, snap.Init(`
STDOUT:
a lone gopher
`))
}

// Test_Standard_Library_Example_Trim_Suffix preserves the upstream example.
func Test_Standard_Library_Example_Trim_Suffix(t *testing.T) {
	snap.Run(t, func() {
		var b = []byte("Hello, goodbye, etc!")
		b = standard_trim_suffix(b, []byte("goodbye, etc!"))
		b = standard_trim_suffix(b, []byte("gopher"))
		b = append(b, standard_trim_suffix([]byte("world!"), []byte("x!"))...)
		snap.Default.Stdout.Write(b)

	}, snap.Init(`
STDOUT:
Hello, world!
`))
}

// Test_Standard_Library_Example_Trim_Right preserves the upstream example.
func Test_Standard_Library_Example_Trim_Right(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprint(
			snap.Default.Stdout,
			string(standard_trim_right([]byte("453gopher8257"), "0123456789")),
		)

	}, snap.Init(`
STDOUT:
453gopher
`))
}

// Test_Standard_Library_Example_Trim_Right_Function preserves the upstream example.
func Test_Standard_Library_Example_Trim_Right_Function(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintln(
			snap.Default.Stdout,
			string(standard_trim_right_function([]byte("go-gopher"), unicode.IsLetter)),
		)
		fmt.Fprintln(
			snap.Default.Stdout,
			string(standard_trim_right_function([]byte("go-gopher!"), unicode.IsPunct)),
		)
		fmt.Fprintln(snap.Default.Stdout, string(standard_trim_right_function(
			[]byte("1234go-gopher!567"),
			unicode.IsNumber,
		)))

	}, snap.Init(`
STDOUT:
go-
go-gopher
1234go-gopher!

`))
}

// Test_Standard_Library_Example_To_Lower preserves the upstream example.
func Test_Standard_Library_Example_To_Lower(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(snap.Default.Stdout, "%s", standard_to_lower([]byte("Gopher")))

	}, snap.Init(`
STDOUT:
gopher
`))
}

// Test_Standard_Library_Example_To_Lower_Special preserves the upstream example.
func Test_Standard_Library_Example_To_Lower_Special(t *testing.T) {
	snap.Run(t, func() {
		string_value := []byte("AHOJ VÝVOJÁRİ GOLANG")
		totitle := standard_to_lower_special(
			ucd.Special_Case(ucd.Azeri_Case()), string_value,
		)
		fmt.Fprintln(snap.Default.Stdout, "Original : "+string(string_value))
		fmt.Fprintln(snap.Default.Stdout, "ToLower : "+string(totitle))

	}, snap.Init(`
STDOUT:
Original : AHOJ VÝVOJÁRİ GOLANG
ToLower : ahoj vývojári golang

`))
}

// Test_Standard_Library_Example_To_Upper preserves the upstream example.
func Test_Standard_Library_Example_To_Upper(t *testing.T) {
	snap.Run(t, func() {
		fmt.Fprintf(snap.Default.Stdout, "%s", standard_to_upper([]byte("Gopher")))

	}, snap.Init(`
STDOUT:
GOPHER
`))
}

// Test_Standard_Library_Example_To_Upper_Special preserves the upstream example.
func Test_Standard_Library_Example_To_Upper_Special(t *testing.T) {
	snap.Run(t, func() {
		string_value := []byte("ahoj vývojári golang")
		totitle := standard_to_upper_special(
			ucd.Special_Case(ucd.Azeri_Case()), string_value,
		)
		fmt.Fprintln(snap.Default.Stdout, "Original : "+string(string_value))
		fmt.Fprintln(snap.Default.Stdout, "ToUpper : "+string(totitle))

	}, snap.Init(`
STDOUT:
Original : ahoj vývojári golang
ToUpper : AHOJ VÝVOJÁRİ GOLANG

`))
}

// Test_Standard_Library_Example_Lines preserves the upstream example.
func Test_Standard_Library_Example_Lines(t *testing.T) {
	snap.Run(t, func() {
		text := []byte("Hello\nWorld\nGo Programming\n")
		for line := range standard_lines(text) {
			fmt.Fprintf(snap.Default.Stdout, "%q\n", line)
		}

	}, snap.Init(`
STDOUT:
"Hello\n"
"World\n"
"Go Programming\n"

`))
}

// Test_Standard_Library_Example_Split_Sequence preserves the upstream example.
func Test_Standard_Library_Example_Split_Sequence(t *testing.T) {
	snap.Run(t, func() {
		s := []byte("a,b,c,d")
		for part := range standard_split_sequence(s, []byte(",")) {
			fmt.Fprintf(snap.Default.Stdout, "%q\n", part)
		}

	}, snap.Init(`
STDOUT:
"a"
"b"
"c"
"d"

`))
}

// Test_Standard_Library_Example_Split_After_Sequence preserves the upstream example.
func Test_Standard_Library_Example_Split_After_Sequence(t *testing.T) {
	snap.Run(t, func() {
		s := []byte("a,b,c,d")
		for part := range standard_split_after_sequence(s, []byte(",")) {
			fmt.Fprintf(snap.Default.Stdout, "%q\n", part)
		}

	}, snap.Init(`
STDOUT:
"a,"
"b,"
"c,"
"d"

`))
}

// Test_Standard_Library_Example_Fields_Sequence preserves the upstream example.
func Test_Standard_Library_Example_Fields_Sequence(t *testing.T) {
	snap.Run(t, func() {
		text := []byte("The quick brown fox")
		fmt.Fprintln(snap.Default.Stdout, "Split byte slice into fields:")
		for word := range standard_fields_sequence(text) {
			fmt.Fprintf(snap.Default.Stdout, "%q\n", word)
		}

		text_with_spaces := []byte("  lots   of   spaces  ")
		fmt.Fprintln(snap.Default.Stdout, "\nSplit byte slice with multiple spaces:")
		for word := range standard_fields_sequence(text_with_spaces) {
			fmt.Fprintf(snap.Default.Stdout, "%q\n", word)
		}

	}, snap.Init(`
STDOUT:
Split byte slice into fields:
"The"
"quick"
"brown"
"fox"

Split byte slice with multiple spaces:
"lots"
"of"
"spaces"

`))
}

// Test_Standard_Library_Example_Fields_Function_Sequence preserves the upstream example.
func Test_Standard_Library_Example_Fields_Function_Sequence(t *testing.T) {
	snap.Run(t, func() {
		text := []byte("The quick brown fox")
		fmt.Fprintln(snap.Default.Stdout, "Split on whitespace(similar to FieldsSeq):")
		for word := range standard_fields_function_sequence(text, unicode.IsSpace) {
			fmt.Fprintf(snap.Default.Stdout, "%q\n", word)
		}

		mixed_text := []byte("abc123def456ghi")
		fmt.Fprintln(snap.Default.Stdout, "\nSplit on digits:")
		for word := range standard_fields_function_sequence(mixed_text, unicode.IsDigit) {
			fmt.Fprintf(snap.Default.Stdout, "%q\n", word)
		}

	}, snap.Init(`
STDOUT:
Split on whitespace(similar to FieldsSeq):
"The"
"quick"
"brown"
"fox"

Split on digits:
"abc"
"def"
"ghi"

`))
}

func Benchmark_Standard_Library_Split_Sequence_Empty_Separator(Benchmark *testing.B) {
	bench_input_hard := make_bench_input_hard()
	for range Benchmark.N {
		for range standard_split_sequence(bench_input_hard, nil) {
		}
	}
}

func Benchmark_Standard_Library_Split_Sequence_Single_Byte_Separator(Benchmark *testing.B) {
	bench_input_hard := make_bench_input_hard()
	Separator := []byte("/")
	for range Benchmark.N {
		for range standard_split_sequence(bench_input_hard, Separator) {
		}
	}
}

func Benchmark_Standard_Library_Split_Sequence_Multiple_Byte_Separator(Benchmark *testing.B) {
	bench_input_hard := make_bench_input_hard()
	Separator := []byte("hello")
	for range Benchmark.N {
		for range standard_split_sequence(bench_input_hard, Separator) {
		}
	}
}

func Benchmark_Standard_Library_Split_After_Sequence_Empty_Separator(Benchmark *testing.B) {
	bench_input_hard := make_bench_input_hard()
	for range Benchmark.N {
		for range standard_split_after_sequence(bench_input_hard, nil) {
		}
	}
}

func Benchmark_Standard_Library_Split_After_Sequence_Single_Byte_Separator(Benchmark *testing.B) {
	bench_input_hard := make_bench_input_hard()
	Separator := []byte("/")
	for range Benchmark.N {
		for range standard_split_after_sequence(bench_input_hard, Separator) {
		}
	}
}

func Benchmark_Standard_Library_Split_After_Sequence_Multiple_Byte_Separator(Benchmark *testing.B) {
	bench_input_hard := make_bench_input_hard()
	Separator := []byte("hello")
	for range Benchmark.N {
		for range standard_split_after_sequence(bench_input_hard, Separator) {
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Reader(t *testing.T) {
	r := standard_new_reader([]byte("0123456789"))
	tests := []struct {
		Off     int64
		Seek    int
		N       int
		Want    string
		Wantpos int64
		Readerr error
		Seekerr string
	}{
		{Seek: standard_io.SeekStart, Off: 0, N: 20, Want: "0123456789"},
		{Seek: standard_io.SeekStart, Off: 1, N: 1, Want: "1"},
		{Seek: standard_io.SeekCurrent, Off: 1, Wantpos: 3, N: 2, Want: "34"},
		{Seek: standard_io.SeekStart, Off: -1, Seekerr: "io: stream invalid offset"},
		{Seek: standard_io.SeekStart, Off: 1 << 33, Seekerr: "io: stream invalid offset"},
		{Seek: standard_io.SeekCurrent, Off: 1 << 33, Seekerr: "io: stream invalid offset"},
		{Seek: standard_io.SeekStart, N: 5, Want: "01234"},
		{Seek: standard_io.SeekCurrent, N: 5, Want: "56789"},
		{Seek: standard_io.SeekEnd, Off: -1, N: 1, Wantpos: 9, Want: "9"},
	}

	for i, tt := range tests {
		position, err := standard_reader_seek(r, tt.Off, tt.Seek)
		if err == nil {
			if tt.Seekerr != "" {
				t.Errorf("%d. want seek error %q", i, tt.Seekerr)
				continue
			}
		}
		if err != nil {
			if err.Error() != tt.Seekerr {
				t.Errorf("%d. seek error = %q; want %q", i, err.Error(), tt.Seekerr)
				continue
			}
		}
		if tt.Wantpos != 0 {
			if tt.Wantpos != position {
				t.Errorf("%d. pos = %d, want %d", i, position, tt.Wantpos)
			}
		}
		buffer := make([]byte, tt.N)
		n, err := standard_reader_read(r, buffer)
		if err != tt.Readerr {
			t.Errorf("%d. read = %v; want %v", i, err, tt.Readerr)
			continue
		}
		got := string(buffer[:n])
		if got != tt.Want {
			t.Errorf("%d. got %q; want %q", i, got, tt.Want)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Read_After_Big_Seek(t *testing.T) {
	r := standard_new_reader([]byte("0123456789"))
	if _, err := standard_reader_seek(r, 1<<31+5, standard_io.SeekStart); err == nil {
		t.Fatal("A seek after the source must report an error")
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Reader_At(t *testing.T) {
	r := standard_new_reader([]byte("0123456789"))
	tests := []struct {
		Off     int64
		N       int
		Want    string
		Wanterr any
	}{
		{0, 10, "0123456789", nil},
		{1, 10, "123456789", standard_io.EOF},
		{1, 9, "123456789", nil},
		{11, 10, "", standard_io.EOF},
		{0, 0, "", nil},
		{-1, 0, "", standard_io.EOF},
	}
	for i, tt := range tests {
		b := make([]byte, tt.N)
		rn, err := standard_reader_read_at(r, b, tt.Off)
		got := string(b[:rn])
		if got != tt.Want {
			t.Errorf("%d. got %q; want %q", i, got, tt.Want)
		}
		if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.Wanterr) {
			t.Errorf("%d. got error = %v; want %v", i, err, tt.Wanterr)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Reader_At_Repeated(t *testing.T) {
	r := standard_new_reader([]byte("0123456789"))
	const READ_SIZE = 1
	for i_index := 0; i_index < 5; i_index++ {
		var buffer [READ_SIZE]byte
		count, err := standard_reader_read_at(r, buffer[:], int64(i_index))
		if count != READ_SIZE {
			t.Fatalf("ReadAt count = %d; want %d", count, READ_SIZE)
		}
		if err != nil {
			t.Fatalf("ReadAt failed: %s", err)
		}
		if buffer[0] != byte('0'+i_index) {
			t.Fatalf("ReadAt byte = %q; want %q", buffer[0], byte('0'+i_index))
		}
	}
	if standard_unread_size(r) != 10 {
		t.Fatal("ReadAt must not change Reader state")
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Empty_Reader_Repeated(t *testing.T) {
	r := standard_new_reader([]byte{})
	const READ_SIZE = 1
	for i_index := 0; i_index < 5; i_index++ {
		var buffer [READ_SIZE]byte
		count, err := standard_reader_read(r, buffer[:])
		if count != 0 {
			t.Fatalf("empty Read count = %d; want 0", count)
		}
		if err != standard_io.EOF {
			t.Fatalf("empty Read error = %v; want EOF", err)
		}
		standard_reader_read(r, nil)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Reader_Write_To(t *testing.T) {
	for i := 0; i < 30; i += 3 {
		var l int
		if i > 0 {
			l = len(test_string()) / i
		}
		s := test_string()[:l]
		r := standard_new_reader(test_bytes()[:l])
		var b standard_buffer
		n, err := standard_reader_write_to(
			r,
			shared_bytes.Buffer_To_Stream(standard_buffer_port(&b)),
		)
		if expect := int64(len(s)); n != expect {
			t.Errorf("got %v; want %v", n, expect)
		}
		if err != nil {
			t.Errorf("for length %d: got error = %v; want nil", l, err)
		}
		if standard_buffer_string(&b) != s {
			t.Errorf("got string %q; want %q", standard_buffer_string(&b), s)
		}
		if standard_unread_size(r) != 0 {
			t.Errorf("reader contains %v bytes; want 0", standard_unread_size(r))
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Reader_Unread_Size(t *testing.T) {
	const DATA = "hello world"
	r := standard_new_reader([]byte(DATA))
	if got, want := standard_unread_size(r), 11; got != want {
		t.Errorf("r.Len(): got %d, want %d", got, want)
	}
	if n, err := standard_reader_read(r, make([]byte, 10)); err != nil {
		t.Errorf("Read failed: read %d %v", n, err)
	} else if n != 10 {
		t.Errorf("Read failed: read %d %v", n, err)
	}
	if got, want := standard_unread_size(r), 1; got != want {
		t.Errorf("r.Len(): got %d, want %d", got, want)
	}
	if n, err := standard_reader_read(r, make([]byte, 1)); err != nil {
		t.Errorf("Read failed: read %d %v; want 1, nil", n, err)
	} else if n != 1 {
		t.Errorf("Read failed: read %d %v; want 1, nil", n, err)
	}
	if got, want := standard_unread_size(r), 0; got != want {
		t.Errorf("r.Len(): got %d, want %d", got, want)
	}
}

func Unread_Rune_Error_Tests() (tests []struct {
	Name string
	F    func(*standard_reader)
}) {
	return []struct {
		Name string
		F    func(*standard_reader)
	}{
		{"Read", func(r *standard_reader) { standard_reader_read(r, []byte{0}) }},
		{"ReadByte", func(r *standard_reader) { standard_read_byte(r) }},
		{"UnreadRune", func(r *standard_reader) { standard_unread_character(r) }},
		{"Seek", func(r *standard_reader) {
			standard_reader_seek(r, 0, standard_io.SeekCurrent)
		}},
		{"WriteTo", func(r *standard_reader) {
			destination := &standard_buffer{}
			standard_reader_write_to(
				r,
				shared_bytes.Buffer_To_Stream(standard_buffer_port(destination)),
			)
		}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Unread_Rune_Error(t *testing.T) {
	for _, tt := range Unread_Rune_Error_Tests() {
		reader := standard_new_reader([]byte("0123456789"))
		if _, _, err := standard_read_character(reader); err != nil {

			t.Fatal(err)
		}
		tt.F(reader)
		err := standard_unread_character(reader)
		if err == nil {
			t.Errorf("Unreading after %s: expected error", tt.Name)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Reader_Double_Unread_Rune(t *testing.T) {
	buffer := standard_new_buffer([]byte("groucho"))
	if _, _, err := standard_read_character(buffer); err != nil {

		t.Fatal(err)
	}
	if err := standard_unread_byte(buffer); err != nil {

		t.Fatal(err)
	}
	if err := standard_unread_byte(buffer); err == nil {
		t.Fatal("UnreadByte: expected error, got nil")
	}
}

// An empty Reader stream reports the same end as the direct operation.
func Test_Standard_Library_Reader_Copy_Empty(t *testing.T) {
	reader := standard_new_reader(nil)
	stream := shared_bytes.Reader_To_Stream(standard_reader_port(reader))
	stream_count, stream_err := io.Read(stream, make([]byte, 1))
	direct_count, direct_err := standard_reader_read(reader, make([]byte, 1))
	if stream_count != int64(direct_count) {
		t.Errorf("Stream count = %d; direct count = %d", stream_count, direct_count)
	}
	if standard_stream_error(stream_err) != direct_err {
		t.Errorf("Stream error = %v; direct error = %v", stream_err, direct_err)
	}
}

// Reads change Len but must not change the original Size.
func Test_Standard_Library_Reader_Unread_And_Total_Size(t *testing.T) {
	r := standard_new_reader([]byte("abc"))
	standard_reader_read(r, make([]byte, 1))
	if standard_unread_size(r) != 2 {
		t.Errorf("Len = %d; want 2", standard_unread_size(r))
	}
	if standard_reader_size(r) != 3 {
		t.Errorf("Size = %d; want 3", standard_reader_size(r))
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Reader_Reset(t *testing.T) {
	r := standard_new_reader([]byte("世界"))
	if _, _, err := standard_read_character(r); err != nil {
		t.Errorf("ReadRune: unexpected error: %v", err)
	}

	const WANT = "abcdef"
	standard_reader_reset(r, []byte(WANT))
	if err := standard_unread_character(r); err == nil {
		t.Errorf("UnreadRune: expected error, got nil")
	}
	buffer := make([]byte, len(WANT))
	count, err := standard_reader_read(r, buffer)
	if err != nil {
		t.Errorf("ReadAll: unexpected error: %v", err)
	}
	if count != len(buffer) {
		t.Errorf("ReadAll: got %d bytes, want %d", count, len(buffer))
	}
	if got := string(buffer); got != WANT {
		t.Errorf("ReadAll: got %q, want %q", got, WANT)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Standard_Library_Reader_Zero(t *testing.T) {
	if l_size := standard_unread_size((&standard_reader{})); l_size != 0 {
		t.Errorf("Len: got %d, want 0", l_size)
	}

	if n, err := standard_reader_read(&standard_reader{}, nil); n != 0 {
		t.Errorf("Read: got %d, %v; want 0, standard_io.EOF", n, err)
	} else if err != standard_io.EOF {
		t.Errorf("Read: got %d, %v; want 0, standard_io.EOF", n, err)
	}

	if n, err := standard_reader_read_at((&standard_reader{}), nil, 11); n != 0 {
		t.Errorf("ReadAt: got %d, %v; want 0, standard_io.EOF", n, err)
	} else if err != standard_io.EOF {
		t.Errorf("ReadAt: got %d, %v; want 0, standard_io.EOF", n, err)
	}

	if b, err := standard_read_byte((&standard_reader{})); b != 0 {
		t.Errorf("ReadByte: got %d, %v; want 0, standard_io.EOF", b, err)
	} else if err != standard_io.EOF {
		t.Errorf("ReadByte: got %d, %v; want 0, standard_io.EOF", b, err)
	}

	if ch, size, err := standard_read_character((&standard_reader{})); ch != 0 {
		t.Errorf("ReadRune: got %d, %d, %v; want 0, 0, standard_io.EOF", ch, size, err)
	} else if size != 0 {
		t.Errorf("ReadRune: got %d, %d, %v; want 0, 0, standard_io.EOF", ch, size, err)
	} else if err != standard_io.EOF {
		t.Errorf("ReadRune: got %d, %d, %v; want 0, 0, standard_io.EOF", ch, size, err)
	}

	if offset, err := standard_reader_seek(
		&standard_reader{}, 11, standard_io.SeekStart,
	); offset != 0 {
		t.Errorf("Seek: got %d, %v; want 0, error", offset, err)
	} else if err == nil {
		t.Errorf("Seek: got %d, nil; want 0, error", offset)
	}

	if s := standard_reader_size(&standard_reader{}); s != 0 {
		t.Errorf("Size: got %d, want 0", s)
	}

	if standard_unread_byte((&standard_reader{})) == nil {
		t.Errorf("UnreadByte: got nil, want error")
	}

	if standard_unread_character((&standard_reader{})) == nil {
		t.Errorf("UnreadRune: got nil, want error")
	}

	discard := &io.Stream_Discard{}
	if n, err := standard_reader_write_to(
		&standard_reader{},
		io.Discard_To_Stream(discard),
	); n != 0 {
		t.Errorf("WriteTo: got %d, %v; want 0, nil", n, err)
	} else if err != nil {
		t.Errorf("WriteTo: got %d, %v; want 0, nil", n, err)
	}
}

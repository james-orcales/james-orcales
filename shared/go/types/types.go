// Package types answers what a name means across one module of Go source. It exists because a
// linter judges a bound, an enum member, and a call by the type a name stands for, and a parse
// tree states syntax alone. Every store is fixed storage the caller owns, thus a module of any
// size costs one tree at a time.
package types

import (
	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/constant"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/sim/aver/default"
)

// FILE_COUNT_MAXIMUM caps the files one module holds.
const FILE_COUNT_MAXIMUM = 1024

// PACKAGE_COUNT_MAXIMUM caps the packages one module holds.
const PACKAGE_COUNT_MAXIMUM = 256

// SYMBOL_COUNT_MAXIMUM caps the symbols one module holds. Slot zero holds no symbol.
const SYMBOL_COUNT_MAXIMUM = 65536

// TYPE_COUNT_MAXIMUM caps the types one module holds. Slot zero holds the invalid type.
const TYPE_COUNT_MAXIMUM = 65536

// MEMBER_COUNT_MAXIMUM caps the struct fields, tuple members, and union terms one module holds.
const MEMBER_COUNT_MAXIMUM = 32768

// BUCKET_COUNT is the name table width. It stands over the symbol count, thus a full module
// still meets short chains.
const BUCKET_COUNT = 131072

// DEPTH_MAXIMUM caps the nesting one type expression spends, thus a hostile source stops rather
// than the machine stack.
const DEPTH_MAXIMUM = 64

// NAME_SIZE_MAXIMUM caps one identifier.
const NAME_SIZE_MAXIMUM = 512

// PATH_SIZE_MAXIMUM caps one import path.
const PATH_SIZE_MAXIMUM = 1024

// COUNT_MAXIMUM caps every counter, and the name table is the widest thing one counter counts.
const COUNT_MAXIMUM = BUCKET_COUNT

// FILE_INDEX_MAXIMUM is the last file slot.
const FILE_INDEX_MAXIMUM = FILE_COUNT_MAXIMUM - 1

// PACKAGE_INDEX_MAXIMUM is the last package slot.
const PACKAGE_INDEX_MAXIMUM = PACKAGE_COUNT_MAXIMUM - 1

// SYMBOL_INDEX_MAXIMUM is the last symbol slot.
const SYMBOL_INDEX_MAXIMUM = SYMBOL_COUNT_MAXIMUM - 1

// TYPE_INDEX_MAXIMUM is the last type slot.
const TYPE_INDEX_MAXIMUM = TYPE_COUNT_MAXIMUM - 1

// MEMBER_INDEX_MAXIMUM is the last member slot.
const MEMBER_INDEX_MAXIMUM = MEMBER_COUNT_MAXIMUM - 1

// INDEX_MINIMUM is the slot that names nothing, which every arena keeps for the absent link.
const INDEX_MINIMUM = 0

// MEMBER_HOLE_FIRST is the member slot no member follows.
const MEMBER_HOLE_FIRST = 1

// SYMBOL_HOLE_FIRST is the first symbol slot the universe binds.
const SYMBOL_HOLE_FIRST = 1

// SYMBOL_HOLE_SECOND is the second symbol slot the universe binds.
const SYMBOL_HOLE_SECOND = 2

// SYMBOL_ABSENT names no symbol.
const SYMBOL_ABSENT Symbol_Index = 0

// TYPE_INVALID_INDEX is the slot of the type no source states.
const TYPE_INVALID_INDEX Type_Index = 0

// MEMBER_ABSENT names no member.
const MEMBER_ABSENT Member_Index = 0

// PACKAGE_UNIVERSE holds the predeclared names, thus a name no file states resolves there.
const PACKAGE_UNIVERSE Package_Index = 0

// ELEMENT_COUNT_UNKNOWN marks an array no literal counts, because a constant expression needs the
// value pass this module does not run.
const ELEMENT_COUNT_UNKNOWN Element_Count = -1

// ELEMENT_COUNT_MAXIMUM caps an array element_count a literal states.
const ELEMENT_COUNT_MAXIMUM = 1 << 40

// COUNT_FILE holds how many files the module bound.
const COUNT_FILE = 0

// COUNT_PACKAGE holds how many packages the module bound.
const COUNT_PACKAGE = 1

// COUNT_SYMBOL holds how many symbol slots the module spent.
const COUNT_SYMBOL = 2

// COUNT_TYPE holds how many type slots the module spent.
const COUNT_TYPE = 3

// COUNT_MEMBER holds how many member slots the module spent.
const COUNT_MEMBER = 4

// COUNT_DEPTH holds how deep the walk stands.
const COUNT_DEPTH = 5

// COUNT_BUCKET holds the name table slot the last hash landed in.
const COUNT_BUCKET = 6

// COUNT_PARAMETER holds how many type parameters stand in scope.
const COUNT_PARAMETER = 7

// COUNT_RESOLVE holds the symbol the resolve pass stands on.
const COUNT_RESOLVE = 8

// COUNT_CURRENT_FILE holds the file the pass stands in.
const COUNT_CURRENT_FILE = 9

// COUNT_RESULT holds the type the last fold wrote.
const COUNT_RESULT = 10

// COUNT_CURRENT_SYMBOL holds the symbol the pass stands on.
const COUNT_CURRENT_SYMBOL = 11

// COUNT_TARGET holds the package a qualifier named.
const COUNT_TARGET = 12

// COUNT_OWNER holds how deep the stack of types a member attaches to stands.
const COUNT_OWNER = 13

// COUNT_CURRENT_MEMBER holds the member the pass links.
const COUNT_CURRENT_MEMBER = 14

// COUNT_RECEIVER holds the named type the receiver of a method states.
const COUNT_RECEIVER = 15

// COUNT_CALL_KIND holds what the head of a call stands for.
const COUNT_CALL_KIND = 16

// COUNT_CALLEE holds the type the head of a call wears.
const COUNT_CALLEE = 17

// COUNT_FIRST holds the type the first argument of a call folded to.
const COUNT_FIRST = 18

// COUNT_OVER holds the type a fold reads through.
const COUNT_OVER = 19

// COUNT_CHAIN holds the member a search stands on.
const COUNT_CHAIN = 20

// COUNT_LEFT holds the type the left side of an operation folded to.
const COUNT_LEFT = 21

// COUNT_WORD holds how far the universe text the module copied stands read.
const COUNT_WORD = 22

// COUNT_SLOT_COUNT is the counter count one module holds.
const COUNT_SLOT_COUNT = 23

// NAME_SLOT holds the name a search or a binding reads. A name lives in a slot rather than in a
// parameter, because a body that binds one predeclared word can never see a name of every width.
const NAME_SLOT = 0

// NAME_SLOT_BOUND holds the name a binding wears while the type behind it folds, because folding
// a type reads names of its own into the first slot.
const NAME_SLOT_BOUND = 1

// NAME_SLOT_COUNT is the name slot count one module holds.
const NAME_SLOT_COUNT = 2

// PATH_SLOT holds the one path a package wears.
const PATH_SLOT = 0

// PATH_SLOT_COUNT is the path slot count one package holds.
const PATH_SLOT_COUNT = 1

// KIND_SLOT holds the one kind a symbol or a type wears.
const KIND_SLOT = 0

// KIND_SLOT_COUNT is the kind slot count one symbol or type holds.
const KIND_SLOT_COUNT = 1

// SOURCE_SLOT_COUNT is the source slot count one file holds.
const SOURCE_SLOT_COUNT = 1

// SOURCE_SLOT holds the source view of one file.
const SOURCE_SLOT = 0

// UNIVERSE_WORDS states every predeclared word, in the order the universe binds them. One text
// holds them all, thus the module copies it once and every predeclared name views that copy.
const UNIVERSE_WORDS = "bool string int int8 int16 int32 int64 uint uint8 uint16 uint32 " +
	"uint64 uintptr float32 float64 complex64 complex128 byte rune any comparable error " +
	"len cap make new append copy delete panic recover close min max clear complex real " +
	"imag print println true false iota nil"

// UNIVERSE_WORD_SIZE is the storage the universe text spends. It stands one byte over the text,
// thus the cursor that reads the final word still names a slot.
const UNIVERSE_WORD_SIZE = 272

// TYPE_PARAMETER_MAXIMUM caps the type parameters one declaration states.
const TYPE_PARAMETER_MAXIMUM = 32

// FLAG_FAILED marks a module that refused something.
const FLAG_FAILED = 0

// FLAG_COUNT is the flag count one module holds.
const FLAG_COUNT = 1

// CAUSE_SLOT holds why the module refused.
const CAUSE_SLOT = 0

// CAUSE_SLOT_COUNT is the cause slot count one module holds.
const CAUSE_SLOT_COUNT = 1

// SYMBOL_UNKNOWN names a slot no declaration filled.
const SYMBOL_UNKNOWN Symbol_Kind = 0

// SYMBOL_CONSTANT names a constant declaration.
const SYMBOL_CONSTANT Symbol_Kind = 1

// SYMBOL_VARIABLE names a variable declaration.
const SYMBOL_VARIABLE Symbol_Kind = 2

// SYMBOL_TYPE names a type declaration or a predeclared type.
const SYMBOL_TYPE Symbol_Kind = 3

// SYMBOL_FUNCTION names a function declaration.
const SYMBOL_FUNCTION Symbol_Kind = 4

// SYMBOL_METHOD names a function declaration that carries a receiver.
const SYMBOL_METHOD Symbol_Kind = 5

// SYMBOL_IMPORT names an import binding, which stands in one file and never in a package.
const SYMBOL_IMPORT Symbol_Kind = 6

// SYMBOL_BUILTIN names a predeclared function.
const SYMBOL_BUILTIN Symbol_Kind = 7

// SYMBOL_FIELD names one field of a struct.
const SYMBOL_FIELD Symbol_Kind = 8

// SYMBOL_PARAMETER names one parameter of a signature.
const SYMBOL_PARAMETER Symbol_Kind = 9

// SYMBOL_RESULT names one result of a signature.
const SYMBOL_RESULT Symbol_Kind = 10

// SYMBOL_TYPE_PARAMETER names one type parameter of a declaration.
const SYMBOL_TYPE_PARAMETER Symbol_Kind = 11

// SYMBOL_KIND_MINIMUM is the first symbol kind.
const SYMBOL_KIND_MINIMUM = uint8(SYMBOL_UNKNOWN)

// SYMBOL_KIND_MAXIMUM is the last symbol kind.
const SYMBOL_KIND_MAXIMUM = uint8(SYMBOL_TYPE_PARAMETER)

// TYPE_INVALID names the type no source states.
const TYPE_INVALID Type_Kind = 0

// TYPE_BOOLEAN names the predeclared bool.
const TYPE_BOOLEAN Type_Kind = 1

// TYPE_STRING names the predeclared string.
const TYPE_STRING Type_Kind = 2

// TYPE_INT names the predeclared int.
const TYPE_INT Type_Kind = 3

// TYPE_INT_8 names the predeclared int8.
const TYPE_INT_8 Type_Kind = 4

// TYPE_INT_16 names the predeclared int16.
const TYPE_INT_16 Type_Kind = 5

// TYPE_INT_32 names the predeclared int32, which rune also names.
const TYPE_INT_32 Type_Kind = 6

// TYPE_INT_64 names the predeclared int64.
const TYPE_INT_64 Type_Kind = 7

// TYPE_UINT names the predeclared uint.
const TYPE_UINT Type_Kind = 8

// TYPE_UINT_8 names the predeclared uint8, which byte also names.
const TYPE_UINT_8 Type_Kind = 9

// TYPE_UINT_16 names the predeclared uint16.
const TYPE_UINT_16 Type_Kind = 10

// TYPE_UINT_32 names the predeclared uint32.
const TYPE_UINT_32 Type_Kind = 11

// TYPE_UINT_64 names the predeclared uint64.
const TYPE_UINT_64 Type_Kind = 12

// TYPE_UINTPTR names the predeclared uintptr.
const TYPE_UINTPTR Type_Kind = 13

// TYPE_FLOAT_32 names the predeclared float32.
const TYPE_FLOAT_32 Type_Kind = 14

// TYPE_FLOAT_64 names the predeclared float64.
const TYPE_FLOAT_64 Type_Kind = 15

// TYPE_COMPLEX_64 names the predeclared complex64.
const TYPE_COMPLEX_64 Type_Kind = 16

// TYPE_COMPLEX_128 names the predeclared complex128.
const TYPE_COMPLEX_128 Type_Kind = 17

// TYPE_UNTYPED_BOOLEAN names the type a comparison folds to before a context names it.
const TYPE_UNTYPED_BOOLEAN Type_Kind = 18

// TYPE_UNTYPED_INTEGER names the type an integer literal folds to.
const TYPE_UNTYPED_INTEGER Type_Kind = 19

// TYPE_UNTYPED_RUNE names the type a character literal folds to.
const TYPE_UNTYPED_RUNE Type_Kind = 20

// TYPE_UNTYPED_FLOAT names the type a floating literal folds to.
const TYPE_UNTYPED_FLOAT Type_Kind = 21

// TYPE_UNTYPED_COMPLEX names the type an imaginary literal folds to.
const TYPE_UNTYPED_COMPLEX Type_Kind = 22

// TYPE_UNTYPED_STRING names the type a string literal folds to.
const TYPE_UNTYPED_STRING Type_Kind = 23

// TYPE_UNTYPED_NIL names the type the nil word folds to.
const TYPE_UNTYPED_NIL Type_Kind = 24

// TYPE_POINTER names a pointer type.
const TYPE_POINTER Type_Kind = 25

// TYPE_ARRAY names an array type.
const TYPE_ARRAY Type_Kind = 26

// TYPE_SLICE names a slice type.
const TYPE_SLICE Type_Kind = 27

// TYPE_MAP names a map type.
const TYPE_MAP Type_Kind = 28

// TYPE_CHANNEL names a channel type of any direction.
const TYPE_CHANNEL Type_Kind = 29

// TYPE_FUNCTION names a signature.
const TYPE_FUNCTION Type_Kind = 30

// TYPE_STRUCTURE names a struct type.
const TYPE_STRUCTURE Type_Kind = 31

// TYPE_CONSTRAINT names an interface, which this grammar admits as a type constraint alone.
const TYPE_CONSTRAINT Type_Kind = 32

// TYPE_NAMED names a type one declaration states.
const TYPE_NAMED Type_Kind = 33

// TYPE_PARAMETER names one type parameter of a declaration.
const TYPE_PARAMETER Type_Kind = 34

// TYPE_TUPLE names the parameters or the results of a signature.
const TYPE_TUPLE Type_Kind = 35

// TYPE_KIND_MINIMUM is the first type kind.
const TYPE_KIND_MINIMUM = uint8(TYPE_INVALID)

// TYPE_KIND_MAXIMUM is the last type kind.
const TYPE_KIND_MAXIMUM = uint8(TYPE_TUPLE)

// BASIC_KIND_MINIMUM is the first predeclared type kind.
const BASIC_KIND_MINIMUM = uint8(TYPE_BOOLEAN)

// BASIC_KIND_MAXIMUM is the last predeclared type kind.
const BASIC_KIND_MAXIMUM = uint8(TYPE_UNTYPED_NIL)

// FAILURE_NONE marks a module that refused nothing.
const FAILURE_NONE Failure_Code = 0

// FAILURE_FILE_COUNT marks a module that met more files than it holds.
const FAILURE_FILE_COUNT Failure_Code = 1

// FAILURE_PACKAGE_COUNT marks a module that met more packages than it holds.
const FAILURE_PACKAGE_COUNT Failure_Code = 2

// FAILURE_SYMBOL_COUNT marks a module that met more symbols than it holds.
const FAILURE_SYMBOL_COUNT Failure_Code = 3

// FAILURE_TYPE_COUNT marks a module that met more types than it holds.
const FAILURE_TYPE_COUNT Failure_Code = 4

// FAILURE_MEMBER_COUNT marks a module that met more members than it holds.
const FAILURE_MEMBER_COUNT Failure_Code = 5

// FAILURE_NAME_SIZE marks a name wider than one slot holds.
const FAILURE_NAME_SIZE Failure_Code = 6

// FAILURE_PATH_SIZE marks an import path wider than one slot holds.
const FAILURE_PATH_SIZE Failure_Code = 7

// FAILURE_NESTING_DEPTH marks a type expression that nests deeper than the walk holds.
const FAILURE_NESTING_DEPTH Failure_Code = 8

// FAILURE_UNKNOWN_NAME marks a name that resolves nowhere.
const FAILURE_UNKNOWN_NAME Failure_Code = 9

// FAILURE_DUPLICATE_NAME marks a name one package states twice.
const FAILURE_DUPLICATE_NAME Failure_Code = 10

// FAILURE_UNKNOWN_IMPORT marks an import path no file of this module declares.
const FAILURE_UNKNOWN_IMPORT Failure_Code = 11

// FAILURE_TYPE_CYCLE marks a named type that stands on itself.
const FAILURE_TYPE_CYCLE Failure_Code = 12

// FAILURE_CODE_MINIMUM is the first failure code.
const FAILURE_CODE_MINIMUM = uint8(FAILURE_NONE)

// FAILURE_CODE_MAXIMUM is the last failure code.
const FAILURE_CODE_MAXIMUM = uint8(FAILURE_TYPE_CYCLE)

// CAUSE_MINIMUM is the first cause a refusal states, which never names a clean module.
const CAUSE_MINIMUM = uint8(FAILURE_FILE_COUNT)

// CAUSE_MAXIMUM is the last cause a refusal states.
const CAUSE_MAXIMUM = uint8(FAILURE_TYPE_CYCLE)

// MESSAGE_SIZE_MAXIMUM is the byte count of the longest failure sentence.
const MESSAGE_SIZE_MAXIMUM = 55

// MESSAGE_HOLE_FIRST is the width no sentence spans.
const MESSAGE_HOLE_FIRST = 1

// MESSAGE_HOLE_SECOND is the second width no sentence spans.
const MESSAGE_HOLE_SECOND = 2

// Boolean is a true or false report about one module.
type Boolean bool

// Boolean_Invariants states both module reports as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The module report is true.").
		Ensure()
}

// Name is one identifier, a view of storage the caller or the module owns.
type Name []byte

// Name_Invariants caps one identifier.
func Name_Invariants(value Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), INDEX_MINIMUM, NAME_SIZE_MAXIMUM).
		Ensure()
}

// Path is one import path, which is what binds a file to its package.
type Path string

// Path_Invariants caps one import path.
func Path_Invariants(value Path, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), INDEX_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Message is one failure sentence.
type Message string

// Message_Invariants caps one failure sentence. No sentence spans one byte or two, because a
// sentence states what to do about a refusal and no word of that width says anything.
func Message_Invariants(value Message, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(len(value), INDEX_MINIMUM, MESSAGE_SIZE_MAXIMUM,
			MESSAGE_HOLE_FIRST, MESSAGE_HOLE_SECOND, MESSAGE_HOLE_SECOND,
			MESSAGE_HOLE_SECOND).
		Ensure()
}

// Count is one counter of the module. Every counter lives in a slot, thus no body owes the whole
// counter domain at a step that can only see one part of it.
type Count int

// Count_Invariants states the widest thing one counter counts.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), INDEX_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Element_Count is how many elements one array type holds, or ELEMENT_COUNT_UNKNOWN where no
// literal states it.
type Element_Count int64

// Element_Count_Invariants admits the unknown element_count, which is a element_count of its own.
func Element_Count_Invariants(value Element_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(ELEMENT_COUNT_UNKNOWN), ELEMENT_COUNT_MAXIMUM).
		Ensure()
}

// File_Index is one file slot of the module.
type File_Index int32

// File_Index_Invariants states the complete file arena.
func File_Index_Invariants(value File_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, FILE_INDEX_MAXIMUM).
		Ensure()
}

// Package_Index is one package slot of the module.
type Package_Index int32

// Package_Index_Invariants states the complete package arena.
func Package_Index_Invariants(value Package_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, PACKAGE_INDEX_MAXIMUM).
		Ensure()
}

// Symbol_Index is one symbol slot of the module. Slot zero holds no symbol.
type Symbol_Index int32

// Symbol_Index_Invariants states the complete symbol arena.
func Symbol_Index_Invariants(value Symbol_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, SYMBOL_INDEX_MAXIMUM).
		Ensure()
}

// Symbol_Successor is the symbol after one symbol in its name table chain.
type Symbol_Successor int32

// Symbol_Successor_Invariants states the complete symbol arena. A chain link is never the first
// slot, because a symbol links to a symbol the table already held.
func Symbol_Successor_Invariants(value Symbol_Successor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, SYMBOL_INDEX_MAXIMUM).
		Ensure()
}

// Symbol_Head is the first symbol one file declared.
type Symbol_Head int32

// Symbol_Head_Invariants states the complete symbol arena.
func Symbol_Head_Invariants(value Symbol_Head, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, SYMBOL_INDEX_MAXIMUM).
		Ensure()
}

// Symbol_Count is how many symbols one file declared.
type Symbol_Count int32

// Symbol_Count_Invariants states the symbol count one file can spend.
func Symbol_Count_Invariants(value Symbol_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, SYMBOL_INDEX_MAXIMUM).
		Ensure()
}

// Type_Index is one type slot of the module. Slot zero holds the invalid type.
type Type_Index int32

// Type_Index_Invariants states the complete type arena.
func Type_Index_Invariants(value Type_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, TYPE_INDEX_MAXIMUM).
		Ensure()
}

// Type_Element is the type one type stands over: a pointee, an element, a map value, the
// base type of a name, or the result tuple of a signature.
type Type_Element int32

// Type_Element_Invariants states the complete type arena.
func Type_Element_Invariants(value Type_Element, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, TYPE_INDEX_MAXIMUM).
		Ensure()
}

// Type_Key is the key type of a map or the parameter tuple of a signature.
type Type_Key int32

// Type_Key_Invariants states the complete type arena.
func Type_Key_Invariants(value Type_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, TYPE_INDEX_MAXIMUM).
		Ensure()
}

// Type_Symbol is the declaration one named type stands for.
type Type_Symbol int32

// Type_Symbol_Invariants excludes the first predeclared symbols. A type stands for a declaration
// a file states, and the first symbol slots hold the words the universe binds.
func Type_Symbol_Invariants(value Type_Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int32(int32(value), INDEX_MINIMUM, SYMBOL_INDEX_MAXIMUM,
			SYMBOL_HOLE_FIRST, SYMBOL_HOLE_SECOND,
			SYMBOL_HOLE_SECOND, SYMBOL_HOLE_SECOND).
		Ensure()
}

// Member_Index is one member slot of the module. Slot zero holds no member.
type Member_Index int32

// Member_Index_Invariants states the complete member arena.
func Member_Index_Invariants(value Member_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, MEMBER_INDEX_MAXIMUM).
		Ensure()
}

// Member_Head is the first member one type holds.
type Member_Head int32

// Member_Head_Invariants states the complete member arena.
func Member_Head_Invariants(value Member_Head, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, MEMBER_INDEX_MAXIMUM).
		Ensure()
}

// Member_Successor is the member after one member of the same type.
type Member_Successor int32

// Member_Successor_Invariants excludes the first member slot. One member links to a member the
// arena took after it, thus the first slot the arena ever takes follows nothing.
func Member_Successor_Invariants(value Member_Successor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int32(int32(value), INDEX_MINIMUM, MEMBER_INDEX_MAXIMUM,
			MEMBER_HOLE_FIRST, MEMBER_HOLE_FIRST, MEMBER_HOLE_FIRST, MEMBER_HOLE_FIRST).
		Ensure()
}

// Member_Symbol is the name one member wears, or no symbol where it wears none.
type Member_Symbol int32

// Member_Symbol_Invariants excludes the first predeclared symbols. A field wears a name a file
// states, and the first symbol slots hold the words the universe binds before any file runs.
func Member_Symbol_Invariants(value Member_Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int32(int32(value), INDEX_MINIMUM, SYMBOL_INDEX_MAXIMUM,
			SYMBOL_HOLE_FIRST, SYMBOL_HOLE_SECOND,
			SYMBOL_HOLE_SECOND, SYMBOL_HOLE_SECOND).
		Ensure()
}

// Member_Type is the type one member wears.
type Member_Type int32

// Member_Type_Invariants states the complete type arena.
func Member_Type_Invariants(value Member_Type, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, TYPE_INDEX_MAXIMUM).
		Ensure()
}

// VALUE_CONSTANT names a constant declaration.
const VALUE_CONSTANT Value_Kind = Value_Kind(SYMBOL_CONSTANT)

// VALUE_VARIABLE names a variable declaration.
const VALUE_VARIABLE Value_Kind = Value_Kind(SYMBOL_VARIABLE)

// DECLARED_TYPE names a type declaration.
const DECLARED_TYPE Declared_Kind = Declared_Kind(SYMBOL_TYPE)

// DECLARED_FUNCTION names a function declaration.
const DECLARED_FUNCTION Declared_Kind = Declared_Kind(SYMBOL_FUNCTION)

// Value_Kind names a declaration that binds a value. It wears a type of its own, because the
// body that reads such a declaration meets these two kinds and no other.
type Value_Kind uint8

// Value_Kind_Invariants states both kinds a value declaration binds.
func Value_Kind_Invariants(value Value_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(VALUE_CONSTANT), uint8(VALUE_VARIABLE)).
		Ensure()
}

// Declared_Kind names a declaration that binds a package name. It wears a type of its own for the
// reason a value declaration does.
type Declared_Kind uint8

// Declared_Kind_Invariants states every kind that binds a package name.
func Declared_Kind_Invariants(value Declared_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value), uint8(VALUE_CONSTANT), uint8(VALUE_VARIABLE),
			uint8(DECLARED_TYPE), uint8(DECLARED_FUNCTION)).
		Ensure()
}

// Symbol_Kind names what one symbol declares.
type Symbol_Kind uint8

// Symbol_Kind_Invariants states every kind a symbol declares.
func Symbol_Kind_Invariants(value Symbol_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), SYMBOL_KIND_MINIMUM, SYMBOL_KIND_MAXIMUM).
		Ensure()
}

// Type_Kind names what one type is.
type Type_Kind uint8

// Type_Kind_Invariants states every kind a type wears.
func Type_Kind_Invariants(value Type_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TYPE_KIND_MINIMUM, TYPE_KIND_MAXIMUM).
		Ensure()
}

// Basic_Kind names one predeclared type. It wears a type of its own, because the body that
// builds the universe meets the predeclared kinds alone.
type Basic_Kind uint8

// Basic_Kind_Invariants states every predeclared type kind.
func Basic_Kind_Invariants(value Basic_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BASIC_KIND_MINIMUM, BASIC_KIND_MAXIMUM).
		Ensure()
}

// Failure_Code names why a module refused, or that it refused nothing.
type Failure_Code uint8

// Failure_Code_Invariants states every code a caller reads back.
func Failure_Code_Invariants(value Failure_Code, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), FAILURE_CODE_MINIMUM, FAILURE_CODE_MAXIMUM).
		Ensure()
}

// Cause names why a module refused. It never names a clean module, thus a refusal that states
// no cause is a refusal this package cannot write.
type Cause uint8

// Cause_Invariants states every cause a refusal names.
func Cause_Invariants(value Cause, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), CAUSE_MINIMUM, CAUSE_MAXIMUM).
		Ensure()
}

// Symbol is one declared name: what it declares, what it is called, where it stands, and the
// type it wears.
type Symbol struct {
	// Kind names what the symbol declares.
	Kind Symbol_Kind
	// Name holds the identifier, a view of the source of the file that states it.
	Name Name
	// File is the file that states the symbol.
	File File_Index
	// Owner is the package the symbol belongs to.
	Owner Package_Index
	// Type is the type the symbol wears, or the invalid type until the resolve pass runs.
	Type Type_Index
	// Next is the symbol after this one in its name table chain.
	Next Symbol_Successor
	// Target is the package an import binding names, or the universe where it names none.
	Target Target_Package
}

// Symbol_Invariants composes every link one symbol holds.
func Symbol_Invariants(value Symbol, namespace aver.Namespace) {
	Symbol_Kind_Invariants(value.Kind, namespace)
	Name_Invariants(value.Name, namespace)
	File_Index_Invariants(value.File, namespace)
	Package_Index_Invariants(value.Owner, namespace)
	Type_Index_Invariants(value.Type, namespace)
	Symbol_Successor_Invariants(value.Next, namespace)
	Target_Package_Invariants(value.Target, namespace)
}

// Type is one type of the module.
type Type struct {
	// Kind names what the type is.
	Kind Type_Kind
	// Element is the pointee, the element, the map value, the base type of a name, or
	// the result tuple of a signature.
	Element Type_Element
	// Key is the key type of a map or the parameter tuple of a signature.
	Key Type_Key
	// Members is the first field, term, or tuple member.
	Members Member_Head
	// Symbol is the declaration a named type stands for.
	Symbol Type_Symbol
	// Element_Count is the element count of an array.
	Element_Count Element_Count
}

// Type_Invariants composes every link one type holds.
func Type_Invariants(value Type, namespace aver.Namespace) {
	Type_Kind_Invariants(value.Kind, namespace)
	Type_Element_Invariants(value.Element, namespace)
	Type_Key_Invariants(value.Key, namespace)
	Member_Head_Invariants(value.Members, namespace)
	Type_Symbol_Invariants(value.Symbol, namespace)
	Element_Count_Invariants(value.Element_Count, namespace)
}

// Member is one field of a struct, one member of a tuple, or one term of a constraint.
type Member struct {
	// Symbol is the name the member wears, or no symbol where it wears none.
	Symbol Member_Symbol
	// Type is the type the member wears.
	Type Member_Type
	// Next is the member after this one.
	Next Member_Successor
}

// Member_Invariants composes every link one member holds.
func Member_Invariants(value Member, namespace aver.Namespace) {
	Member_Symbol_Invariants(value.Symbol, namespace)
	Member_Type_Invariants(value.Type, namespace)
	Member_Successor_Invariants(value.Next, namespace)
}

// File is one source file bound to its package and to the symbols it declares.
type File struct {
	// Source holds the source view the caller owns.
	Source token.Source
	// Owner is the package the file belongs to.
	Owner Package_Index
	// First is the first symbol the declare pass wrote for this file.
	First Symbol_Head
	// Count is how many symbols the declare pass wrote for this file.
	Count Symbol_Count
}

// File_Invariants composes every link one file holds.
func File_Invariants(value File, namespace aver.Namespace) {
	token.Source_Invariants(value.Source, namespace)
	Package_Index_Invariants(value.Owner, namespace)
	Symbol_Head_Invariants(value.First, namespace)
	Symbol_Count_Invariants(value.Count, namespace)
}

// Package is one import path and the members bound under it.
type Package struct {
	// Path holds the import path the package answers to.
	Path Path
}

// Package_Invariants states the storage one package holds.
func Package_Invariants(value Package, namespace aver.Namespace) {
	Path_Invariants(value.Path, namespace)
}

// File_Storage holds every file in caller-owned storage.
type File_Storage []File

// File_Storage_Invariants fixes the file arena and prevents growth.
func File_Storage_Invariants(value File_Storage, _ aver.Namespace) {
	aver.Always(len(value) == FILE_COUNT_MAXIMUM, "Module file storage has complete length.")
	aver.Always(cap(value) == FILE_COUNT_MAXIMUM, "Module file storage cannot grow.")
}

// Package_Storage holds every package in caller-owned storage.
type Package_Storage []Package

// Package_Storage_Invariants fixes the package arena and prevents growth.
func Package_Storage_Invariants(value Package_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == PACKAGE_COUNT_MAXIMUM, "Module package storage has complete length.",
	)
	aver.Always(cap(value) == PACKAGE_COUNT_MAXIMUM, "Module package storage cannot grow.")
}

// Symbol_Storage holds every symbol in caller-owned storage.
type Symbol_Storage []Symbol

// Symbol_Storage_Invariants fixes the symbol arena and prevents growth.
func Symbol_Storage_Invariants(value Symbol_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == SYMBOL_COUNT_MAXIMUM, "Module symbol storage has complete length.",
	)
	aver.Always(cap(value) == SYMBOL_COUNT_MAXIMUM, "Module symbol storage cannot grow.")
}

// Type_Storage holds every type in caller-owned storage.
type Type_Storage []Type

// Type_Storage_Invariants fixes the type arena and prevents growth.
func Type_Storage_Invariants(value Type_Storage, _ aver.Namespace) {
	aver.Always(len(value) == TYPE_COUNT_MAXIMUM, "Module type storage has complete length.")
	aver.Always(cap(value) == TYPE_COUNT_MAXIMUM, "Module type storage cannot grow.")
}

// Member_Storage holds every member in caller-owned storage.
type Member_Storage []Member

// Member_Storage_Invariants fixes the member arena and prevents growth.
func Member_Storage_Invariants(value Member_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == MEMBER_COUNT_MAXIMUM, "Module member storage has complete length.",
	)
	aver.Always(cap(value) == MEMBER_COUNT_MAXIMUM, "Module member storage cannot grow.")
}

// Bucket_Storage holds every name-table head in caller-owned storage.
type Bucket_Storage []Symbol_Index

// Bucket_Storage_Invariants fixes the name table and prevents growth.
func Bucket_Storage_Invariants(value Bucket_Storage, _ aver.Namespace) {
	aver.Always(len(value) == BUCKET_COUNT, "Module bucket storage has complete length.")
	aver.Always(cap(value) == BUCKET_COUNT, "Module bucket storage cannot grow.")
}

// Node_Storage holds every active tree depth in caller-owned storage.
type Node_Storage []ast.Index

// Node_Storage_Invariants fixes the tree stack and prevents growth.
func Node_Storage_Invariants(value Node_Storage, _ aver.Namespace) {
	aver.Always(len(value) == DEPTH_MAXIMUM, "Module node storage has complete length.")
	aver.Always(cap(value) == DEPTH_MAXIMUM, "Module node storage cannot grow.")
}

// Parameter_Storage holds every in-scope type parameter in caller-owned storage.
type Parameter_Storage []Symbol_Index

// Parameter_Storage_Invariants fixes parameter storage and prevents growth.
func Parameter_Storage_Invariants(value Parameter_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == TYPE_PARAMETER_MAXIMUM,
		"Module parameter storage has complete length.",
	)
	aver.Always(cap(value) == TYPE_PARAMETER_MAXIMUM, "Module parameter storage cannot grow.")
}

// Owner_Storage holds every active member owner in caller-owned storage.
type Owner_Storage []Type_Index

// Owner_Storage_Invariants fixes owner storage and prevents growth.
func Owner_Storage_Invariants(value Owner_Storage, _ aver.Namespace) {
	aver.Always(len(value) == DEPTH_MAXIMUM, "Module owner storage has complete length.")
	aver.Always(cap(value) == DEPTH_MAXIMUM, "Module owner storage cannot grow.")
}

// Module_Names_Fields names the two unlike source views one pass retains.
type Module_Names_Fields struct {
	// Value is the name the current lookup reads.
	Value Name
	// Bound survives while another name is read.
	Bound Name
}

// Module_Names_Fields_Invariants composes both retained source views.
func Module_Names_Fields_Invariants(value Module_Names_Fields, namespace aver.Namespace) {
	Name_Invariants(value.Value, namespace)
	Name_Invariants(value.Bound, namespace)
}

// Module_Names_Fields_Stored prevents one pass from proving stale name lengths.
type Module_Names_Fields_Stored interface{}

// Module_Names_Fields_Stored_Invariants fixes retained-name representation.
func Module_Names_Fields_Stored_Invariants(
	value Module_Names_Fields_Stored, _ aver.Namespace,
) {
	_, valid := value.(Module_Names_Fields)
	aver.Always(valid == (value != nil), "Module names have expected storage type.")
}

// Module_Names_Envelope keeps retained views behind one representation boundary.
type Module_Names_Envelope struct {
	// Module_Names_Fields stays embedded so passes retain concrete field access.
	Module_Names_Fields
}

// Module_Names_Envelope_Invariants composes retained-name storage.
func Module_Names_Envelope_Invariants(value Module_Names_Envelope, namespace aver.Namespace) {
	Module_Names_Fields_Invariants(value.Module_Names_Fields, namespace)
}

// Module_Names is the mutable name state one pass carries.
type Module_Names Module_Names_Envelope

// Module_Names_Invariants fixes name representation without proving stale views.
func Module_Names_Invariants(value Module_Names, namespace aver.Namespace) {
	Module_Names_Fields_Stored_Invariants(
		Module_Names_Fields_Stored(value.Module_Names_Fields), namespace,
	)
}

// Word_Storage holds universe text in caller-owned storage.
type Word_Storage []byte

// Word_Storage_Invariants fixes universe-word storage and prevents growth.
func Word_Storage_Invariants(value Word_Storage, _ aver.Namespace) {
	aver.Always(len(value) == UNIVERSE_WORD_SIZE, "Module word storage has complete length.")
	aver.Always(cap(value) == UNIVERSE_WORD_SIZE, "Module word storage cannot grow.")
}

// Count_Storage holds every module cursor in caller-owned storage.
type Count_Storage []Count

// Count_Storage_Invariants fixes cursor storage and prevents growth.
func Count_Storage_Invariants(value Count_Storage, _ aver.Namespace) {
	aver.Always(len(value) == COUNT_SLOT_COUNT, "Module count storage has complete length.")
	aver.Always(cap(value) == COUNT_SLOT_COUNT, "Module count storage cannot grow.")
}

// Module_Cause_Fields holds the first refusal one module retains.
type Module_Cause_Fields struct {
	// Failure survives later passes so the first cause remains the answer.
	Failure Failure_Code
}

// Module_Cause_Fields_Invariants composes the retained refusal.
func Module_Cause_Fields_Invariants(value Module_Cause_Fields, namespace aver.Namespace) {
	Failure_Code_Invariants(value.Failure, namespace)
}

// Module_Cause_Fields_Stored prevents one pass from proving every refusal code.
type Module_Cause_Fields_Stored interface{}

// Module_Cause_Fields_Stored_Invariants fixes refusal representation.
func Module_Cause_Fields_Stored_Invariants(
	value Module_Cause_Fields_Stored, _ aver.Namespace,
) {
	_, valid := value.(Module_Cause_Fields)
	aver.Always(valid == (value != nil), "Module cause has expected storage type.")
}

// Module_Cause_Envelope keeps mutable refusal state behind one representation boundary.
type Module_Cause_Envelope struct {
	// Module_Cause_Fields stays embedded so passes retain concrete field access.
	Module_Cause_Fields
}

// Module_Cause_Envelope_Invariants composes retained refusal storage.
func Module_Cause_Envelope_Invariants(value Module_Cause_Envelope, namespace aver.Namespace) {
	Module_Cause_Fields_Invariants(value.Module_Cause_Fields, namespace)
}

// Module_Causes is the mutable refusal state one module carries.
type Module_Causes Module_Cause_Envelope

// Module_Causes_Invariants fixes refusal representation without proving stale state.
func Module_Causes_Invariants(value Module_Causes, namespace aver.Namespace) {
	Module_Cause_Fields_Stored_Invariants(
		Module_Cause_Fields_Stored(value.Module_Cause_Fields), namespace,
	)
}

// Module_Flag_Fields holds whether one module already refused input.
type Module_Flag_Fields struct {
	// Failed survives later passes so none may erase an earlier refusal.
	Failed Boolean
}

// Module_Flag_Fields_Invariants composes the retained report.
func Module_Flag_Fields_Invariants(value Module_Flag_Fields, namespace aver.Namespace) {
	Boolean_Invariants(value.Failed, namespace)
}

// Module_Flag_Fields_Stored prevents one pass from proving both mutable reports.
type Module_Flag_Fields_Stored interface{}

// Module_Flag_Fields_Stored_Invariants fixes report representation.
func Module_Flag_Fields_Stored_Invariants(
	value Module_Flag_Fields_Stored, _ aver.Namespace,
) {
	_, valid := value.(Module_Flag_Fields)
	aver.Always(valid == (value != nil), "Module flag has expected storage type.")
}

// Module_Flag_Envelope keeps mutable report state behind one representation boundary.
type Module_Flag_Envelope struct {
	// Module_Flag_Fields stays embedded so passes retain concrete field access.
	Module_Flag_Fields
}

// Module_Flag_Envelope_Invariants composes retained report storage.
func Module_Flag_Envelope_Invariants(value Module_Flag_Envelope, namespace aver.Namespace) {
	Module_Flag_Fields_Invariants(value.Module_Flag_Fields, namespace)
}

// Module_Flags is the mutable report state one module carries.
type Module_Flags Module_Flag_Envelope

// Module_Flags_Invariants fixes report representation without proving stale state.
func Module_Flags_Invariants(value Module_Flags, namespace aver.Namespace) {
	Module_Flag_Fields_Stored_Invariants(
		Module_Flag_Fields_Stored(value.Module_Flag_Fields), namespace,
	)
}

// Module_Fields states the concrete module layout independently of its validated view.
type Module_Fields struct {
	// Files holds every file the caller bound.
	Files File_Storage
	// Packages holds every package the files named.
	Packages Package_Storage
	// Symbols holds every declared name. Slot zero holds no symbol.
	Symbols Symbol_Storage
	// Types holds every type. Slot zero holds the invalid type.
	Types Type_Storage
	// Members holds every field, term, and tuple member. Slot zero holds no member.
	Members Member_Storage
	// Buckets holds the head of each name table chain.
	Buckets Bucket_Storage
	// Nodes holds the tree slot the walk stands on at each depth.
	Nodes Node_Storage
	// Parameters holds the type parameters that stand in scope.
	Parameters Parameter_Storage
	// Owners holds the type a member attaches to at each nesting depth. A type index lives in
	// a slot rather than in a parameter, because a body that builds a struct never meets the
	// arena slots the universe holds and a parameter would owe them.
	Owners Owner_Storage
	// Names holds the name a search or a binding reads.
	Names Module_Names
	// Words holds the universe text, thus a predeclared name views storage the module owns.
	Words Word_Storage
	// Counts holds every counter and cursor the passes keep.
	Counts Count_Storage
	// Causes holds why the module refused, or FAILURE_NONE while it stands.
	Causes Module_Causes
	// Flags holds whether the module refused something.
	Flags Module_Flags
}

// Module_Fields_Invariants states every caller-owned store once.
func Module_Fields_Invariants(value Module_Fields, namespace aver.Namespace) {
	File_Storage_Invariants(value.Files, namespace)
	Package_Storage_Invariants(value.Packages, namespace)
	Symbol_Storage_Invariants(value.Symbols, namespace)
	Type_Storage_Invariants(value.Types, namespace)
	Member_Storage_Invariants(value.Members, namespace)
	Bucket_Storage_Invariants(value.Buckets, namespace)
	Node_Storage_Invariants(value.Nodes, namespace)
	Parameter_Storage_Invariants(value.Parameters, namespace)
	Owner_Storage_Invariants(value.Owners, namespace)
	Module_Names_Invariants(value.Names, namespace)
	Word_Storage_Invariants(value.Words, namespace)
	Count_Storage_Invariants(value.Counts, namespace)
	Module_Causes_Invariants(value.Causes, namespace)
	Module_Flags_Invariants(value.Flags, namespace)
}

// Module_Fields_Stored prevents one recursive step from revalidating every arena.
type Module_Fields_Stored interface{}

// Module_Fields_Stored_Invariants fixes module representation.
func Module_Fields_Stored_Invariants(value Module_Fields_Stored, _ aver.Namespace) {
	_, valid := value.(Module_Fields)
	aver.Always(valid == (value != nil), "Module has expected storage type.")
}

// Module_Envelope keeps caller storage behind one representation boundary.
type Module_Envelope struct {
	// Module_Fields stays embedded so callers retain concrete field access.
	Module_Fields
}

// Module_Envelope_Invariants composes concrete caller storage.
func Module_Envelope_Invariants(value Module_Envelope, namespace aver.Namespace) {
	Module_Fields_Invariants(value.Module_Fields, namespace)
}

// Module is the whole analysis. Every cursor lives in a caller-owned slice slot, because a cursor
// field would owe its whole domain at every step that receives the state and a step deep in one
// declaration can never see a cursor at zero.
type Module Module_Envelope

// Module_Invariants fixes representation without revalidating every arena in recursive steps.
func Module_Invariants(value Module, namespace aver.Namespace) {
	Module_Fields_Stored_Invariants(Module_Fields_Stored(value.Module_Fields), namespace)
}

// Module_Handle gives caller-owned module storage one identity.
type Module_Handle *Module

// Module_Handle_Invariants composes present module storage.
func Module_Handle_Invariants(value Module_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Module_Invariants(*value, namespace)
}

// TREE_ROOT is the arena slot the parser writes the file node into, thus a caller hands over the
// tree alone and never the root it already knows.
const TREE_ROOT ast.Index = 1

// Target_Package is the package one import binding names.
type Target_Package int32

// Target_Package_Invariants states the complete package arena.
func Target_Package_Invariants(value Target_Package, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, PACKAGE_INDEX_MAXIMUM).
		Ensure()
}

// Reset clears the module and states the universe. A caller reuses one module for one analysis,
// thus the predeclared names stand again without a file that declares them.
func Reset(subject Module_Handle) {
	Module_Handle_Invariants(subject, "reset.subject")
	Module_Fields_Invariants(subject.Module_Fields, "reset.storage")
	for index := range subject.Counts {
		subject.Counts[index] = 0
	}
	for index := range subject.Buckets {
		subject.Buckets[index] = SYMBOL_ABSENT
	}
	subject.Causes.Failure = FAILURE_NONE
	subject.Flags.Failed = false
	subject.Names.Value = nil
	copy(subject.Words[:], UNIVERSE_WORDS)
	// The absent slot of each arena is taken the way every other slot is, thus one body states
	// what an arena slot holds and no slot stands outside it.
	add_symbol(subject, SYMBOL_UNKNOWN)
	add_member(subject)
	add_type(subject, TYPE_INVALID)
	subject.Packages[PACKAGE_UNIVERSE].Path = ""
	subject.Counts[COUNT_PACKAGE] = 1
	define_universe(subject)
}

// States every predeclared name. The basic types take the arena slot their kind names, thus a
// caller reads the type of a builtin word without a search.
func define_universe(subject Module_Handle) {
	Module_Handle_Invariants(subject, "define_universe.subject")
	kind := Basic_Kind(BASIC_KIND_MINIMUM)
	for uint8(kind) <= BASIC_KIND_MAXIMUM {
		define_basic(subject, kind)
		kind = kind + 1
	}
	define_universe_aliases(subject)
	define_universe_functions(subject)
	define_universe_values(subject)
}

// Takes one arena slot for a predeclared type and binds the word that names it. An untyped kind
// binds no word, because no source spells the type a literal folds to.
func define_basic(subject Module_Handle, kind Basic_Kind) {
	Module_Handle_Invariants(subject, "define_basic.subject")
	Basic_Kind_Invariants(kind, "define_basic.kind")
	index := add_type(subject, Type_Kind(kind))
	aver.Always(
		index == Type_Index(kind),
		"A predeclared type takes the arena slot its kind names.",
	)
	take_basic_name(subject, kind)
	if len(subject.Names.Value) == 0 {
		return
	}
	symbol := add_symbol(subject, SYMBOL_TYPE)
	subject.Symbols[symbol].Type = index
	bind_name(subject, symbol)
}

// Puts the word one predeclared type answers to in the name slot. An untyped kind answers to no
// word, because no source spells the type a literal folds to.
func take_basic_name(subject Module_Handle, kind Basic_Kind) {
	Module_Handle_Invariants(subject, "take_basic_name.subject")
	Basic_Kind_Invariants(kind, "take_basic_name.kind")
	subject.Names.Value = nil
	if uint8(kind) > uint8(TYPE_COMPLEX_128) {
		return
	}
	take_word(subject)
}

// Takes the next word of the universe text into the name slot. The words stand in one text the
// module copies into storage of its own, thus a predeclared name is a view the module owns and
// binding one copies nothing onto the heap.
func take_word(subject Module_Handle) {
	Module_Handle_Invariants(subject, "take_word.subject")
	start := int(subject.Counts[COUNT_WORD])
	stop := start
	for stop < UNIVERSE_WORD_SIZE {
		if subject.Words[stop] == ' ' {
			break
		}
		if subject.Words[stop] == 0 {
			break
		}
		stop = stop + 1
	}
	subject.Names.Value = Name(subject.Words[start:stop])
	subject.Counts[COUNT_WORD] = Count(stop + 1)
}

// Binds the words that name a type another word already names, and the two constraint words the
// language predeclares. A byte is a uint8 and a rune is an int32, thus both bind a slot that
// already stands.
func define_universe_aliases(subject Module_Handle) {
	Module_Handle_Invariants(subject, "define_universe_aliases.subject")
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(TYPE_UINT_8)
	bind_type_name(subject)
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(TYPE_INT_32)
	bind_type_name(subject)
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(add_type(subject, TYPE_CONSTRAINT))
	bind_type_name(subject)
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(add_type(subject, TYPE_CONSTRAINT))
	bind_type_name(subject)
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(add_type(subject, TYPE_CONSTRAINT))
	bind_type_name(subject)
}

// Binds the name slot to the type in the result slot.
func bind_type_name(subject Module_Handle) {
	Module_Handle_Invariants(subject, "bind_type_name.subject")
	symbol := add_symbol(subject, SYMBOL_TYPE)
	subject.Symbols[symbol].Type = Type_Index(subject.Counts[COUNT_RESULT])
	bind_name(subject, symbol)
}

// Binds every predeclared function. A builtin wears no signature, because its result follows the
// call it stands in and never the word alone.
func define_universe_functions(subject Module_Handle) {
	Module_Handle_Invariants(subject, "define_universe_functions.subject")
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
	take_word(subject)
	bind_builtin(subject)
}

// Binds the name slot as a predeclared function.
func bind_builtin(subject Module_Handle) {
	Module_Handle_Invariants(subject, "bind_builtin.subject")
	bind_name(subject, add_symbol(subject, SYMBOL_BUILTIN))
}

// Binds every predeclared value. Each one is a constant, thus a caller reads what it stands for
// from the type it wears.
func define_universe_values(subject Module_Handle) {
	Module_Handle_Invariants(subject, "define_universe_values.subject")
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_BOOLEAN)
	bind_value(subject)
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_BOOLEAN)
	bind_value(subject)
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_INTEGER)
	bind_value(subject)
	take_word(subject)
	subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_NIL)
	bind_value(subject)
}

// Binds the name slot as a predeclared constant of the type in the result slot.
func bind_value(subject Module_Handle) {
	Module_Handle_Invariants(subject, "bind_value.subject")
	symbol := add_symbol(subject, SYMBOL_CONSTANT)
	subject.Symbols[symbol].Type = Type_Index(subject.Counts[COUNT_RESULT])
	bind_name(subject, symbol)
}

// Takes one arena slot for a type of this kind.
func add_type(subject Module_Handle, kind Type_Kind) (index Type_Index) {
	defer func() { Type_Index_Invariants(index, "add_type.index") }()
	Module_Handle_Invariants(subject, "add_type.subject")
	Type_Kind_Invariants(kind, "add_type.kind")
	if int(subject.Counts[COUNT_TYPE]) >= TYPE_COUNT_MAXIMUM {
		fail(subject, Cause(FAILURE_TYPE_COUNT))
		return TYPE_INVALID_INDEX
	}
	index = Type_Index(subject.Counts[COUNT_TYPE])
	subject.Counts[COUNT_TYPE] = subject.Counts[COUNT_TYPE] + 1
	subject.Types[index] = Type{Element_Count: ELEMENT_COUNT_UNKNOWN}
	subject.Types[index].Kind = kind
	return index
}

// Takes one arena slot for a symbol of this kind and gives it the name in the name slot.
func add_symbol(subject Module_Handle, kind Symbol_Kind) (index Symbol_Index) {
	defer func() { Symbol_Index_Invariants(index, "add_symbol.index") }()
	Module_Handle_Invariants(subject, "add_symbol.subject")
	Symbol_Kind_Invariants(kind, "add_symbol.kind")
	if int(subject.Counts[COUNT_SYMBOL]) >= SYMBOL_COUNT_MAXIMUM {
		fail(subject, Cause(FAILURE_SYMBOL_COUNT))
		return SYMBOL_ABSENT
	}
	index = Symbol_Index(subject.Counts[COUNT_SYMBOL])
	subject.Counts[COUNT_SYMBOL] = subject.Counts[COUNT_SYMBOL] + 1
	subject.Symbols[index] = Symbol{}
	subject.Symbols[index].Kind = kind
	subject.Symbols[index].Name = subject.Names.Value
	subject.Symbols[index].File = File_Index(subject.Counts[COUNT_CURRENT_FILE])
	subject.Symbols[index].Owner = current_package(subject)
	return index
}

// Takes one arena slot for a member.
func add_member(subject Module_Handle) (index Member_Index) {
	defer func() { Member_Index_Invariants(index, "add_member.index") }()
	Module_Handle_Invariants(subject, "add_member.subject")
	if int(subject.Counts[COUNT_MEMBER]) >= MEMBER_COUNT_MAXIMUM {
		fail(subject, Cause(FAILURE_MEMBER_COUNT))
		return MEMBER_ABSENT
	}
	index = Member_Index(subject.Counts[COUNT_MEMBER])
	subject.Counts[COUNT_MEMBER] = subject.Counts[COUNT_MEMBER] + 1
	subject.Members[index] = Member{}
	return index
}

// Names the package the pass stands in, which is the universe until a file binds one.
func current_package(subject Module_Handle) (index Package_Index) {
	defer func() { Package_Index_Invariants(index, "current_package.index") }()
	Module_Handle_Invariants(subject, "current_package.subject")
	if subject.Counts[COUNT_FILE] == 0 {
		return PACKAGE_UNIVERSE
	}
	return subject.Files[subject.Counts[COUNT_CURRENT_FILE]].Owner
}

// Puts the name table slot of the name slot in the bucket cursor. The table is a power of two
// wide, thus the fold needs one mask and never a division.
func hash_name(subject Module_Handle) {
	Module_Handle_Invariants(subject, "hash_name.subject")
	folded := uint32(2166136261)
	name := subject.Names.Value
	for index := range len(name) {
		folded = folded ^ uint32(name[index])
		folded = folded * 16777619
	}
	subject.Counts[COUNT_BUCKET] = Count(folded % uint32(BUCKET_COUNT))
}

// Binds one symbol under the name it already wears, thus a later search finds it.
func bind_name(subject Module_Handle, symbol Symbol_Index) {
	Module_Handle_Invariants(subject, "bind_name.subject")
	Symbol_Index_Invariants(symbol, "bind_name.symbol")
	if symbol == SYMBOL_ABSENT {
		return
	}
	hash_name(subject)
	bucket := subject.Counts[COUNT_BUCKET]
	subject.Symbols[symbol].Next = Symbol_Successor(subject.Buckets[bucket])
	subject.Buckets[bucket] = symbol
}

// Names the symbol one package binds under the name slot, or no symbol.
func find_name(subject Module_Handle, owner Package_Index) (symbol Symbol_Index) {
	defer func() { Symbol_Index_Invariants(symbol, "find_name.symbol") }()
	Module_Handle_Invariants(subject, "find_name.subject")
	Package_Index_Invariants(owner, "find_name.owner")
	hash_name(subject)
	current := subject.Buckets[subject.Counts[COUNT_BUCKET]]
	for current != SYMBOL_ABSENT {
		matches := subject.Symbols[current].Owner == owner
		held := string(subject.Symbols[current].Name)
		if held != string(subject.Names.Value) {
			matches = false
		}
		if matches {
			return current
		}
		current = Symbol_Index(subject.Symbols[current].Next)
	}
	return SYMBOL_ABSENT
}

// Marks the module refused and records why. The first cause stands, because a later pass reads
// state the first refusal already left half built.
func fail(subject Module_Handle, cause Cause) {
	Module_Handle_Invariants(subject, "fail.subject")
	Cause_Invariants(cause, "fail.cause")
	if bool(subject.Flags.Failed) {
		return
	}
	subject.Flags.Failed = true
	subject.Causes.Failure = Failure_Code(cause)
}

// Reports whether the module already refused something.
func failed(subject Module_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "failed.yes") }()
	Module_Handle_Invariants(subject, "failed.subject")
	return subject.Flags.Failed
}

// Failure reads why the module refused, or FAILURE_NONE while it stands.
func Failure(subject Module_Handle) (code Failure_Code) {
	defer func() { Failure_Code_Invariants(code, "failure.code") }()
	Module_Handle_Invariants(subject, "failure.subject")
	return subject.Causes.Failure
}

// Failure_Message reads one code as an imperative sentence that says what to do about it.
func Failure_Message(code Failure_Code) (text Message) {
	defer func() { Message_Invariants(text, "failure_message.text") }()
	Failure_Code_Invariants(code, "failure_message.code")
	switch code {
	case FAILURE_NONE:
		return ""
	case FAILURE_FILE_COUNT:
		return "Analyse fewer files in one module."
	case FAILURE_PACKAGE_COUNT:
		return "Analyse fewer packages in one module."
	case FAILURE_SYMBOL_COUNT:
		return "Declare fewer names in one module."
	case FAILURE_TYPE_COUNT:
		return "Write fewer distinct types in one module."
	case FAILURE_MEMBER_COUNT:
		return "Write fewer fields and terms in one module."
	case FAILURE_NAME_SIZE:
		return "Write a shorter name."
	case FAILURE_PATH_SIZE:
		return "Write a shorter import path."
	case FAILURE_NESTING_DEPTH:
		return "Name the inner type and write the name."
	case FAILURE_UNKNOWN_NAME:
		return "Declare the name, or import the package that states it."
	case FAILURE_DUPLICATE_NAME:
		return "Rename one of the two declarations of this name."
	case FAILURE_UNKNOWN_IMPORT:
		return "Feed the package this import names to the module."
	}
	return "Break the cycle with a pointer, a slice, or a map."
}

// File_Result keeps file slot and insertion outcome at one output boundary.
type File_Result struct {
	// File remains zero when module has no capacity for another file.
	File File_Index
	// OK distinguishes first file slot from refused insertion.
	OK Boolean
}

// File_Result_Invariants composes one file insertion.
func File_Result_Invariants(value File_Result, namespace aver.Namespace) {
	File_Index_Invariants(value.File, namespace)
	Boolean_Invariants(value.OK, namespace)
}

// Add_File binds one file to the package its path names and to the source the caller owns. The
// caller keeps those bytes alive, because every name a symbol wears is a view of them.
func Add_File(subject Module_Handle, path Path, source token.Source) (result File_Result) {
	defer func() { File_Result_Invariants(result, "add_file.result") }()
	Module_Handle_Invariants(subject, "add_file.subject")
	Path_Invariants(path, "add_file.path")
	token.Source_Invariants(source, "add_file.source")
	if int(subject.Counts[COUNT_FILE]) >= FILE_COUNT_MAXIMUM {
		fail(subject, Cause(FAILURE_FILE_COUNT))
		return File_Result{File: 0, OK: false}
	}
	owner := package_of_path(subject, path)
	if !bool(owner.OK) {
		return File_Result{File: 0, OK: false}
	}
	file := File_Index(subject.Counts[COUNT_FILE])
	subject.Counts[COUNT_FILE] = subject.Counts[COUNT_FILE] + 1
	subject.Files[file] = File{Owner: owner.Package, First: 0, Count: 0}
	subject.Files[file].Source = source
	return File_Result{File: file, OK: true}
}

// Package_Result keeps package slot and insertion outcome at one output boundary.
type Package_Result struct {
	// Package remains universe where module has no capacity for another package.
	Package Package_Index
	// OK distinguishes universe path from refused insertion.
	OK Boolean
}

// Package_Result_Invariants composes one package lookup or insertion.
func Package_Result_Invariants(value Package_Result, namespace aver.Namespace) {
	Package_Index_Invariants(value.Package, namespace)
	Boolean_Invariants(value.OK, namespace)
}

// Names the package one path answers to, and takes a slot for a path no file named yet.
func package_of_path(subject Module_Handle, path Path) (result Package_Result) {
	defer func() { Package_Result_Invariants(result, "package_of_path.result") }()
	Module_Handle_Invariants(subject, "package_of_path.subject")
	Path_Invariants(path, "package_of_path.path")
	for slot := range int(subject.Counts[COUNT_PACKAGE]) {
		if subject.Packages[slot].Path == path {
			return Package_Result{Package: Package_Index(slot), OK: true}
		}
	}
	if int(subject.Counts[COUNT_PACKAGE]) >= PACKAGE_COUNT_MAXIMUM {
		fail(subject, Cause(FAILURE_PACKAGE_COUNT))
		return Package_Result{Package: PACKAGE_UNIVERSE, OK: false}
	}
	index := Package_Index(subject.Counts[COUNT_PACKAGE])
	subject.Counts[COUNT_PACKAGE] = subject.Counts[COUNT_PACKAGE] + 1
	subject.Packages[index].Path = path
	return Package_Result{Package: index, OK: true}
}

// Path_Of reads the import path one package answers to.
func Path_Of(subject Module_Handle, index Package_Index) (path Path) {
	defer func() { Path_Invariants(path, "path_of.path") }()
	Module_Handle_Invariants(subject, "path_of.subject")
	Package_Index_Invariants(index, "path_of.index")
	Package_Invariants(subject.Packages[index], "path_of.package")
	return subject.Packages[index].Path
}

// Package_Of names the package one file belongs to.
func Package_Of(subject Module_Handle, file File_Index) (index Package_Index) {
	defer func() { Package_Index_Invariants(index, "package_of.index") }()
	Module_Handle_Invariants(subject, "package_of.subject")
	File_Index_Invariants(file, "package_of.file")
	return subject.Files[file].Owner
}

// Source_Of reads the source view one file was bound to.
func Source_Of(subject Module_Handle, file File_Index) (source token.Source) {
	defer func() { token.Source_Invariants(source, "source_of.source") }()
	Module_Handle_Invariants(subject, "source_of.subject")
	File_Index_Invariants(file, "source_of.file")
	return subject.Files[file].Source
}

// Puts the walk on the file node of one tree. A tree the parser wrote holds one file node at its
// root even where the parse refused the source, thus a pass reads a partial tree the same way.
func open_walk(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "open_walk.subject")
	ast.Parse_State_Handle_Invariants(tree, "open_walk.tree")
	subject.Counts[COUNT_DEPTH] = 0
	subject.Nodes[0] = TREE_ROOT
	subject.Counts[COUNT_PARAMETER] = 0
	aver.Always(
		node_kind(subject, tree) == ast.NODE_FILE,
		"A tree a pass reads holds one file node at its root.",
	)
}

// Reads the class of the node the walk stands on.
func node_kind(subject Module_Handle, tree ast.Parse_State_Handle) (kind ast.Node_Kind) {
	defer func() { ast.Node_Kind_Invariants(kind, "node_kind.kind") }()
	Module_Handle_Invariants(subject, "node_kind.subject")
	ast.Parse_State_Handle_Invariants(tree, "node_kind.tree")
	return ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Kind
}

// Moves the walk to the first child of the node it stands on.
func descend(subject Module_Handle, tree ast.Parse_State_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "descend.ok") }()
	Module_Handle_Invariants(subject, "descend.subject")
	ast.Parse_State_Handle_Invariants(tree, "descend.tree")
	child := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).First_Child
	if ast.Index(child) == 0 {
		return false
	}
	if int(subject.Counts[COUNT_DEPTH])+1 >= DEPTH_MAXIMUM {
		fail(subject, Cause(FAILURE_NESTING_DEPTH))
		return false
	}
	subject.Counts[COUNT_DEPTH] = subject.Counts[COUNT_DEPTH] + 1
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = ast.Index(child)
	if bool(skip_trivia(subject, tree)) {
		return true
	}
	// A chain of trivia alone leaves the walk where it stood, thus a step that reads false
	// never owes a step back out.
	ascend(subject)
	return false
}

// Moves the walk to the sibling after the node it stands on.
func advance(subject Module_Handle, tree ast.Parse_State_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "advance.ok") }()
	Module_Handle_Invariants(subject, "advance.subject")
	ast.Parse_State_Handle_Invariants(tree, "advance.tree")
	next := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Next
	if ast.Index(next) == 0 {
		return false
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = ast.Index(next)
	return skip_trivia(subject, tree)
}

// Steps the walk past a comment and a run of empty lines, which stand in the tree where the
// author wrote them and name no declaration.
func skip_trivia(subject Module_Handle, tree ast.Parse_State_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "skip_trivia.ok") }()
	Module_Handle_Invariants(subject, "skip_trivia.subject")
	ast.Parse_State_Handle_Invariants(tree, "skip_trivia.tree")
	// The two trivia classes stand in the loop rather than in a body of their own, because a
	// body would owe every class the tree holds and a walk that skips trivia meets two.
	kind := node_kind(subject, tree)
	for kind == ast.NODE_BLANK || kind == ast.NODE_COMMENT {
		next := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Next
		if ast.Index(next) == 0 {
			return false
		}
		subject.Nodes[subject.Counts[COUNT_DEPTH]] = ast.Index(next)
		kind = node_kind(subject, tree)
	}
	return true
}

// Leaves the child chain the walk stands in.
func ascend(subject Module_Handle) {
	Module_Handle_Invariants(subject, "ascend.subject")
	if subject.Counts[COUNT_DEPTH] == 0 {
		return
	}
	subject.Counts[COUNT_DEPTH] = subject.Counts[COUNT_DEPTH] - 1
}

// Puts the source text of the node the walk stands on in the name slot.
func take_name(subject Module_Handle, tree ast.Parse_State_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "take_name.ok") }()
	Module_Handle_Invariants(subject, "take_name.subject")
	ast.Parse_State_Handle_Invariants(tree, "take_name.tree")
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	source := subject.Files[subject.Counts[COUNT_CURRENT_FILE]].Source
	text := token.Text(source, ast.Token_At(tree, node.Token))
	if len(text) > NAME_SIZE_MAXIMUM {
		fail(subject, Cause(FAILURE_NAME_SIZE))
		return false
	}
	subject.Names.Value = Name(text)
	return true
}

// Reports whether the token before the node the walk stands on opens another name, which is what
// tells a second name of one declaration from the type behind the first.
func opens_name(subject Module_Handle, tree ast.Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_name.yes") }()
	Module_Handle_Invariants(subject, "opens_name.subject")
	ast.Parse_State_Handle_Invariants(tree, "opens_name.tree")
	position := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	if position == 0 {
		return false
	}
	kind := ast.Token_At(tree, position-1).Kind
	if kind == token.KIND_COMMA {
		return true
	}
	if kind == token.KIND_CONSTANT {
		return true
	}
	return kind == token.KIND_VARIABLE
}

// Reports whether the token before the node the walk stands on opens the value of a declaration.
func opens_value(subject Module_Handle, tree ast.Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_value.yes") }()
	Module_Handle_Invariants(subject, "opens_value.subject")
	ast.Parse_State_Handle_Invariants(tree, "opens_value.tree")
	position := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	if position == 0 {
		return false
	}
	return ast.Token_At(tree, position-1).Kind == token.KIND_ASSIGN
}

// Declare records one symbol for each name a file states at package level, and one binding for
// each import. It reads no type, because a name a later file declares is a name this pass
// cannot know.
func Declare(subject Module_Handle, file File_Index, tree ast.Parse_State_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "declare.ok") }()
	Module_Handle_Invariants(subject, "declare.subject")
	File_Index_Invariants(file, "declare.file")
	ast.Parse_State_Handle_Invariants(tree, "declare.tree")
	subject.Counts[COUNT_CURRENT_FILE] = Count(file)
	subject.Files[file].First = Symbol_Head(subject.Counts[COUNT_SYMBOL])
	open_walk(subject, tree)
	open := descend(subject, tree)
	for bool(open) {
		declare_one(subject, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
	spent := subject.Counts[COUNT_SYMBOL] - Count(subject.Files[file].First)
	subject.Files[file].Count = Symbol_Count(spent)
	return !failed(subject)
}

// Records the symbols of the one declaration the walk stands on.
func declare_one(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "declare_one.subject")
	ast.Parse_State_Handle_Invariants(tree, "declare_one.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_IMPORT:
		declare_import(subject, tree)
	case ast.NODE_CONSTANT:
		declare_values(subject, tree, VALUE_CONSTANT)
	case ast.NODE_VARIABLE:
		declare_values(subject, tree, VALUE_VARIABLE)
	case ast.NODE_TYPE:
		declare_type(subject, tree)
	case ast.NODE_FUNCTION:
		declare_function(subject, tree)
	}
}

// Records one import binding. A binding wears the name the file states, or the segment after the
// final slash of the path where it states none.
func declare_import(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "declare_import.subject")
	ast.Parse_State_Handle_Invariants(tree, "declare_import.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	named := node_kind(subject, tree) == ast.NODE_IMPORT_NAME
	if !bool(take_name(subject, tree)) {
		ascend(subject)
		return
	}
	if !bool(named) {
		unquote(subject)
		take_tail(subject)
	}
	add_symbol(subject, SYMBOL_IMPORT)
	ascend(subject)
}

// Cuts the name slot back to the segment after its final slash, which is the name an import with
// no name of its own binds.
func take_tail(subject Module_Handle) {
	Module_Handle_Invariants(subject, "take_tail.subject")
	name := subject.Names.Value
	cut := 0
	for index := range len(name) {
		if name[index] == '/' {
			cut = index + 1
		}
	}
	subject.Names.Value = name[cut:]
}

// Takes the quotes off the name slot, which is how a path literal reads as a path.
func unquote(subject Module_Handle) {
	Module_Handle_Invariants(subject, "unquote.subject")
	name := subject.Names.Value
	if len(name) < 2 {
		return
	}
	if name[0] != '"' {
		return
	}
	subject.Names.Value = name[1 : len(name)-1]
}

// Records one symbol for each name a constant or a variable declaration states.
func declare_values(subject Module_Handle, tree ast.Parse_State_Handle, kind Value_Kind) {
	Module_Handle_Invariants(subject, "declare_values.subject")
	ast.Parse_State_Handle_Invariants(tree, "declare_values.tree")
	Value_Kind_Invariants(kind, "declare_values.kind")
	open := descend(subject, tree)
	named := Boolean(true)
	valued := Boolean(false)
	for bool(open) {
		valued = valued || opens_value(subject, tree)
		if bool(names_child(named, valued)) {
			bind_declared(subject, tree, Declared_Kind(kind))
		}
		open = advance(subject, tree)
		named = named && open
		named = named && opens_name(subject, tree)
	}
	ascend(subject)
}

// Reports whether one child of a value declaration binds a name.
func names_child(named Boolean, valued Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_child.yes") }()
	Boolean_Invariants(named, "names_child.named")
	Boolean_Invariants(valued, "names_child.valued")
	if bool(valued) {
		return false
	}
	return named
}

// Records one declared name in the current symbol slot and refuses a name its package already
// states.
func bind_declared(
	subject Module_Handle, tree ast.Parse_State_Handle, kind Declared_Kind,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "bind_declared.ok") }()
	Module_Handle_Invariants(subject, "bind_declared.subject")
	ast.Parse_State_Handle_Invariants(tree, "bind_declared.tree")
	Declared_Kind_Invariants(kind, "bind_declared.kind")
	subject.Counts[COUNT_CURRENT_SYMBOL] = Count(SYMBOL_ABSENT)
	if !bool(take_name(subject, tree)) {
		return false
	}
	if find_name(subject, current_package(subject)) != SYMBOL_ABSENT {
		fail(subject, Cause(FAILURE_DUPLICATE_NAME))
		return false
	}
	symbol := add_symbol(subject, Symbol_Kind(kind))
	bind_name(subject, symbol)
	subject.Counts[COUNT_CURRENT_SYMBOL] = Count(symbol)
	return symbol != SYMBOL_ABSENT
}

// Records one type declaration and takes the named type it states. The named type stands before
// its base type is known, thus a declaration that names a type a later file states resolves.
func declare_type(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "declare_type.subject")
	ast.Parse_State_Handle_Invariants(tree, "declare_type.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	if !bool(bind_declared(subject, tree, DECLARED_TYPE)) {
		ascend(subject)
		return
	}
	symbol := Symbol_Index(subject.Counts[COUNT_CURRENT_SYMBOL])
	index := add_type(subject, TYPE_NAMED)
	subject.Types[index].Symbol = Type_Symbol(symbol)
	subject.Symbols[symbol].Type = index
	ascend(subject)
}

// Records one function declaration. A declaration that carries a receiver names a method, and a
// method binds no package name, because two types may each state a method of one name.
func declare_function(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "declare_function.subject")
	ast.Parse_State_Handle_Invariants(tree, "declare_function.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	if node_kind(subject, tree) == ast.NODE_RECEIVER {
		declare_method(subject, tree)
		ascend(subject)
		return
	}
	bind_declared(subject, tree, DECLARED_FUNCTION)
	ascend(subject)
}

// Records the name of one method, which stands under its receiver and never under its package.
func declare_method(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "declare_method.subject")
	ast.Parse_State_Handle_Invariants(tree, "declare_method.tree")
	subject.Counts[COUNT_CURRENT_SYMBOL] = Count(SYMBOL_ABSENT)
	if !bool(advance(subject, tree)) {
		return
	}
	if !bool(take_name(subject, tree)) {
		return
	}
	subject.Counts[COUNT_CURRENT_SYMBOL] = Count(add_symbol(subject, SYMBOL_METHOD))
}

// Resolve folds the type expression of every symbol one file states. Every file declares before
// any file resolves, thus a type a later file states is a name this pass already knows.
func Resolve(subject Module_Handle, file File_Index, tree ast.Parse_State_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "resolve.ok") }()
	Module_Handle_Invariants(subject, "resolve.subject")
	File_Index_Invariants(file, "resolve.file")
	ast.Parse_State_Handle_Invariants(tree, "resolve.tree")
	subject.Counts[COUNT_CURRENT_FILE] = Count(file)
	subject.Counts[COUNT_RESOLVE] = Count(subject.Files[file].First)
	subject.Counts[COUNT_OWNER] = 0
	open_walk(subject, tree)
	open := descend(subject, tree)
	for bool(open) {
		resolve_one(subject, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
	return !failed(subject)
}

// Folds the one declaration the walk stands on.
func resolve_one(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_one.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_one.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_IMPORT:
		resolve_import(subject, tree)
	case ast.NODE_CONSTANT:
		resolve_values(subject, tree)
	case ast.NODE_VARIABLE:
		resolve_values(subject, tree)
	case ast.NODE_TYPE:
		resolve_type_declaration(subject, tree)
	case ast.NODE_FUNCTION:
		resolve_function(subject, tree)
	}
}

// Puts the symbol the resolve pass stands on in the current symbol slot and steps past it.
func take_symbol(subject Module_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "take_symbol.ok") }()
	Module_Handle_Invariants(subject, "take_symbol.subject")
	subject.Counts[COUNT_CURRENT_SYMBOL] = Count(SYMBOL_ABSENT)
	if subject.Counts[COUNT_RESOLVE] >= subject.Counts[COUNT_SYMBOL] {
		return false
	}
	subject.Counts[COUNT_CURRENT_SYMBOL] = subject.Counts[COUNT_RESOLVE]
	subject.Counts[COUNT_RESOLVE] = subject.Counts[COUNT_RESOLVE] + 1
	return true
}

// Binds one import to the package its path names.
func resolve_import(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_import.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_import.tree")
	if !bool(take_symbol(subject)) {
		return
	}
	symbol := Symbol_Index(subject.Counts[COUNT_CURRENT_SYMBOL])
	if !bool(descend(subject, tree)) {
		return
	}
	if node_kind(subject, tree) == ast.NODE_IMPORT_NAME {
		advance(subject, tree)
	}
	if !bool(take_name(subject, tree)) {
		ascend(subject)
		return
	}
	unquote(subject)
	path := subject.Names.Value
	for slot := range int(subject.Counts[COUNT_PACKAGE]) {
		if string(subject.Packages[slot].Path) == string(path) {
			subject.Symbols[symbol].Target = Target_Package(slot)
			slot = int(subject.Counts[COUNT_PACKAGE])
		}
	}
	ascend(subject)
}

// Folds the type a constant or a variable declaration states and gives it to every name the
// declaration binds. A declaration that states no type wears the invalid type, because the value
// that names it is an expression this pass does not fold.
func resolve_values(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_values.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_values.tree")
	open := descend(subject, tree)
	names := 0
	folded := TYPE_INVALID_INDEX
	named := Boolean(true)
	valued := Boolean(false)
	for bool(open) {
		valued = valued || opens_value(subject, tree)
		if bool(names_child(named, valued)) {
			names = names + 1
		}
		if bool(types_child(named, valued)) {
			resolve_expression(subject, tree)
			folded = Type_Index(subject.Counts[COUNT_RESULT])
		}
		open = advance(subject, tree)
		named = named && open
		named = named && opens_name(subject, tree)
	}
	ascend(subject)
	for range names {
		if bool(take_symbol(subject)) {
			subject.Symbols[subject.Counts[COUNT_CURRENT_SYMBOL]].Type = folded
		}
	}
}

// Reports whether one child of a value declaration states the type the names wear.
func types_child(named Boolean, valued Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "types_child.yes") }()
	Boolean_Invariants(named, "types_child.named")
	Boolean_Invariants(valued, "types_child.valued")
	if bool(valued) {
		return false
	}
	return !named
}

// Folds the base type one type declaration states.
func resolve_type_declaration(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_type_declaration.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_type_declaration.tree")
	if !bool(take_symbol(subject)) {
		return
	}
	named := subject.Symbols[subject.Counts[COUNT_CURRENT_SYMBOL]].Type
	if !bool(descend(subject, tree)) {
		return
	}
	subject.Counts[COUNT_PARAMETER] = 0
	open := advance(subject, tree)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_TYPE_PARAMETER {
			resolve_expression(subject, tree)
			subject.Types[named].Element = Type_Element(subject.Counts[COUNT_RESULT])
			open = false
			continue
		}
		declare_parameter(subject, tree)
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_PARAMETER] = 0
	ascend(subject)
	subject.Counts[COUNT_RESULT] = Count(named)
	guard_cycle(subject)
}

// Refuses a named type that stands on itself. Such a type has no size, thus no caller can read
// what it holds.
func guard_cycle(subject Module_Handle) {
	Module_Handle_Invariants(subject, "guard_cycle.subject")
	index := Type_Index(subject.Counts[COUNT_RESULT])
	current := index
	for range DEPTH_MAXIMUM {
		if subject.Types[current].Kind != TYPE_NAMED {
			return
		}
		current = Type_Index(subject.Types[current].Element)
		if current == TYPE_INVALID_INDEX {
			return
		}
		if current == index {
			fail(subject, Cause(FAILURE_TYPE_CYCLE))
			return
		}
	}
	fail(subject, Cause(FAILURE_TYPE_CYCLE))
}

// Records one type parameter of a declaration and folds the constraint it states.
func declare_parameter(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "declare_parameter.subject")
	ast.Parse_State_Handle_Invariants(tree, "declare_parameter.tree")
	if int(subject.Counts[COUNT_PARAMETER]) >= TYPE_PARAMETER_MAXIMUM {
		return
	}
	if !bool(descend(subject, tree)) {
		return
	}
	if !bool(take_name(subject, tree)) {
		ascend(subject)
		return
	}
	symbol := add_symbol(subject, SYMBOL_TYPE_PARAMETER)
	index := add_type(subject, TYPE_PARAMETER)
	subject.Types[index].Symbol = Type_Symbol(symbol)
	subject.Symbols[symbol].Type = index
	subject.Parameters[subject.Counts[COUNT_PARAMETER]] = symbol
	subject.Counts[COUNT_PARAMETER] = subject.Counts[COUNT_PARAMETER] + 1
	if bool(advance(subject, tree)) {
		resolve_expression(subject, tree)
		subject.Types[index].Element = Type_Element(subject.Counts[COUNT_RESULT])
	}
	ascend(subject)
}

// Folds the signature one function declaration states.
func resolve_function(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_function.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_function.tree")
	if !bool(take_symbol(subject)) {
		return
	}
	symbol := Symbol_Index(subject.Counts[COUNT_CURRENT_SYMBOL])
	if !bool(descend(subject, tree)) {
		return
	}
	subject.Counts[COUNT_PARAMETER] = 0
	subject.Counts[COUNT_RECEIVER] = Count(TYPE_INVALID_INDEX)
	index := add_type(subject, TYPE_FUNCTION)
	subject.Types[index].Symbol = Type_Symbol(symbol)
	subject.Symbols[symbol].Type = index
	if node_kind(subject, tree) == ast.NODE_RECEIVER {
		resolve_receiver(subject, tree)
		advance(subject, tree)
	}
	subject.Counts[COUNT_RESULT] = Count(index)
	push_owner(subject)
	open := advance(subject, tree)
	for bool(open) {
		open = resolve_signature_part(subject, tree)
	}
	pop_owner(subject)
	subject.Counts[COUNT_PARAMETER] = 0
	ascend(subject)
	attach_method(subject)
}

// Folds the type the receiver of a method states and puts the named type it stands for in the
// receiver slot. A pointer receiver names the type it points at, thus both forms reach one type.
func resolve_receiver(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_receiver.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_receiver.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	if !bool(descend(subject, tree)) {
		ascend(subject)
		return
	}
	if node_kind(subject, tree) == ast.NODE_PARAMETER_NAME {
		advance(subject, tree)
	}
	resolve_expression(subject, tree)
	ascend(subject)
	ascend(subject)
	folded := Type_Index(subject.Counts[COUNT_RESULT])
	if subject.Types[folded].Kind == TYPE_POINTER {
		folded = Type_Index(subject.Types[folded].Element)
	}
	subject.Counts[COUNT_RECEIVER] = Count(folded)
}

// Attaches one method to the named type its receiver states, thus a selector reads a method the
// way it reads a field.
func attach_method(subject Module_Handle) {
	Module_Handle_Invariants(subject, "attach_method.subject")
	receiver := Type_Index(subject.Counts[COUNT_RECEIVER])
	if receiver == TYPE_INVALID_INDEX {
		return
	}
	symbol := Symbol_Index(subject.Counts[COUNT_CURRENT_SYMBOL])
	member := add_member(subject)
	subject.Members[member].Symbol = Member_Symbol(symbol)
	subject.Members[member].Type = Member_Type(subject.Symbols[symbol].Type)
	subject.Counts[COUNT_RESULT] = Count(receiver)
	push_owner(subject)
	subject.Counts[COUNT_CURRENT_MEMBER] = Count(member)
	link_member(subject)
	pop_owner(subject)
	subject.Counts[COUNT_RECEIVER] = Count(TYPE_INVALID_INDEX)
}

// Puts the folded type on the stack of types the members that follow attach to.
func push_owner(subject Module_Handle) {
	Module_Handle_Invariants(subject, "push_owner.subject")
	if int(subject.Counts[COUNT_OWNER]) >= DEPTH_MAXIMUM {
		fail(subject, Cause(FAILURE_NESTING_DEPTH))
		return
	}
	subject.Owners[subject.Counts[COUNT_OWNER]] = Type_Index(subject.Counts[COUNT_RESULT])
	subject.Counts[COUNT_OWNER] = subject.Counts[COUNT_OWNER] + 1
}

// Takes the type the members that follow attach to off the stack.
func pop_owner(subject Module_Handle) {
	Module_Handle_Invariants(subject, "pop_owner.subject")
	if subject.Counts[COUNT_OWNER] == 0 {
		return
	}
	subject.Counts[COUNT_OWNER] = subject.Counts[COUNT_OWNER] - 1
}

// Folds one part of a signature and reports whether another part stands behind it.
func resolve_signature_part(subject Module_Handle, tree ast.Parse_State_Handle) (open Boolean) {
	defer func() { Boolean_Invariants(open, "resolve_signature_part.open") }()
	Module_Handle_Invariants(subject, "resolve_signature_part.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_signature_part.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_TYPE_PARAMETER {
		declare_parameter(subject, tree)
		return advance(subject, tree)
	}
	if kind == ast.NODE_PARAMETER {
		attach_member(subject, tree, false)
		return advance(subject, tree)
	}
	if kind == ast.NODE_RESULT {
		attach_member(subject, tree, true)
		return advance(subject, tree)
	}
	return false
}

// Adds one parameter or one result to the tuple of the signature on the owner stack.
func attach_member(subject Module_Handle, tree ast.Parse_State_Handle, result Boolean) {
	Module_Handle_Invariants(subject, "attach_member.subject")
	ast.Parse_State_Handle_Invariants(tree, "attach_member.tree")
	Boolean_Invariants(result, "attach_member.result")
	take_tuple(subject, result)
	push_owner(subject)
	if !bool(descend(subject, tree)) {
		pop_owner(subject)
		return
	}
	kind := SYMBOL_PARAMETER
	if bool(result) {
		kind = SYMBOL_RESULT
	}
	symbol := SYMBOL_ABSENT
	if node_kind(subject, tree) == ast.NODE_PARAMETER_NAME {
		if bool(take_name(subject, tree)) {
			symbol = add_symbol(subject, kind)
		}
		advance(subject, tree)
	}
	resolve_expression(subject, tree)
	member := add_member(subject)
	subject.Members[member].Symbol = Member_Symbol(symbol)
	subject.Members[member].Type = Member_Type(subject.Counts[COUNT_RESULT])
	subject.Symbols[symbol].Type = Type_Index(subject.Counts[COUNT_RESULT])
	subject.Counts[COUNT_CURRENT_MEMBER] = Count(member)
	link_member(subject)
	ascend(subject)
	pop_owner(subject)
}

// Puts the tuple the signature on the owner stack holds its parameters or its results in into the
// result slot, and takes one where the signature has none yet.
func take_tuple(subject Module_Handle, result Boolean) {
	Module_Handle_Invariants(subject, "take_tuple.subject")
	Boolean_Invariants(result, "take_tuple.result")
	signature := subject.Owners[subject.Counts[COUNT_OWNER]-1]
	open := Type_Index(subject.Types[signature].Key)
	if bool(result) {
		open = Type_Index(subject.Types[signature].Element)
	}
	if open != TYPE_INVALID_INDEX {
		subject.Counts[COUNT_RESULT] = Count(open)
		return
	}
	open = add_type(subject, TYPE_TUPLE)
	subject.Counts[COUNT_RESULT] = Count(open)
	if bool(result) {
		subject.Types[signature].Element = Type_Element(open)
		return
	}
	subject.Types[signature].Key = Type_Key(open)
}

// Puts the member in the current member slot at the end of the chain the type on the owner stack
// holds.
func link_member(subject Module_Handle) {
	Module_Handle_Invariants(subject, "link_member.subject")
	member := Member_Index(subject.Counts[COUNT_CURRENT_MEMBER])
	if member == MEMBER_ABSENT {
		return
	}
	index := subject.Owners[subject.Counts[COUNT_OWNER]-1]
	current := Member_Index(subject.Types[index].Members)
	if current == MEMBER_ABSENT {
		subject.Types[index].Members = Member_Head(member)
		return
	}
	for Member_Index(subject.Members[current].Next) != MEMBER_ABSENT {
		current = Member_Index(subject.Members[current].Next)
	}
	subject.Members[current].Next = Member_Successor(member)
}

// WRAPPER_POINTER names the pointer wrapper.
const WRAPPER_POINTER Wrapper_Kind = Wrapper_Kind(TYPE_POINTER)

// WRAPPER_SLICE names the slice wrapper.
const WRAPPER_SLICE Wrapper_Kind = Wrapper_Kind(TYPE_SLICE)

// WRAPPER_CHANNEL names the channel wrapper.
const WRAPPER_CHANNEL Wrapper_Kind = Wrapper_Kind(TYPE_CHANNEL)

// Wrapper_Kind names a type that stands over one element alone. It wears a type of its own,
// because the body that folds such a type meets these three kinds and no other.
type Wrapper_Kind uint8

// Wrapper_Kind_Invariants states every kind that stands over one element alone.
func Wrapper_Kind_Invariants(value Wrapper_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), uint8(WRAPPER_POINTER), uint8(WRAPPER_SLICE),
			uint8(WRAPPER_CHANNEL)).
		Ensure()
}

// Folds the type expression the walk stands on and puts it in the result slot.
func resolve_expression(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_expression.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_expression.tree")
	subject.Counts[COUNT_RESULT] = Count(TYPE_INVALID_INDEX)
	switch node_kind(subject, tree) {
	case ast.NODE_IDENTIFIER:
		resolve_name(subject, tree)
	case ast.NODE_SELECTOR:
		resolve_qualified(subject, tree)
	case ast.NODE_POINTER_TYPE:
		resolve_wrapper(subject, tree, WRAPPER_POINTER)
	case ast.NODE_SLICE_TYPE, ast.NODE_ELLIPSIS:
		resolve_wrapper(subject, tree, WRAPPER_SLICE)
	case ast.NODE_CHANNEL_TYPE, ast.NODE_CHANNEL_SEND, ast.NODE_CHANNEL_RECEIVE:
		resolve_wrapper(subject, tree, WRAPPER_CHANNEL)
	case ast.NODE_ARRAY_TYPE:
		resolve_array(subject, tree)
	case ast.NODE_MAP_TYPE:
		resolve_map(subject, tree)
	case ast.NODE_FUNCTION_TYPE:
		resolve_signature(subject, tree)
	case ast.NODE_STRUCTURE_TYPE:
		resolve_structure(subject, tree)
	case ast.NODE_INTERFACE_TYPE:
		resolve_constraint(subject, tree)
	case ast.NODE_GENERIC:
		resolve_instance(subject, tree)
	case ast.NODE_PARENTHESIS:
		resolve_inner(subject, tree)
	}
}

// Puts the type one identifier stands for in the result slot. A type parameter of the declaration
// stands ahead of the package, because it names a type only inside the declaration that states it.
func resolve_name(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_name.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_name.tree")
	if !bool(take_name(subject, tree)) {
		return
	}
	for slot := range int(subject.Counts[COUNT_PARAMETER]) {
		parameter := subject.Parameters[slot]
		held := string(subject.Symbols[parameter].Name)
		if held == string(subject.Names.Value) {
			subject.Counts[COUNT_RESULT] = Count(subject.Symbols[parameter].Type)
			return
		}
	}
	symbol := find_name(subject, current_package(subject))
	if symbol == SYMBOL_ABSENT {
		symbol = find_name(subject, PACKAGE_UNIVERSE)
	}
	if symbol == SYMBOL_ABSENT {
		fail(subject, Cause(FAILURE_UNKNOWN_NAME))
		return
	}
	subject.Counts[COUNT_RESULT] = Count(subject.Symbols[symbol].Type)
}

// Puts the type one qualified identifier stands for in the result slot. The qualifier names an
// import of this file, thus a name of another package reads the way a caller wrote it.
func resolve_qualified(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_qualified.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_qualified.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	if !bool(resolve_qualifier(subject, tree)) {
		ascend(subject)
		return
	}
	target := Package_Index(subject.Counts[COUNT_TARGET])
	if !bool(advance(subject, tree)) {
		ascend(subject)
		return
	}
	if !bool(take_name(subject, tree)) {
		ascend(subject)
		return
	}
	symbol := find_name(subject, target)
	ascend(subject)
	if symbol == SYMBOL_ABSENT {
		fail(subject, Cause(FAILURE_UNKNOWN_NAME))
		return
	}
	subject.Counts[COUNT_RESULT] = Count(subject.Symbols[symbol].Type)
}

// Puts the package the qualifier of a selector stands for in the target slot.
func resolve_qualifier(subject Module_Handle, tree ast.Parse_State_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "resolve_qualifier.ok") }()
	Module_Handle_Invariants(subject, "resolve_qualifier.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_qualifier.tree")
	subject.Counts[COUNT_TARGET] = Count(PACKAGE_UNIVERSE)
	if !bool(take_name(subject, tree)) {
		return false
	}
	file := subject.Files[subject.Counts[COUNT_CURRENT_FILE]]
	for slot := range int(file.Count) {
		symbol := Symbol_Index(int(file.First) + slot)
		if subject.Symbols[symbol].Kind != SYMBOL_IMPORT {
			continue
		}
		held := string(subject.Symbols[symbol].Name)
		if held != string(subject.Names.Value) {
			continue
		}
		if subject.Symbols[symbol].Target == Target_Package(PACKAGE_UNIVERSE) {
			fail(subject, Cause(FAILURE_UNKNOWN_IMPORT))
			return false
		}
		subject.Counts[COUNT_TARGET] = Count(subject.Symbols[symbol].Target)
		return true
	}
	fail(subject, Cause(FAILURE_UNKNOWN_NAME))
	return false
}

// Folds a type that stands over one element: a pointer, a slice, or a channel of any direction.
func resolve_wrapper(subject Module_Handle, tree ast.Parse_State_Handle, kind Wrapper_Kind) {
	Module_Handle_Invariants(subject, "resolve_wrapper.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_wrapper.tree")
	Wrapper_Kind_Invariants(kind, "resolve_wrapper.kind")
	index := add_type(subject, Type_Kind(kind))
	subject.Counts[COUNT_RESULT] = Count(index)
	if !bool(descend(subject, tree)) {
		return
	}
	resolve_expression(subject, tree)
	subject.Types[index].Element = Type_Element(subject.Counts[COUNT_RESULT])
	ascend(subject)
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Folds an array type and the count one literal states. A count no literal states stays unknown,
// because the constant pass this module does not run is what folds an expression.
func resolve_array(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_array.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_array.tree")
	index := add_type(subject, TYPE_ARRAY)
	subject.Counts[COUNT_RESULT] = Count(index)
	if !bool(descend(subject, tree)) {
		return
	}
	subject.Types[index].Element_Count = take_element_count(subject, tree)
	if !bool(advance(subject, tree)) {
		ascend(subject)
		subject.Counts[COUNT_RESULT] = Count(index)
		return
	}
	resolve_expression(subject, tree)
	subject.Types[index].Element = Type_Element(subject.Counts[COUNT_RESULT])
	ascend(subject)
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Reads the element_count one array literal states.
func take_element_count(
	subject Module_Handle, tree ast.Parse_State_Handle,
) (element_count Element_Count) {
	defer func() { Element_Count_Invariants(element_count, "take_element_count.count") }()
	Module_Handle_Invariants(subject, "take_element_count.subject")
	ast.Parse_State_Handle_Invariants(tree, "take_element_count.tree")
	if node_kind(subject, tree) != ast.NODE_INTEGER {
		return ELEMENT_COUNT_UNKNOWN
	}
	if !bool(take_name(subject, tree)) {
		return ELEMENT_COUNT_UNKNOWN
	}
	folded := constant.Int_64_Value(
		constant.Make_From_Literal(constant.Text(subject.Names.Value)))
	if !bool(folded.OK) {
		return ELEMENT_COUNT_UNKNOWN
	}
	if int64(folded.Value) > ELEMENT_COUNT_MAXIMUM {
		return ELEMENT_COUNT_UNKNOWN
	}
	if int64(folded.Value) < 0 {
		return ELEMENT_COUNT_UNKNOWN
	}
	return Element_Count(folded.Value)
}

// Folds a map type, which states a key and a value.
func resolve_map(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_map.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_map.tree")
	index := add_type(subject, TYPE_MAP)
	subject.Counts[COUNT_RESULT] = Count(index)
	if !bool(descend(subject, tree)) {
		return
	}
	resolve_expression(subject, tree)
	subject.Types[index].Key = Type_Key(subject.Counts[COUNT_RESULT])
	if bool(advance(subject, tree)) {
		resolve_expression(subject, tree)
		subject.Types[index].Element = Type_Element(subject.Counts[COUNT_RESULT])
	}
	ascend(subject)
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Folds a signature that stands as a type rather than as a declaration.
func resolve_signature(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_signature.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_signature.tree")
	index := add_type(subject, TYPE_FUNCTION)
	subject.Counts[COUNT_RESULT] = Count(index)
	push_owner(subject)
	open := descend(subject, tree)
	for bool(open) {
		open = resolve_signature_part(subject, tree)
	}
	ascend(subject)
	pop_owner(subject)
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Folds a struct type and the fields it states. An embedded type names the field it embeds, thus
// a field with no name of its own still stands in the chain.
func resolve_structure(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_structure.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_structure.tree")
	index := add_type(subject, TYPE_STRUCTURE)
	subject.Counts[COUNT_RESULT] = Count(index)
	push_owner(subject)
	open := descend(subject, tree)
	for bool(open) {
		if node_kind(subject, tree) == ast.NODE_FIELD {
			attach_field(subject, tree)
		}
		open = advance(subject, tree)
	}
	ascend(subject)
	pop_owner(subject)
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Adds one field to the chain the struct on the owner stack holds.
func attach_field(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "attach_field.subject")
	ast.Parse_State_Handle_Invariants(tree, "attach_field.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	symbol := SYMBOL_ABSENT
	if node_kind(subject, tree) == ast.NODE_FIELD_NAME {
		if bool(take_name(subject, tree)) {
			symbol = add_symbol(subject, SYMBOL_FIELD)
		}
		advance(subject, tree)
	}
	resolve_expression(subject, tree)
	member := add_member(subject)
	subject.Members[member].Symbol = Member_Symbol(symbol)
	subject.Members[member].Type = Member_Type(subject.Counts[COUNT_RESULT])
	subject.Symbols[symbol].Type = Type_Index(subject.Counts[COUNT_RESULT])
	subject.Counts[COUNT_CURRENT_MEMBER] = Count(member)
	link_member(subject)
	ascend(subject)
}

// Folds a constraint interface, which is the one interface this grammar admits.
func resolve_constraint(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_constraint.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_constraint.tree")
	index := add_type(subject, TYPE_CONSTRAINT)
	subject.Counts[COUNT_RESULT] = Count(index)
	push_owner(subject)
	open := descend(subject, tree)
	for bool(open) {
		attach_term(subject, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
	pop_owner(subject)
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Adds the terms one element of a constraint states. A union states each term of its own, thus
// one element of the source may add several terms.
func attach_term(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "attach_term.subject")
	ast.Parse_State_Handle_Invariants(tree, "attach_term.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_UNION {
		open := descend(subject, tree)
		for bool(open) {
			attach_term(subject, tree)
			open = advance(subject, tree)
		}
		ascend(subject)
		return
	}
	if kind == ast.NODE_TERM {
		if !bool(descend(subject, tree)) {
			return
		}
		attach_term(subject, tree)
		ascend(subject)
		return
	}
	resolve_expression(subject, tree)
	member := add_member(subject)
	subject.Members[member].Type = Member_Type(subject.Counts[COUNT_RESULT])
	subject.Counts[COUNT_CURRENT_MEMBER] = Count(member)
	link_member(subject)
}

// Folds a generic instance, which wears the name of its base and the arguments it states.
func resolve_instance(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_instance.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_instance.tree")
	index := add_type(subject, TYPE_NAMED)
	subject.Counts[COUNT_RESULT] = Count(index)
	push_owner(subject)
	if !bool(descend(subject, tree)) {
		pop_owner(subject)
		return
	}
	resolve_expression(subject, tree)
	base := Type_Index(subject.Counts[COUNT_RESULT])
	subject.Types[index].Symbol = subject.Types[base].Symbol
	subject.Types[index].Element = subject.Types[base].Element
	open := advance(subject, tree)
	for bool(open) {
		resolve_expression(subject, tree)
		member := add_member(subject)
		subject.Members[member].Type = Member_Type(subject.Counts[COUNT_RESULT])
		subject.Counts[COUNT_CURRENT_MEMBER] = Count(member)
		link_member(subject)
		open = advance(subject, tree)
	}
	ascend(subject)
	pop_owner(subject)
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Folds the type one grouping states, which is the type inside it.
func resolve_inner(subject Module_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "resolve_inner.subject")
	ast.Parse_State_Handle_Invariants(tree, "resolve_inner.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	resolve_expression(subject, tree)
	ascend(subject)
}

// Lookup_Result keeps symbol slot and presence at one output boundary.
type Lookup_Result struct {
	// Symbol remains absent when package binds no matching name.
	Symbol Symbol_Index
	// Found distinguishes absent sentinel from present symbol.
	Found Boolean
}

// Lookup_Result_Invariants composes one symbol lookup.
func Lookup_Result_Invariants(value Lookup_Result, namespace aver.Namespace) {
	Symbol_Index_Invariants(value.Symbol, namespace)
	Boolean_Invariants(value.Found, namespace)
}

// Lookup names the symbol one package binds under a name. The universe holds the predeclared
// names, thus a caller reads int and len from PACKAGE_UNIVERSE.
func Lookup(
	subject Module_Handle, owner Package_Index, name Name,
) (result Lookup_Result) {
	defer func() { Lookup_Result_Invariants(result, "lookup.result") }()
	Module_Handle_Invariants(subject, "lookup.subject")
	Package_Index_Invariants(owner, "lookup.owner")
	Name_Invariants(name, "lookup.name")
	subject.Names.Value = name
	symbol := find_name(subject, owner)
	return Lookup_Result{Symbol: symbol, Found: Boolean(symbol != SYMBOL_ABSENT)}
}

// Symbol_Kind_Of names what one symbol declares.
func Symbol_Kind_Of(subject Module_Handle, symbol Symbol_Index) (kind Symbol_Kind) {
	defer func() { Symbol_Kind_Invariants(kind, "symbol_kind_of.kind") }()
	Module_Handle_Invariants(subject, "symbol_kind_of.subject")
	Symbol_Index_Invariants(symbol, "symbol_kind_of.symbol")
	return subject.Symbols[symbol].Kind
}

// Symbol_Name_Of reads the identifier one symbol wears.
func Symbol_Name_Of(subject Module_Handle, symbol Symbol_Index) (name Name) {
	defer func() { Name_Invariants(name, "symbol_name_of.name") }()
	Module_Handle_Invariants(subject, "symbol_name_of.subject")
	Symbol_Index_Invariants(symbol, "symbol_name_of.symbol")
	return subject.Symbols[symbol].Name
}

// Symbol_File_Of names the file that states one symbol.
func Symbol_File_Of(subject Module_Handle, symbol Symbol_Index) (file File_Index) {
	defer func() { File_Index_Invariants(file, "symbol_file_of.file") }()
	Module_Handle_Invariants(subject, "symbol_file_of.subject")
	Symbol_Index_Invariants(symbol, "symbol_file_of.symbol")
	return subject.Symbols[symbol].File
}

// Symbol_Type_Of names the type one symbol wears.
func Symbol_Type_Of(subject Module_Handle, symbol Symbol_Index) (index Type_Index) {
	defer func() { Type_Index_Invariants(index, "symbol_type_of.index") }()
	Module_Handle_Invariants(subject, "symbol_type_of.subject")
	Symbol_Index_Invariants(symbol, "symbol_type_of.symbol")
	return subject.Symbols[symbol].Type
}

// Symbol_Target_Of names the package one import binding stands for.
func Symbol_Target_Of(subject Module_Handle, symbol Symbol_Index) (target Package_Index) {
	defer func() { Package_Index_Invariants(target, "symbol_target_of.target") }()
	Module_Handle_Invariants(subject, "symbol_target_of.subject")
	Symbol_Index_Invariants(symbol, "symbol_target_of.symbol")
	return Package_Index(subject.Symbols[symbol].Target)
}

// Type_Kind_Of names what one type is.
func Type_Kind_Of(subject Module_Handle, index Type_Index) (kind Type_Kind) {
	defer func() { Type_Kind_Invariants(kind, "type_kind_of.kind") }()
	Module_Handle_Invariants(subject, "type_kind_of.subject")
	Type_Index_Invariants(index, "type_kind_of.index")
	return subject.Types[index].Kind
}

// Type_Element_Of names the type one type stands over.
func Type_Element_Of(subject Module_Handle, index Type_Index) (element Type_Element) {
	defer func() { Type_Element_Invariants(element, "type_element_of.element") }()
	Module_Handle_Invariants(subject, "type_element_of.subject")
	Type_Index_Invariants(index, "type_element_of.index")
	return subject.Types[index].Element
}

// Type_Key_Of names the key of a map or the parameters of a signature.
func Type_Key_Of(subject Module_Handle, index Type_Index) (key Type_Key) {
	defer func() { Type_Key_Invariants(key, "type_key_of.key") }()
	Module_Handle_Invariants(subject, "type_key_of.subject")
	Type_Index_Invariants(index, "type_key_of.index")
	return subject.Types[index].Key
}

// Type_Element_Count_Of reads the element count one array type holds.
func Type_Element_Count_Of(subject Module_Handle, index Type_Index) (element_count Element_Count) {
	defer func() { Element_Count_Invariants(element_count, "type_element_count_of.count") }()
	Module_Handle_Invariants(subject, "type_element_count_of.subject")
	Type_Index_Invariants(index, "type_element_count_of.index")
	return subject.Types[index].Element_Count
}

// Type_Symbol_Of names the declaration one named type stands for.
func Type_Symbol_Of(subject Module_Handle, index Type_Index) (symbol Type_Symbol) {
	defer func() { Type_Symbol_Invariants(symbol, "type_symbol_of.symbol") }()
	Module_Handle_Invariants(subject, "type_symbol_of.subject")
	Type_Index_Invariants(index, "type_symbol_of.index")
	return subject.Types[index].Symbol
}

// Type_Members_Of names the first field, term, or tuple member one type holds.
func Type_Members_Of(subject Module_Handle, index Type_Index) (member Member_Head) {
	defer func() { Member_Head_Invariants(member, "type_members_of.member") }()
	Module_Handle_Invariants(subject, "type_members_of.subject")
	Type_Index_Invariants(index, "type_members_of.index")
	return subject.Types[index].Members
}

// Member_Type_Of names the type one member wears.
func Member_Type_Of(subject Module_Handle, member Member_Index) (index Member_Type) {
	defer func() { Member_Type_Invariants(index, "member_type_of.index") }()
	Module_Handle_Invariants(subject, "member_type_of.subject")
	Member_Index_Invariants(member, "member_type_of.member")
	return subject.Members[member].Type
}

// Member_Symbol_Of names the symbol one member wears, or no symbol where it wears none.
func Member_Symbol_Of(subject Module_Handle, member Member_Index) (symbol Member_Symbol) {
	defer func() { Member_Symbol_Invariants(symbol, "member_symbol_of.symbol") }()
	Module_Handle_Invariants(subject, "member_symbol_of.subject")
	Member_Index_Invariants(member, "member_symbol_of.member")
	return subject.Members[member].Symbol
}

// Member_Next_Of names the member after one member.
func Member_Next_Of(subject Module_Handle, member Member_Index) (next Member_Successor) {
	defer func() { Member_Successor_Invariants(next, "member_next_of.next") }()
	Module_Handle_Invariants(subject, "member_next_of.subject")
	Member_Index_Invariants(member, "member_next_of.member")
	return subject.Members[member].Next
}

// Underlying_Of walks a named type down to the type its declaration states. A type that names
// nothing is its own base type, thus a caller walks once and reads what it holds.
func Underlying_Of(subject Module_Handle, index Type_Index) (base Type_Index) {
	defer func() { Type_Index_Invariants(base, "underlying_of.base") }()
	Module_Handle_Invariants(subject, "underlying_of.subject")
	Type_Index_Invariants(index, "underlying_of.index")
	base = index
	for range DEPTH_MAXIMUM {
		if subject.Types[base].Kind != TYPE_NAMED {
			return base
		}
		element := Type_Index(subject.Types[base].Element)
		if element == TYPE_INVALID_INDEX {
			return TYPE_INVALID_INDEX
		}
		base = element
	}
	return TYPE_INVALID_INDEX
}

// LOCAL_COUNT_MAXIMUM caps the names one file binds inside its bodies.
const LOCAL_COUNT_MAXIMUM = 8192

// LOCAL_INDEX_MAXIMUM is the last local slot.
const LOCAL_INDEX_MAXIMUM = LOCAL_COUNT_MAXIMUM - 1

// SCOPE_DEPTH_MAXIMUM caps the blocks one body nests.
const SCOPE_DEPTH_MAXIMUM = 64

// BODY_COUNT_LOCAL holds how many locals stand in scope.
const BODY_COUNT_LOCAL = 0

// BODY_COUNT_SCOPE holds how deep the blocks stand.
const BODY_COUNT_SCOPE = 1

// BODY_COUNT_STAGED holds how many names one short declaration staged so far.
const BODY_COUNT_STAGED = 2

// BODY_COUNT_PLACE holds which value of a short declaration the fold stands on.
const BODY_COUNT_PLACE = 3

// BODY_COUNT_SLOT_COUNT is the counter count one body holds.
const BODY_COUNT_SLOT_COUNT = 4

// STAGED_NAME_MAXIMUM caps the names one short declaration binds at once.
const STAGED_NAME_MAXIMUM = 16

// EMBEDDED_DEPTH_MAXIMUM caps the embedded fields one selector walks through.
const EMBEDDED_DEPTH_MAXIMUM = 8

// Local_Index is one slot of the local name stack.
type Local_Index int32

// Local_Index_Invariants states the complete local stack.
func Local_Index_Invariants(value Local_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, LOCAL_INDEX_MAXIMUM).
		Ensure()
}

// Local is one name a block binds and the type it wears.
type Local struct {
	// Name holds the identifier the block bound.
	Name Name
	// Type is the type the name wears.
	Type Type_Index
}

// Local_Invariants composes the type one local wears.
func Local_Invariants(value Local, namespace aver.Namespace) {
	Name_Invariants(value.Name, namespace)
	Type_Index_Invariants(value.Type, namespace)
}

// Body_Type_Storage holds every folded tree type in caller-owned storage.
type Body_Type_Storage []Type_Index

// Body_Type_Storage_Invariants fixes tree-type storage and prevents growth.
func Body_Type_Storage_Invariants(value Body_Type_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == ast.NODE_COUNT_MAXIMUM,
		"Body type storage has complete length.",
	)
	aver.Always(cap(value) == ast.NODE_COUNT_MAXIMUM, "Body type storage cannot grow.")
}

// Local_Storage holds every block name in caller-owned storage.
type Local_Storage []Local

// Local_Storage_Invariants fixes local storage and prevents growth.
func Local_Storage_Invariants(value Local_Storage, _ aver.Namespace) {
	aver.Always(len(value) == LOCAL_COUNT_MAXIMUM, "Body local storage has complete length.")
	aver.Always(cap(value) == LOCAL_COUNT_MAXIMUM, "Body local storage cannot grow.")
}

// Scope_Storage holds every open block base in caller-owned storage.
type Scope_Storage []Local_Index

// Scope_Storage_Invariants fixes scope storage and prevents growth.
func Scope_Storage_Invariants(value Scope_Storage, _ aver.Namespace) {
	aver.Always(len(value) == SCOPE_DEPTH_MAXIMUM, "Body scope storage has complete length.")
	aver.Always(cap(value) == SCOPE_DEPTH_MAXIMUM, "Body scope storage cannot grow.")
}

// Staged_Storage holds one declaration's names in caller-owned storage.
type Staged_Storage []Name

// Staged_Storage_Invariants fixes staged-name storage and prevents growth.
func Staged_Storage_Invariants(value Staged_Storage, _ aver.Namespace) {
	aver.Always(len(value) == STAGED_NAME_MAXIMUM, "Body staged storage has complete length.")
	aver.Always(cap(value) == STAGED_NAME_MAXIMUM, "Body staged storage cannot grow.")
}

// Body_Count_Fields gives each unlike body cursor one name.
type Body_Count_Fields struct {
	// Local is how many names stand in scope.
	Local Count
	// Scope is how many blocks stand open.
	Scope Count
	// Staged is how many names one declaration retains.
	Staged Count
	// Place is which staged value the fold stands on.
	Place Count
}

// Body_Count_Fields_Invariants composes each body cursor once.
func Body_Count_Fields_Invariants(value Body_Count_Fields, namespace aver.Namespace) {
	Count_Invariants(value.Local, namespace)
	Count_Invariants(value.Scope, namespace)
	Count_Invariants(value.Staged, namespace)
	Count_Invariants(value.Place, namespace)
}

// Body_Count_Fields_Stored prevents one step from proving stale cursor values.
type Body_Count_Fields_Stored interface{}

// Body_Count_Fields_Stored_Invariants fixes body-cursor representation.
func Body_Count_Fields_Stored_Invariants(
	value Body_Count_Fields_Stored, _ aver.Namespace,
) {
	_, valid := value.(Body_Count_Fields)
	aver.Always(valid == (value != nil), "Body counts have expected storage type.")
}

// Body_Count_Envelope keeps mutable cursors behind one representation boundary.
type Body_Count_Envelope struct {
	// Body_Count_Fields stays embedded so passes retain concrete field access.
	Body_Count_Fields
}

// Body_Count_Envelope_Invariants composes body-cursor storage.
func Body_Count_Envelope_Invariants(value Body_Count_Envelope, namespace aver.Namespace) {
	Body_Count_Fields_Invariants(value.Body_Count_Fields, namespace)
}

// Body_Counts is the mutable cursor state one check carries.
type Body_Counts Body_Count_Envelope

// Body_Counts_Invariants fixes cursor representation without proving stale values.
func Body_Counts_Invariants(value Body_Counts, namespace aver.Namespace) {
	Body_Count_Fields_Stored_Invariants(
		Body_Count_Fields_Stored(value.Body_Count_Fields), namespace,
	)
}

// Body_Fields states the concrete check layout independently of its validated view.
type Body_Fields struct {
	// Types holds the type each tree slot folded to.
	Types Body_Type_Storage
	// Locals holds the names the blocks bound, innermost last.
	Locals Local_Storage
	// Scopes holds the local count each open block started with.
	Scopes Scope_Storage
	// Staged holds the names one short declaration states until its values fold.
	Staged Staged_Storage
	// Counts holds the local count, the block depth, and the staged name count.
	Counts Body_Counts
}

// Body_Fields_Invariants states every caller-owned store once.
func Body_Fields_Invariants(value Body_Fields, namespace aver.Namespace) {
	Body_Type_Storage_Invariants(value.Types, namespace)
	Local_Storage_Invariants(value.Locals, namespace)
	Scope_Storage_Invariants(value.Scopes, namespace)
	Staged_Storage_Invariants(value.Staged, namespace)
	Body_Counts_Invariants(value.Counts, namespace)
}

// Body_Fields_Stored prevents one recursive step from revalidating every body store.
type Body_Fields_Stored interface{}

// Body_Fields_Stored_Invariants fixes body representation.
func Body_Fields_Stored_Invariants(value Body_Fields_Stored, _ aver.Namespace) {
	_, valid := value.(Body_Fields)
	aver.Always(valid == (value != nil), "Body has expected storage type.")
}

// Body_Envelope keeps caller storage behind one representation boundary.
type Body_Envelope struct {
	// Body_Fields stays embedded so callers retain concrete field access.
	Body_Fields
}

// Body_Envelope_Invariants composes concrete caller storage.
func Body_Envelope_Invariants(value Body_Envelope, namespace aver.Namespace) {
	Body_Fields_Invariants(value.Body_Fields, namespace)
}

// Body is the state of one check pass. Every field is a slice for the reason the module states:
// a cursor field would owe its whole domain at every step that receives the state.
type Body Body_Envelope

// Body_Invariants fixes representation without revalidating every store in recursive steps.
func Body_Invariants(value Body, namespace aver.Namespace) {
	Body_Fields_Stored_Invariants(Body_Fields_Stored(value.Body_Fields), namespace)
}

// Body_Handle gives caller-owned check storage one identity.
type Body_Handle *Body

// Body_Handle_Invariants composes present body storage.
func Body_Handle_Invariants(value Body_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Body_Invariants(*value, namespace)
}

// Check folds the bodies of one file and records the type of every expression it reads. Every
// file resolves before any body is checked, thus a body reads a name any file of the module states.
func Check(
	subject Module_Handle, body Body_Handle, file File_Index, tree ast.Parse_State_Handle,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "check.ok") }()
	Module_Handle_Invariants(subject, "check.subject")
	Body_Handle_Invariants(body, "check.body")
	Module_Fields_Invariants(subject.Module_Fields, "check.module_storage")
	Body_Fields_Invariants(body.Body_Fields, "check.body_storage")
	File_Index_Invariants(file, "check.file")
	ast.Parse_State_Handle_Invariants(tree, "check.tree")
	subject.Counts[COUNT_CURRENT_FILE] = Count(file)
	subject.Counts[COUNT_OWNER] = 0
	body.Counts.Local = 0
	body.Counts.Scope = 0
	for slot := range body.Types {
		body.Types[slot] = TYPE_INVALID_INDEX
	}
	open_walk(subject, tree)
	open := descend(subject, tree)
	for bool(open) {
		check_declaration(subject, body, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
	return !failed(subject)
}

// Type_At reads the type one tree slot folded to, or the invalid type where the pass folded none.
func Type_At(body Body_Handle, node ast.Index) (index Type_Index) {
	defer func() { Type_Index_Invariants(index, "type_at.index") }()
	Body_Handle_Invariants(body, "type_at.body")
	Body_Fields_Invariants(body.Body_Fields, "type_at.storage")
	ast.Index_Invariants(node, "type_at.node")
	return body.Types[node]
}

// Folds the one declaration the walk stands on.
func check_declaration(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_declaration.subject")
	Body_Handle_Invariants(body, "check_declaration.body")
	ast.Parse_State_Handle_Invariants(tree, "check_declaration.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_FUNCTION:
		check_function(subject, body, tree)
	case ast.NODE_CONSTANT, ast.NODE_VARIABLE:
		check_values(subject, body, tree)
	}
}

// Folds the value expressions one package-level declaration states.
func check_values(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_values.subject")
	Body_Handle_Invariants(body, "check_values.body")
	ast.Parse_State_Handle_Invariants(tree, "check_values.tree")
	open := descend(subject, tree)
	valued := Boolean(false)
	for bool(open) {
		if bool(valued) {
			check_expression(subject, body, tree)
		}
		valued = valued || opens_value(subject, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
}

// Folds one function declaration: its receiver, its parameters, its results, and its body.
func check_function(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_function.subject")
	Body_Handle_Invariants(body, "check_function.body")
	ast.Parse_State_Handle_Invariants(tree, "check_function.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	subject.Counts[COUNT_PARAMETER] = 0
	open_scope(body)
	open := Boolean(true)
	for bool(open) {
		check_signature_part(subject, body, tree)
		open = advance(subject, tree)
	}
	close_scope(body)
	subject.Counts[COUNT_PARAMETER] = 0
	ascend(subject)
}

// Binds one part of a signature into the open scope, or folds the block behind it.
func check_signature_part(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_signature_part.subject")
	Body_Handle_Invariants(body, "check_signature_part.body")
	ast.Parse_State_Handle_Invariants(tree, "check_signature_part.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_TYPE_PARAMETER:
		declare_parameter(subject, tree)
	case ast.NODE_RECEIVER:
		check_receiver(subject, body, tree)
	case ast.NODE_PARAMETER, ast.NODE_RESULT:
		bind_signature_name(subject, body, tree)
	case ast.NODE_BLOCK:
		check_block(subject, body, tree)
	}
}

// Binds the receiver of a method, which stands in its body the way a parameter does.
func check_receiver(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_receiver.subject")
	Body_Handle_Invariants(body, "check_receiver.body")
	ast.Parse_State_Handle_Invariants(tree, "check_receiver.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	bind_signature_name(subject, body, tree)
	ascend(subject)
}

// Binds the name one parameter or result states and folds the type behind it.
func bind_signature_name(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "bind_signature_name.subject")
	Body_Handle_Invariants(body, "bind_signature_name.body")
	ast.Parse_State_Handle_Invariants(tree, "bind_signature_name.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	named := Boolean(node_kind(subject, tree) == ast.NODE_PARAMETER_NAME)
	if bool(named) {
		take_name(subject, tree)
		subject.Names.Bound = subject.Names.Value
		advance(subject, tree)
	}
	resolve_expression(subject, tree)
	if bool(named) {
		bind_local(subject, body)
	}
	ascend(subject)
}

// Opens one block scope. The names a block binds stand until it closes, thus a name of an inner
// block shadows the one an outer block states.
func open_scope(body Body_Handle) {
	Body_Handle_Invariants(body, "open_scope.body")
	if int(body.Counts.Scope) >= SCOPE_DEPTH_MAXIMUM {
		return
	}
	body.Scopes[body.Counts.Scope] = Local_Index(body.Counts.Local)
	body.Counts.Scope = body.Counts.Scope + 1
}

// Closes one block scope and drops every name it bound.
func close_scope(body Body_Handle) {
	Body_Handle_Invariants(body, "close_scope.body")
	if body.Counts.Scope == 0 {
		return
	}
	body.Counts.Scope = body.Counts.Scope - 1
	body.Counts.Local = Count(body.Scopes[body.Counts.Scope])
}

// Binds the name in the bound slot to the type in the result slot.
func bind_local(subject Module_Handle, body Body_Handle) {
	Module_Handle_Invariants(subject, "bind_local.subject")
	Body_Handle_Invariants(body, "bind_local.body")
	if int(body.Counts.Local) >= LOCAL_COUNT_MAXIMUM {
		return
	}
	slot := body.Counts.Local
	body.Locals[slot].Name = subject.Names.Bound
	body.Locals[slot].Type = Type_Index(subject.Counts[COUNT_RESULT])
	body.Counts.Local = slot + 1
}

// Puts the type the innermost binding of the name slot wears in the result slot, and reports
// whether a block binds that name at all.
func find_local(subject Module_Handle, body Body_Handle) (found Boolean) {
	defer func() { Boolean_Invariants(found, "find_local.found") }()
	Module_Handle_Invariants(subject, "find_local.subject")
	Body_Handle_Invariants(body, "find_local.body")
	for step := range int(body.Counts.Local) {
		slot := int(body.Counts.Local) - 1 - step
		if string(body.Locals[slot].Name) != string(subject.Names.Value) {
			continue
		}
		subject.Counts[COUNT_RESULT] = Count(body.Locals[slot].Type)
		return true
	}
	return false
}

// Folds one block: every statement it holds, in the scope it opens.
func check_block(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_block.subject")
	Body_Handle_Invariants(body, "check_block.body")
	ast.Parse_State_Handle_Invariants(tree, "check_block.tree")
	open_scope(body)
	entered := descend(subject, tree)
	open := entered
	for bool(open) {
		check_statement(subject, body, tree)
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	close_scope(body)
}

// Folds the one statement the walk stands on.
func check_statement(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_statement.subject")
	Body_Handle_Invariants(body, "check_statement.body")
	ast.Parse_State_Handle_Invariants(tree, "check_statement.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_BLOCK:
		check_block(subject, body, tree)
	case ast.NODE_DEFINE:
		check_define(subject, body, tree)
	case ast.NODE_DECLARATION_STATEMENT:
		check_local_declaration(subject, body, tree)
	case ast.NODE_RANGE:
		check_range(subject, body, tree)
	case ast.NODE_TYPE_SWITCH:
		check_type_switch(subject, body, tree)
	case ast.NODE_IF, ast.NODE_FOR, ast.NODE_SWITCH, ast.NODE_SELECT, ast.NODE_CASE,
		ast.NODE_DEFAULT, ast.NODE_LABEL:
		check_clause(subject, body, tree)
	case ast.NODE_IDENTIFIER, ast.NODE_INTEGER, ast.NODE_FLOAT, ast.NODE_IMAGINARY,
		ast.NODE_CHARACTER, ast.NODE_STRING, ast.NODE_BINARY, ast.NODE_UNARY,
		ast.NODE_CALL, ast.NODE_INDEX, ast.NODE_SLICE_EXPRESSION, ast.NODE_SELECTOR,
		ast.NODE_ASSERTION, ast.NODE_COMPOSITE, ast.NODE_PARENTHESIS,
		ast.NODE_FUNCTION_LITERAL:
		check_expression(subject, body, tree)
	default:
		check_children(subject, body, tree)
	}
}

// Folds every child of the node the walk stands on, which is what a statement that binds no name
// owes its expressions.
func check_children(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_children.subject")
	Body_Handle_Invariants(body, "check_children.body")
	ast.Parse_State_Handle_Invariants(tree, "check_children.tree")
	entered := descend(subject, tree)
	open := entered
	for bool(open) {
		check_statement(subject, body, tree)
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
}

// Folds one clause statement: a condition, a header, and the block behind it, in a scope of its
// own, because the names a header binds stand in the block alone.
func check_clause(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_clause.subject")
	Body_Handle_Invariants(body, "check_clause.body")
	ast.Parse_State_Handle_Invariants(tree, "check_clause.tree")
	open_scope(body)
	check_children(subject, body, tree)
	close_scope(body)
}

// Folds one short declaration and binds every name it states to the type its value folds to.
func check_define(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_define.subject")
	Body_Handle_Invariants(body, "check_define.body")
	ast.Parse_State_Handle_Invariants(tree, "check_define.tree")
	entered := descend(subject, tree)
	body.Counts.Staged = 0
	body.Counts.Place = 0
	open := entered
	for bool(open) {
		if bool(defines_name(subject, tree)) {
			stage_name(subject, body, tree)
		}
		if !bool(defines_name(subject, tree)) {
			bind_place(subject, body, tree)
			body.Counts.Place = body.Counts.Place + 1
		}
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	if body.Counts.Place == 1 {
		spread_tuple(subject, body)
	}
}

// Stages one name a short declaration states until the value behind it folds.
func stage_name(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "stage_name.subject")
	Body_Handle_Invariants(body, "stage_name.body")
	ast.Parse_State_Handle_Invariants(tree, "stage_name.tree")
	if int(body.Counts.Staged) >= STAGED_NAME_MAXIMUM {
		return
	}
	if !bool(take_name(subject, tree)) {
		return
	}
	body.Staged[body.Counts.Staged] = subject.Names.Value
	body.Counts.Staged = body.Counts.Staged + 1
}

// Folds one value of a short declaration and binds the name that stands at its place.
func bind_place(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "bind_place.subject")
	Body_Handle_Invariants(body, "bind_place.body")
	ast.Parse_State_Handle_Invariants(tree, "bind_place.tree")
	if node_kind(subject, tree) == ast.NODE_RANGE {
		check_range(subject, body, tree)
		return
	}
	place := body.Counts.Place
	check_expression(subject, body, tree)
	if place >= body.Counts.Staged {
		return
	}
	subject.Names.Bound = body.Staged[place]
	bind_local(subject, body)
}

// Binds every staged name to one member of the tuple one call folded to, because a call is the
// one value that states more than one type.
func spread_tuple(subject Module_Handle, body Body_Handle) {
	Module_Handle_Invariants(subject, "spread_tuple.subject")
	Body_Handle_Invariants(body, "spread_tuple.body")
	folded := Type_Index(subject.Counts[COUNT_RESULT])
	if subject.Types[folded].Kind != TYPE_TUPLE {
		spread_report(subject, body)
		return
	}
	member := Member_Index(subject.Types[folded].Members)
	for place := range int(body.Counts.Staged) {
		if member == MEMBER_ABSENT {
			return
		}
		subject.Names.Bound = body.Staged[place]
		subject.Counts[COUNT_RESULT] = Count(subject.Members[member].Type)
		bind_local(subject, body)
		member = Member_Index(subject.Members[member].Next)
	}
}

// Binds the two names of a form that states a value and a report about it: a map read, an
// assertion, and a receive each answer that way.
func spread_report(subject Module_Handle, body Body_Handle) {
	Module_Handle_Invariants(subject, "spread_report.subject")
	Body_Handle_Invariants(body, "spread_report.body")
	folded := subject.Counts[COUNT_RESULT]
	if body.Counts.Staged != 2 {
		return
	}
	subject.Names.Bound = body.Staged[0]
	subject.Counts[COUNT_RESULT] = folded
	bind_local(subject, body)
	subject.Names.Bound = body.Staged[1]
	subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_BOOLEAN)
	bind_local(subject, body)
}

// Binds the names one range clause states: the place a step stands at and the value it reads.
func check_range(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_range.subject")
	Body_Handle_Invariants(body, "check_range.body")
	ast.Parse_State_Handle_Invariants(tree, "check_range.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	check_expression(subject, body, tree)
	ascend(subject)
	bind_range_names(subject, body)
}

// Binds the staged names of a range clause to the types the value in the result slot states.
func bind_range_names(subject Module_Handle, body Body_Handle) {
	Module_Handle_Invariants(subject, "bind_range_names.subject")
	Body_Handle_Invariants(body, "bind_range_names.body")
	settle_result(subject)
	base := Underlying_Of(subject, Type_Index(subject.Counts[COUNT_RESULT]))
	kind := subject.Types[base].Kind
	place := Type_Index(TYPE_INT)
	value := Type_Index(subject.Types[base].Element)
	if kind == TYPE_MAP {
		place = Type_Index(subject.Types[base].Key)
	}
	if kind == TYPE_STRING {
		place = Type_Index(TYPE_INT)
		value = Type_Index(TYPE_INT_32)
	}
	if kind == TYPE_CHANNEL {
		place = Type_Index(subject.Types[base].Element)
		value = TYPE_INVALID_INDEX
	}
	for slot := range int(body.Counts.Staged) {
		subject.Names.Bound = body.Staged[slot]
		subject.Counts[COUNT_RESULT] = Count(place)
		if slot == 1 {
			subject.Counts[COUNT_RESULT] = Count(value)
		}
		bind_local(subject, body)
	}
	// A range clause binds every name it states here, thus the value pass behind it finds
	// nothing left to bind and never binds one name twice.
	body.Counts.Staged = 0
}

// Reports whether the node the walk stands on is a name the short declaration binds rather than
// the value behind the sign.
func defines_name(subject Module_Handle, tree ast.Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "defines_name.yes") }()
	Module_Handle_Invariants(subject, "defines_name.subject")
	ast.Parse_State_Handle_Invariants(tree, "defines_name.tree")
	if node_kind(subject, tree) != ast.NODE_IDENTIFIER {
		return false
	}
	position := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	if position == 0 {
		return true
	}
	kind := ast.Token_At(tree, position-1).Kind
	if kind == token.KIND_DEFINE {
		return false
	}
	return kind != token.KIND_ASSIGN
}

// Binds the names one local declaration states and folds the value behind them.
func check_local_declaration(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_local_declaration.subject")
	Body_Handle_Invariants(body, "check_local_declaration.body")
	ast.Parse_State_Handle_Invariants(tree, "check_local_declaration.tree")
	entered := descend(subject, tree)
	open := entered
	for bool(open) {
		check_local_names(subject, body, tree)
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
}

// Binds one constant, variable, or type a body states.
func check_local_names(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_local_names.subject")
	Body_Handle_Invariants(body, "check_local_names.body")
	ast.Parse_State_Handle_Invariants(tree, "check_local_names.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_TYPE {
		return
	}
	if kind != ast.NODE_CONSTANT {
		if kind != ast.NODE_VARIABLE {
			return
		}
	}
	entered := descend(subject, tree)
	body.Counts.Staged = 0
	open := entered
	named := Boolean(true)
	valued := Boolean(false)
	stated := TYPE_INVALID_INDEX
	for bool(open) {
		valued = valued || opens_value(subject, tree)
		fold_local_child(subject, body, tree, named, valued)
		if bool(types_child(named, valued)) {
			stated = Type_Index(subject.Counts[COUNT_RESULT])
		}
		open = advance(subject, tree)
		named = named && open
		named = named && opens_name(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	// The type a declaration states stands over the type its value folds to, which is what Go
	// says: the value converts to the type the name wears.
	if stated != TYPE_INVALID_INDEX {
		subject.Counts[COUNT_RESULT] = Count(stated)
	}
	bind_staged(subject, body)
}

// Folds one child of a local declaration: a name it states, the type behind them, or a value.
func fold_local_child(
	subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle,
	named Boolean, valued Boolean,
) {
	Module_Handle_Invariants(subject, "fold_local_child.subject")
	Body_Handle_Invariants(body, "fold_local_child.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_local_child.tree")
	Boolean_Invariants(named, "fold_local_child.named")
	Boolean_Invariants(valued, "fold_local_child.valued")
	if bool(names_child(named, valued)) {
		stage_name(subject, body, tree)
		return
	}
	if bool(types_child(named, valued)) {
		resolve_expression(subject, tree)
		return
	}
	check_expression(subject, body, tree)
}

// Binds every staged name to the type the last fold wrote, which is the type the declaration
// states or the type its value folded to.
func bind_staged(subject Module_Handle, body Body_Handle) {
	Module_Handle_Invariants(subject, "bind_staged.subject")
	Body_Handle_Invariants(body, "bind_staged.body")
	folded := subject.Counts[COUNT_RESULT]
	for slot := range int(body.Counts.Staged) {
		subject.Names.Bound = body.Staged[slot]
		subject.Counts[COUNT_RESULT] = folded
		bind_local(subject, body)
	}
}

// Folds one type switch: the operand it reads and each case that names a type of its own.
func check_type_switch(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_type_switch.subject")
	Body_Handle_Invariants(body, "check_type_switch.body")
	ast.Parse_State_Handle_Invariants(tree, "check_type_switch.tree")
	open_scope(body)
	entered := descend(subject, tree)
	open := entered
	for bool(open) {
		check_statement(subject, body, tree)
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	close_scope(body)
}

// Folds the expression the walk stands on, writes its type into the body, and puts that type in
// the result slot.
func check_expression(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_expression.subject")
	Body_Handle_Invariants(body, "check_expression.body")
	ast.Parse_State_Handle_Invariants(tree, "check_expression.tree")
	subject.Counts[COUNT_RESULT] = Count(TYPE_INVALID_INDEX)
	fold_expression(subject, body, tree)
	node := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	body.Types[node] = Type_Index(subject.Counts[COUNT_RESULT])
}

// Folds one expression by the class the parser gave it.
func fold_expression(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_expression.subject")
	Body_Handle_Invariants(body, "fold_expression.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_expression.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_IDENTIFIER:
		fold_identifier(subject, body, tree)
	case ast.NODE_INTEGER:
		subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_INTEGER)
	case ast.NODE_FLOAT:
		subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_FLOAT)
	case ast.NODE_IMAGINARY:
		subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_COMPLEX)
	case ast.NODE_CHARACTER:
		subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_RUNE)
	case ast.NODE_STRING:
		subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_STRING)
	case ast.NODE_BINARY:
		fold_binary(subject, body, tree)
	case ast.NODE_UNARY:
		fold_unary(subject, body, tree)
	case ast.NODE_CALL:
		fold_call(subject, body, tree)
	case ast.NODE_INDEX:
		fold_index(subject, body, tree)
	case ast.NODE_SLICE_EXPRESSION:
		fold_slice(subject, body, tree)
	case ast.NODE_SELECTOR:
		fold_selector(subject, body, tree)
	case ast.NODE_ASSERTION:
		fold_assertion(subject, body, tree)
	case ast.NODE_COMPOSITE:
		fold_composite(subject, body, tree)
	case ast.NODE_FUNCTION_LITERAL:
		fold_function_literal(subject, body, tree)
	case ast.NODE_PARENTHESIS:
		fold_group(subject, body, tree)
	default:
		check_children(subject, body, tree)
	}
}

// Folds one identifier: a name a block bound, a name its package states, or a predeclared word.
func fold_identifier(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_identifier.subject")
	Body_Handle_Invariants(body, "fold_identifier.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_identifier.tree")
	if !bool(take_name(subject, tree)) {
		return
	}
	if bool(find_local(subject, body)) {
		return
	}
	symbol := find_name(subject, current_package(subject))
	if symbol == SYMBOL_ABSENT {
		symbol = find_name(subject, PACKAGE_UNIVERSE)
	}
	if symbol == SYMBOL_ABSENT {
		return
	}
	subject.Counts[COUNT_RESULT] = Count(subject.Symbols[symbol].Type)
}

// Settles the type in the result slot: a value of an untyped kind reads as the type Go gives it
// where a body reads through it. A literal keeps its own kind, thus only a read settles one.
func settle_result(subject Module_Handle) {
	Module_Handle_Invariants(subject, "settle_result.subject")
	switch subject.Types[subject.Counts[COUNT_RESULT]].Kind {
	case TYPE_UNTYPED_BOOLEAN:
		subject.Counts[COUNT_RESULT] = Count(TYPE_BOOLEAN)
	case TYPE_UNTYPED_INTEGER:
		subject.Counts[COUNT_RESULT] = Count(TYPE_INT)
	case TYPE_UNTYPED_RUNE:
		subject.Counts[COUNT_RESULT] = Count(TYPE_INT_32)
	case TYPE_UNTYPED_FLOAT:
		subject.Counts[COUNT_RESULT] = Count(TYPE_FLOAT_64)
	case TYPE_UNTYPED_COMPLEX:
		subject.Counts[COUNT_RESULT] = Count(TYPE_COMPLEX_128)
	case TYPE_UNTYPED_STRING:
		subject.Counts[COUNT_RESULT] = Count(TYPE_STRING)
	}
}

// Folds one operation over a single value: an address, a read through a pointer, a receive, a
// sign, or a negation.
func fold_unary(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_unary.subject")
	Body_Handle_Invariants(body, "fold_unary.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_unary.tree")
	operator := ast.Token_At(tree, ast.Node_At(
		tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token).Kind
	if !bool(descend(subject, tree)) {
		return
	}
	check_expression(subject, body, tree)
	ascend(subject)
	folded := Type_Index(subject.Counts[COUNT_RESULT])
	switch operator {
	case token.KIND_AND:
		wrap_pointer(subject)
	case token.KIND_STAR, token.KIND_ARROW:
		subject.Counts[COUNT_RESULT] = Count(
			subject.Types[Underlying_Of(subject, folded)].Element)
	case token.KIND_NOT:
		subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_BOOLEAN)
	}
}

// Takes one pointer type over the type in the result slot and leaves it there.
func wrap_pointer(subject Module_Handle) {
	Module_Handle_Invariants(subject, "wrap_pointer.subject")
	over := Type_Element(subject.Counts[COUNT_RESULT])
	index := add_type(subject, TYPE_POINTER)
	subject.Types[index].Element = over
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Folds one call: a conversion states the type it names, a predeclared function states the type
// its own rule gives, and every other call states what its signature sends back.
func fold_call(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_call.subject")
	Body_Handle_Invariants(body, "fold_call.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_call.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	call_target(subject, body, tree)
	subject.Counts[COUNT_CALLEE] = subject.Counts[COUNT_RESULT]
	subject.Counts[COUNT_FIRST] = Count(TYPE_INVALID_INDEX)
	name := subject.Names.Bound
	typed := Boolean(string(name) == "make")
	typed = typed || Boolean(string(name) == "new")
	open := advance(subject, tree)
	for bool(open) {
		fold_argument(subject, body, tree, typed)
		typed = false
		if subject.Counts[COUNT_FIRST] == Count(TYPE_INVALID_INDEX) {
			subject.Counts[COUNT_FIRST] = subject.Counts[COUNT_RESULT]
		}
		open = advance(subject, tree)
	}
	ascend(subject)
	subject.Names.Bound = name
	fold_call_result(subject)
}

// Folds one argument of a call, as a type where the call takes one and as a value otherwise.
func fold_argument(
	subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle, typed Boolean,
) {
	Module_Handle_Invariants(subject, "fold_argument.subject")
	Body_Handle_Invariants(body, "fold_argument.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_argument.tree")
	Boolean_Invariants(typed, "fold_argument.typed")
	if !bool(typed) {
		check_expression(subject, body, tree)
		return
	}
	resolve_expression(subject, tree)
	record_type(subject, body, tree)
}

// Names what the head of a call stands for. The kind stands in the call slot and the type it
// wears in the result slot.
func call_target(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "call_target.subject")
	Body_Handle_Invariants(body, "call_target.body")
	ast.Parse_State_Handle_Invariants(tree, "call_target.tree")
	subject.Counts[COUNT_RESULT] = Count(TYPE_INVALID_INDEX)
	subject.Counts[COUNT_CALL_KIND] = Count(SYMBOL_FUNCTION)
	if node_kind(subject, tree) != ast.NODE_IDENTIFIER {
		check_expression(subject, body, tree)
		return
	}
	if !bool(take_name(subject, tree)) {
		subject.Counts[COUNT_CALL_KIND] = Count(SYMBOL_UNKNOWN)
		return
	}
	subject.Names.Bound = subject.Names.Value
	if bool(find_local(subject, body)) {
		record_type(subject, body, tree)
		return
	}
	symbol := find_name(subject, current_package(subject))
	if symbol == SYMBOL_ABSENT {
		symbol = find_name(subject, PACKAGE_UNIVERSE)
	}
	subject.Counts[COUNT_CALL_KIND] = Count(subject.Symbols[symbol].Kind)
	subject.Counts[COUNT_RESULT] = Count(subject.Symbols[symbol].Type)
	record_type(subject, body, tree)
}

// Writes the type in the result slot into the body at the slot the walk stands on, which is how
// a name a fold read answers for itself.
func record_type(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "record_type.subject")
	Body_Handle_Invariants(body, "record_type.body")
	ast.Parse_State_Handle_Invariants(tree, "record_type.tree")
	node := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	body.Types[node] = Type_Index(subject.Counts[COUNT_RESULT])
}

// Puts the type one call states in the result slot. The kind of its head, the type that head
// wears, and the type of its first argument each stand in a slot of their own.
func fold_call_result(subject Module_Handle) {
	Module_Handle_Invariants(subject, "fold_call_result.subject")
	kind := Symbol_Kind(subject.Counts[COUNT_CALL_KIND])
	callee := Type_Index(subject.Counts[COUNT_CALLEE])
	subject.Counts[COUNT_RESULT] = Count(TYPE_INVALID_INDEX)
	if kind == SYMBOL_TYPE {
		subject.Counts[COUNT_RESULT] = Count(callee)
		return
	}
	if kind == SYMBOL_BUILTIN {
		fold_builtin(subject)
		return
	}
	signature := Underlying_Of(subject, callee)
	if subject.Types[signature].Kind != TYPE_FUNCTION {
		return
	}
	results := Type_Index(subject.Types[signature].Element)
	member := Member_Index(subject.Types[results].Members)
	if member == MEMBER_ABSENT {
		return
	}
	if Member_Index(subject.Members[member].Next) == MEMBER_ABSENT {
		subject.Counts[COUNT_RESULT] = Count(subject.Members[member].Type)
		return
	}
	subject.Counts[COUNT_RESULT] = Count(results)
}

// Puts the type one predeclared function states in the result slot. The word the call spells
// stands in the bound slot, thus one body serves every builtin the universe binds.
func fold_builtin(subject Module_Handle) {
	Module_Handle_Invariants(subject, "fold_builtin.subject")
	first := subject.Counts[COUNT_FIRST]
	subject.Counts[COUNT_RESULT] = Count(TYPE_INVALID_INDEX)
	switch string(subject.Names.Bound) {
	case "len", "cap", "copy":
		subject.Counts[COUNT_RESULT] = Count(TYPE_INT)
	case "make", "append", "min", "max":
		subject.Counts[COUNT_RESULT] = first
	case "new":
		subject.Counts[COUNT_RESULT] = first
		wrap_pointer(subject)
	case "complex":
		subject.Counts[COUNT_RESULT] = Count(TYPE_COMPLEX_128)
	case "real", "imag":
		subject.Counts[COUNT_RESULT] = Count(TYPE_FLOAT_64)
	}
}

// Folds one operation over two values. A comparison states a truth whatever it compares, and
// every other operation states the type of the side that is not a bare literal.
func fold_binary(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_binary.subject")
	Body_Handle_Invariants(body, "fold_binary.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_binary.tree")
	operator := ast.Token_At(tree, ast.Node_At(
		tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token).Kind
	if !bool(descend(subject, tree)) {
		return
	}
	check_expression(subject, body, tree)
	subject.Counts[COUNT_LEFT] = subject.Counts[COUNT_RESULT]
	if bool(advance(subject, tree)) {
		check_expression(subject, body, tree)
	}
	ascend(subject)
	take_wider(subject)
	switch operator {
	case token.KIND_EQUAL, token.KIND_NOT_EQUAL, token.KIND_LESS, token.KIND_GREATER:
		subject.Counts[COUNT_RESULT] = Count(TYPE_UNTYPED_BOOLEAN)
	}
}

// Puts the side of an operation that states the type of its result in the result slot. An
// untyped side takes the type of the other one, which is what Go does with a literal beside a
// named value.
func take_wider(subject Module_Handle) {
	Module_Handle_Invariants(subject, "take_wider.subject")
	left := subject.Counts[COUNT_LEFT]
	right := subject.Counts[COUNT_RESULT]
	kind := subject.Types[left].Kind
	settled := uint8(kind) < uint8(TYPE_UNTYPED_BOOLEAN)
	if settled {
		subject.Counts[COUNT_RESULT] = left
		return
	}
	if subject.Types[right].Kind == TYPE_INVALID {
		subject.Counts[COUNT_RESULT] = left
	}
}

// Folds one bracket suffix: an element of a slice, an array, or a string, the value of a map, or
// the type a generic instance states.
func fold_index(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_index.subject")
	Body_Handle_Invariants(body, "fold_index.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_index.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	check_expression(subject, body, tree)
	settle_result(subject)
	subject.Counts[COUNT_OVER] = Count(Underlying_Of(
		subject, Type_Index(subject.Counts[COUNT_RESULT])))
	if bool(advance(subject, tree)) {
		check_expression(subject, body, tree)
	}
	ascend(subject)
	read_index(subject)
}

// Puts the type one bracket suffix reads out of the value in the over slot in the result slot.
func read_index(subject Module_Handle) {
	Module_Handle_Invariants(subject, "read_index.subject")
	over := Type_Index(subject.Counts[COUNT_OVER])
	subject.Counts[COUNT_RESULT] = Count(subject.Types[over].Element)
	if subject.Types[over].Kind == TYPE_STRING {
		subject.Counts[COUNT_RESULT] = Count(TYPE_UINT_8)
		return
	}
	if subject.Types[over].Kind != TYPE_POINTER {
		return
	}
	inner := Underlying_Of(subject, Type_Index(subject.Types[over].Element))
	subject.Counts[COUNT_RESULT] = Count(subject.Types[inner].Element)
}

// Folds one slice suffix, which states a slice over the same element the value behind it holds.
func fold_slice(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_slice.subject")
	Body_Handle_Invariants(body, "fold_slice.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_slice.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	check_expression(subject, body, tree)
	settle_result(subject)
	subject.Counts[COUNT_OVER] = subject.Counts[COUNT_RESULT]
	open := advance(subject, tree)
	for bool(open) {
		check_expression(subject, body, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
	read_slice(subject)
}

// Puts the type one slice suffix states in the result slot. A slice of a string is a string, and
// a slice of an array is a slice over the element that array holds.
func read_slice(subject Module_Handle) {
	Module_Handle_Invariants(subject, "read_slice.subject")
	over := Type_Index(subject.Counts[COUNT_OVER])
	subject.Counts[COUNT_RESULT] = Count(over)
	base := Underlying_Of(subject, over)
	if subject.Types[base].Kind != TYPE_ARRAY {
		return
	}
	element := subject.Types[base].Element
	index := add_type(subject, TYPE_SLICE)
	subject.Types[index].Element = element
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Folds one selector: a name another package states, a field of a struct, or a method of a named
// type.
func fold_selector(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_selector.subject")
	Body_Handle_Invariants(body, "fold_selector.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_selector.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	valued := selector_operand(subject, body, tree)
	if !bool(advance(subject, tree)) {
		ascend(subject)
		return
	}
	if !bool(take_name(subject, tree)) {
		ascend(subject)
		return
	}
	if !bool(valued) {
		fold_package_member(subject)
		ascend(subject)
		return
	}
	read_member(subject)
	ascend(subject)
}

// Folds the value a selector reads from and reports whether it names a value at all. A head that
// names an import states a package, and the name behind it stands in that package alone.
func selector_operand(
	subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle,
) (valued Boolean) {
	defer func() { Boolean_Invariants(valued, "selector_operand.valued") }()
	Module_Handle_Invariants(subject, "selector_operand.subject")
	Body_Handle_Invariants(body, "selector_operand.body")
	ast.Parse_State_Handle_Invariants(tree, "selector_operand.tree")
	subject.Counts[COUNT_OVER] = Count(TYPE_INVALID_INDEX)
	if node_kind(subject, tree) != ast.NODE_IDENTIFIER {
		check_expression(subject, body, tree)
		subject.Counts[COUNT_OVER] = subject.Counts[COUNT_RESULT]
		return true
	}
	if !bool(take_name(subject, tree)) {
		return false
	}
	subject.Names.Bound = subject.Names.Value
	if bool(find_local(subject, body)) {
		record_type(subject, body, tree)
		subject.Counts[COUNT_OVER] = subject.Counts[COUNT_RESULT]
		return true
	}
	if bool(package_qualifier(subject)) {
		return false
	}
	package_member_operand(subject, body, tree)
	return true
}

// Folds the head of a selector that names a member of its own package or a predeclared word.
func package_member_operand(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "package_member_operand.subject")
	Body_Handle_Invariants(body, "package_member_operand.body")
	ast.Parse_State_Handle_Invariants(tree, "package_member_operand.tree")
	symbol := find_name(subject, current_package(subject))
	if symbol == SYMBOL_ABSENT {
		symbol = find_name(subject, PACKAGE_UNIVERSE)
	}
	subject.Counts[COUNT_RESULT] = Count(subject.Symbols[symbol].Type)
	record_type(subject, body, tree)
	subject.Counts[COUNT_OVER] = subject.Counts[COUNT_RESULT]
}

// Reports whether the name in the bound slot names an import of the file the pass stands in, and
// puts the package it names in the target slot.
func package_qualifier(subject Module_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "package_qualifier.yes") }()
	Module_Handle_Invariants(subject, "package_qualifier.subject")
	file := subject.Files[subject.Counts[COUNT_CURRENT_FILE]]
	for slot := range int(file.Count) {
		symbol := Symbol_Index(int(file.First) + slot)
		if subject.Symbols[symbol].Kind != SYMBOL_IMPORT {
			continue
		}
		held := string(subject.Symbols[symbol].Name)
		if held != string(subject.Names.Bound) {
			continue
		}
		subject.Counts[COUNT_TARGET] = Count(subject.Symbols[symbol].Target)
		return true
	}
	return false
}

// Puts the type one package member wears in the result slot. The qualifier stands in the target
// slot, thus the name alone states which member the selector reads.
func fold_package_member(subject Module_Handle) {
	Module_Handle_Invariants(subject, "fold_package_member.subject")
	symbol := find_name(subject, Package_Index(subject.Counts[COUNT_TARGET]))
	subject.Counts[COUNT_RESULT] = Count(subject.Symbols[symbol].Type)
}

// Puts the type one field or method of the value in the over slot wears in the result slot. A
// pointer reads the fields of the value it stands over, and an embedded field carries the fields
// it holds up to the struct that states it.
func read_member(subject Module_Handle) {
	Module_Handle_Invariants(subject, "read_member.subject")
	over := Type_Index(subject.Counts[COUNT_OVER])
	named := over
	base := Underlying_Of(subject, over)
	if subject.Types[base].Kind == TYPE_POINTER {
		named = Type_Index(subject.Types[base].Element)
		base = Underlying_Of(subject, named)
	}
	subject.Counts[COUNT_CHAIN] = Count(subject.Types[named].Members)
	read_chain(subject)
	if subject.Counts[COUNT_RESULT] != Count(TYPE_INVALID_INDEX) {
		return
	}
	if subject.Types[base].Kind != TYPE_STRUCTURE {
		return
	}
	subject.Counts[COUNT_CHAIN] = Count(subject.Types[base].Members)
	read_chain(subject)
	if subject.Counts[COUNT_RESULT] != Count(TYPE_INVALID_INDEX) {
		return
	}
	subject.Counts[COUNT_OVER] = Count(base)
	read_embedded(subject)
}

// Puts the type the member of the chain in the chain slot that answers to the name slot wears in
// the result slot, or the invalid type where no member of the chain answers to it.
func read_chain(subject Module_Handle) {
	Module_Handle_Invariants(subject, "read_chain.subject")
	subject.Counts[COUNT_RESULT] = Count(TYPE_INVALID_INDEX)
	current := Member_Index(subject.Counts[COUNT_CHAIN])
	for current != MEMBER_ABSENT {
		symbol := Symbol_Index(subject.Members[current].Symbol)
		held := string(subject.Symbols[symbol].Name)
		if held == string(subject.Names.Value) {
			subject.Counts[COUNT_RESULT] = Count(subject.Members[current].Type)
			return
		}
		current = Member_Index(subject.Members[current].Next)
	}
}

// Puts the type one field of an embedded struct wears in the result slot. A struct carries the
// fields of the types it embeds, thus a selector reads them the way it reads a field of its own.
func read_embedded(subject Module_Handle) {
	Module_Handle_Invariants(subject, "read_embedded.subject")
	base := Type_Index(subject.Counts[COUNT_OVER])
	current := Member_Index(subject.Types[base].Members)
	for step := range EMBEDDED_DEPTH_MAXIMUM {
		if current == MEMBER_ABSENT {
			return
		}
		if Symbol_Index(subject.Members[current].Symbol) == SYMBOL_ABSENT {
			inner := Underlying_Of(subject, Type_Index(subject.Members[current].Type))
			subject.Counts[COUNT_CHAIN] = Count(subject.Types[inner].Members)
			read_chain(subject)
			if subject.Counts[COUNT_RESULT] != Count(TYPE_INVALID_INDEX) {
				return
			}
		}
		aver.Always(
			step < EMBEDDED_DEPTH_MAXIMUM,
			"An embedded walk stops inside the depth it holds.",
		)
		current = Member_Index(subject.Members[current].Next)
	}
}

// Folds one assertion, which states the type it names whatever the value behind it holds.
func fold_assertion(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_assertion.subject")
	Body_Handle_Invariants(body, "fold_assertion.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_assertion.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	check_expression(subject, body, tree)
	folded := Count(TYPE_INVALID_INDEX)
	if bool(advance(subject, tree)) {
		resolve_expression(subject, tree)
		folded = subject.Counts[COUNT_RESULT]
	}
	ascend(subject)
	subject.Counts[COUNT_RESULT] = folded
}

// Folds one composite literal, which states the type it opens with. Every element folds too,
// thus a caller reads the type of what stands inside the braces.
func fold_composite(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_composite.subject")
	Body_Handle_Invariants(body, "fold_composite.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_composite.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	resolve_expression(subject, tree)
	folded := subject.Counts[COUNT_RESULT]
	open := advance(subject, tree)
	for bool(open) {
		check_element(subject, body, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
	subject.Counts[COUNT_RESULT] = folded
}

// Folds one element of a composite literal. A keyed element states a field name on its left,
// which names no value of its own.
func check_element(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "check_element.subject")
	Body_Handle_Invariants(body, "check_element.body")
	ast.Parse_State_Handle_Invariants(tree, "check_element.tree")
	if node_kind(subject, tree) != ast.NODE_KEY_VALUE {
		check_expression(subject, body, tree)
		return
	}
	if !bool(descend(subject, tree)) {
		return
	}
	if bool(advance(subject, tree)) {
		check_expression(subject, body, tree)
	}
	ascend(subject)
}

// Folds one function literal: its signature and the block behind it, in a scope of its own.
func fold_function_literal(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_function_literal.subject")
	Body_Handle_Invariants(body, "fold_function_literal.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_function_literal.tree")
	index := add_type(subject, TYPE_FUNCTION)
	subject.Counts[COUNT_RESULT] = Count(index)
	push_owner(subject)
	open_scope(body)
	entered := descend(subject, tree)
	open := entered
	for bool(open) {
		fold_literal_part(subject, body, tree)
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	close_scope(body)
	pop_owner(subject)
	subject.Counts[COUNT_RESULT] = Count(index)
}

// Folds one part of a function literal: a parameter, a result, or the block behind them.
func fold_literal_part(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_literal_part.subject")
	Body_Handle_Invariants(body, "fold_literal_part.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_literal_part.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLOCK {
		check_block(subject, body, tree)
		return
	}
	if kind == ast.NODE_PARAMETER {
		attach_member(subject, tree, false)
	}
	if kind == ast.NODE_RESULT {
		attach_member(subject, tree, true)
	}
	bind_signature_name(subject, body, tree)
}

// Folds one grouping, which states the type of the expression inside it.
func fold_group(subject Module_Handle, body Body_Handle, tree ast.Parse_State_Handle) {
	Module_Handle_Invariants(subject, "fold_group.subject")
	Body_Handle_Invariants(body, "fold_group.body")
	ast.Parse_State_Handle_Invariants(tree, "fold_group.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	check_expression(subject, body, tree)
	ascend(subject)
}

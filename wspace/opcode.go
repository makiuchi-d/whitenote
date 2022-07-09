package wspace

import "fmt"

// OpCode is an operation code contains the command and its parameter, and the position of the definition on the loaded code segment.
type OpCode struct {
	Cmd   Command
	Param any // int or string

	Seg int // code segment number
	Pos int // code position
}

// Command is a command code of the whitespace.
//go:generate stringer -type=Command
type Command int

const (
	_ Command = iota

	// StackManipulation

	Push    // [space][space] Number : Push the number onto the stack
	Dup     // [space][LF][space] - : Duplicate the top item on the stack
	Copy    // [space][Tab][Space] Number : Copy the nth item on the stack (given by the argument) onto the top of the stack
	Swap    // [Space][LF][Tab] - : Swap the top two items on the stack
	Discard // [Space][LF][LF] - : Discard the top item on the stack
	Slide   // [Space][Tab][LF] Number : Slide n items off the stack, keeping the top item

	// Arithmetic

	Add // [Tab][Space][Space][Space] - : Addition
	Sub // [Tab][Space][Space][Tab] - : Subtraction
	Mul // [Tab][Space][Space][LF] - : Multiplication
	Div // [Tab][Space][Tab][Space] - : Integer Division
	Mod // [Tab][Space][Tab][Tab] - : Modulo

	// Heap access

	Store    // [Tab][Tab][Space] - : Store in heap
	Retrieve // [Tab][Tab][Tab] - : Retrieve from heap

	// Flow Control

	Mark  // [LF][Space][Space] Label : Mark a location in the program
	Call  // [LF][Space][Tab] Label : Call a subroutine
	Jump  // [LF][Space][LF] Label : Jump to a label
	JZero // [LF][Tab][Space] Label : Jump to a label if the top of the stack is zero
	JNeg  // [LF][Tab][Tab] Label : Jump to a label if the top of the stack is negative
	Ret   // [LF][Tab][LF] - : End a subroutine and transfer control back to the caller
	End   // [LF][LF][LF] - : End the program

	// I/O

	WriteChar // [Tab][LF][Space][Space] - : Output the character at the top of the stack
	WriteNum  // [Tab][LF][Space][Tab] - : Output the number at the top of the stack
	ReadChar  // [Tab][LF][Tab][Space] - : Read a character and place it in the location given by the top of the stack
	ReadNum   // [Tab][LF][Tab][Tab] - : Read a number and place it in the location given by the top of the stack
)

func (op OpCode) String() string {
	param := ""
	if op.Param != nil {
		param = fmt.Sprintf(" %#v", op.Param)
	}
	return fmt.Sprintf("(%v:%v) %s%s", op.Seg, op.Pos, op.Cmd, param)
}

package wspace

type Error string

const (
	ErrIncompleteCode = Error("incomplete sequence")
	ErrInvalidCode    = Error("invalid sequence")
	ErrOverflow       = Error("integer overflow")
	ErrDuplicateLabel = Error("label already exists")
	ErrTerminated     = Error("vm already terminated")
	ErrNotLoaded      = Error("no program loaded")
	ErrNotEnoughStack = Error("not enough stack to do")
	ErrInvalidParam   = Error("invalid parameter")
	ErrUndefinedLabel = Error("undefined label")
	ErrEmptyCallStack = Error("callstack is empty")
	ErrContextDone    = Error("context done")

	ErrUnknownOpCode = Error("unknown opcode")
)

func (e Error) Error() string {
	return string(e)
}

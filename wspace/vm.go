package wspace

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
)

// VM whitespace virtual machine.
type VM struct {
	Program []OpCode
	Labels  map[string]int

	Terminated bool
	PC         int
	Stack      []int
	Heap       map[int]int
	CallStack  []int

	Seg int // segment number to be loaded
}

// InputReader is the stdin interface for Step()
type InputReader interface {
	io.Reader
	io.ByteReader
}

// New VM
func New() *VM {
	return &VM{
		Program:   make([]OpCode, 0),
		Labels:    make(map[string]int),
		Stack:     make([]int, 0),
		Heap:      make(map[int]int),
		CallStack: make([]int, 0),
		Seg:       1,
	}
}

// Load loads code segment to VM
// return: segment number, read size, error
func (vm *VM) Load(code []byte) (int, int, error) {
	pos := 0
	defer func() {
		if pos > 0 {
			vm.Seg++
		}
	}()
	for pos < len(code) {
		_, p := findWhite(code[pos:])
		if p < 0 {
			pos = len(code)
			break
		}
		pos += p
		c3, read := read3code(code[pos:])
		if read == 0 {
			return vm.Seg, pos, ErrIncompleteCode
		}
		switch c3 {
		case "   ", "  \t": // Push number
			n, r, err := readNum(code[pos+read-1:]) // contains last white.
			if err != nil {
				return vm.Seg, pos, err
			}
			read += r - 1
			vm.appendOpCodeNumber(Push, n, pos)
		case " \n ": // Dup
			vm.appendOpCode(Dup, pos)

		case " \t ": // Copy
			n, r, err := readNum(code[pos+read:])
			if err != nil {
				return vm.Seg, pos, err
			}
			read += r
			vm.appendOpCodeNumber(Copy, n, pos)
		case " \n\t": // Swap
			vm.appendOpCode(Swap, pos)

		case " \n\n": // Discard
			vm.appendOpCode(Discard, pos)

		case " \t\n": // Slide
			n, r, err := readNum(code[pos+read:])
			if err != nil {
				return vm.Seg, pos, err
			}
			read += r
			vm.appendOpCodeNumber(Slide, n, pos)
		case "\t  ": // Add, Sub, Mul
			c, p := findWhite(code[pos+read:])
			if p < 0 {
				return vm.Seg, pos, ErrIncompleteCode
			}
			read += p + 1
			switch c {
			case ' ':
				vm.appendOpCode(Add, pos)
			case '\t':
				vm.appendOpCode(Sub, pos)
			case '\n':
				vm.appendOpCode(Mul, pos)
			}
		case "\t \t": // Div, Mod
			c, p := findWhite(code[pos+read:])
			if p < 0 {
				return vm.Seg, pos, ErrIncompleteCode
			}
			read += p + 1
			switch c {
			case ' ':
				vm.appendOpCode(Div, pos)
			case '\t':
				vm.appendOpCode(Mod, pos)
			case '\n':
				return vm.Seg, pos, ErrInvalidCode
			}
		case "\t\t ": // Store
			vm.appendOpCode(Store, pos)
		case "\t\t\t": // Retrieve
			vm.appendOpCode(Retrieve, pos)
		case "\n  ": // Mark
			l, r, err := readLabel(code[pos+read:])
			if err != nil {
				return vm.Seg, pos, err
			}
			if _, exists := vm.Labels[l]; exists {
				return vm.Seg, pos, ErrDuplicateLabel
			}
			read += r
			vm.Labels[l] = len(vm.Program)
			vm.appendOpCodeLabel(Mark, l, pos) // for visualization
		case "\n \t": // Call
			l, r, err := readLabel(code[pos+read:])
			if err != nil {
				return vm.Seg, pos, err
			}
			read += r
			vm.appendOpCodeLabel(Call, l, pos)
		case "\n \n": // Jump
			l, r, err := readLabel(code[pos+read:])
			if err != nil {
				return vm.Seg, pos, err
			}
			read += r
			vm.appendOpCodeLabel(Jump, l, pos)
		case "\n\t ": // JZero
			l, r, err := readLabel(code[pos+read:])
			if err != nil {
				return vm.Seg, pos, err
			}
			read += r
			vm.appendOpCodeLabel(JZero, l, pos)
		case "\n\t\t": // JNeg
			l, r, err := readLabel(code[pos+read:])
			if err != nil {
				return vm.Seg, pos, err
			}
			read += r
			vm.appendOpCodeLabel(JNeg, l, pos)
		case "\n\t\n": // Ret
			vm.appendOpCode(Ret, pos)
		case "\n\n\n": // End
			vm.appendOpCode(End, pos)
		case "\t\n ": // WriteChar, WriteNum
			c, p := findWhite(code[pos+read:])
			if p < 0 {
				return vm.Seg, pos, ErrIncompleteCode
			}
			read += p + 1
			switch c {
			case ' ':
				vm.appendOpCode(WriteChar, pos)
			case '\t':
				vm.appendOpCode(WriteNum, pos)
			default:
				return vm.Seg, pos, ErrInvalidCode
			}
		case "\t\n\t": // ReadChar, ReadNum
			c, p := findWhite(code[pos+read:])
			if p < 0 {
				return vm.Seg, pos, ErrIncompleteCode
			}
			read += p + 1
			switch c {
			case ' ':
				vm.appendOpCode(ReadChar, pos)
			case '\t':
				vm.appendOpCode(ReadNum, pos)
			default:
				return vm.Seg, pos, ErrInvalidCode
			}
		default:
			return vm.Seg, pos, ErrInvalidCode
		}
		pos += read
	}

	return vm.Seg, pos, nil
}

// CurrentOpcode returns current opcode.
func (vm *VM) CurrentOpCode() *OpCode {
	if vm.PC >= len(vm.Program) {
		return nil
	}
	return &vm.Program[vm.PC]
}

// Run the program.
func (vm *VM) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	b, ok := in.(InputReader)
	if !ok {
		b = bufio.NewReader(in)
	}
	for !vm.Terminated {
		select {
		case <-ctx.Done():
			return ErrContextDone
		default:
		}
		err := vm.Step(b, out)
		if err != nil {
			if err == ErrNotLoaded {
				break
			}
			return err
		}
	}
	return nil
}

// Step runs an opecode.
func (vm *VM) Step(in InputReader, out io.Writer) error {
	if vm.Terminated {
		return ErrTerminated
	}
	if vm.PC >= len(vm.Program) {
		return ErrNotLoaded
	}

	switch op := vm.Program[vm.PC]; op.Cmd {
	case Push:
		vm.Stack = append(vm.Stack, op.Param.(int))
		vm.PC++
	case Dup:
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		vm.Stack = append(vm.Stack, vm.Stack[len(vm.Stack)-1])
		vm.PC++
	case Copy:
		idx := op.Param.(int)
		if idx < 0 || idx >= len(vm.Stack) {
			vm.Terminated = true
			return ErrInvalidParam
		}
		vm.Stack = append(vm.Stack, vm.Stack[len(vm.Stack)-idx-1])
		vm.PC++
	case Swap:
		if len(vm.Stack) < 2 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		last := len(vm.Stack) - 1
		vm.Stack[last], vm.Stack[last-1] = vm.Stack[last-1], vm.Stack[last]
		vm.PC++
	case Discard:
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		vm.Stack = vm.Stack[:len(vm.Stack)-1]
		vm.PC++
	case Slide:
		n := op.Param.(int)
		if n < 0 || n >= len(vm.Stack)-1 {
			vm.Terminated = true
			return ErrInvalidParam
		}
		top := vm.Stack[len(vm.Stack)-1]
		vm.Stack = vm.Stack[:len(vm.Stack)-n]
		vm.Stack[len(vm.Stack)-1] = top
		vm.PC++
	case Add:
		if len(vm.Stack) < 2 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		p := len(vm.Stack) - 1
		vm.Stack[p-1] += vm.Stack[p]
		vm.Stack = vm.Stack[:p]
		vm.PC++
	case Sub:
		if len(vm.Stack) < 2 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		p := len(vm.Stack) - 1
		vm.Stack[p-1] -= vm.Stack[p]
		vm.Stack = vm.Stack[:p]
		vm.PC++
	case Mul:
		if len(vm.Stack) < 2 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		p := len(vm.Stack) - 1
		vm.Stack[p-1] *= vm.Stack[p]
		vm.Stack = vm.Stack[:p]
		vm.PC++
	case Div:
		if len(vm.Stack) < 2 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		p := len(vm.Stack) - 1
		vm.Stack[p-1] /= vm.Stack[p]
		vm.Stack = vm.Stack[:p]
		vm.PC++
	case Mod:
		if len(vm.Stack) < 2 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		p := len(vm.Stack) - 1
		vm.Stack[p-1] %= vm.Stack[p]
		vm.Stack = vm.Stack[:p]
		vm.PC++
	case Store:
		if len(vm.Stack) < 2 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		l := len(vm.Stack)
		v := vm.Stack[l-1]
		a := vm.Stack[l-2]
		vm.Stack = vm.Stack[:l-2]
		vm.Heap[a] = v
		vm.PC++
	case Retrieve:
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		l := len(vm.Stack)
		a := vm.Stack[l-1]
		v, _ := vm.Heap[a]
		vm.Stack[l-1] = v
		vm.PC++
	case Mark:
		// nothing to do.
		vm.PC++
	case Call:
		p, ok := vm.Labels[op.Param.(string)]
		if !ok {
			vm.Terminated = true
			return ErrUndefinedLabel
		}
		vm.CallStack = append(vm.CallStack, vm.PC+1)
		vm.PC = p
	case Jump:
		p, ok := vm.Labels[op.Param.(string)]
		if !ok {
			vm.Terminated = true
			return ErrUndefinedLabel
		}
		vm.PC = p
	case JZero:
		p, ok := vm.Labels[op.Param.(string)]
		if !ok {
			vm.Terminated = true
			return ErrUndefinedLabel
		}
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		l := len(vm.Stack)
		if vm.Stack[l-1] == 0 {
			vm.PC = p
		} else {
			vm.PC++
		}
		vm.Stack = vm.Stack[:l-1]
	case JNeg:
		p, ok := vm.Labels[op.Param.(string)]
		if !ok {
			vm.Terminated = true
			return ErrUndefinedLabel
		}
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		l := len(vm.Stack)
		if vm.Stack[l-1] < 0 {
			vm.PC = p
		} else {
			vm.PC++
		}
		vm.Stack = vm.Stack[:l-1]
	case Ret:
		if len(vm.CallStack) == 0 {
			vm.Terminated = true
			return ErrEmptyCallStack
		}
		l := len(vm.CallStack)
		vm.PC = vm.CallStack[l-1]
		vm.CallStack = vm.CallStack[:l-1]
	case End:
		vm.Terminated = true
	case WriteChar:
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		l := len(vm.Stack)
		_, err := out.Write([]byte{byte(vm.Stack[l-1])})
		if err != nil {
			vm.Terminated = true
			return err
		}
		vm.Stack = vm.Stack[:l-1]
		vm.PC++
	case WriteNum:
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		l := len(vm.Stack)
		_, err := fmt.Fprintf(out, "%d", vm.Stack[l-1])
		if err != nil {
			vm.Terminated = true
			return err
		}
		vm.Stack = vm.Stack[:l-1]
		vm.PC++
	case ReadChar:
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		c, err := in.ReadByte()
		if err != nil {
			vm.Terminated = true
			return err
		}
		l := len(vm.Stack)
		vm.Heap[vm.Stack[l-1]] = int(c)
		vm.Stack = vm.Stack[:l-1]
		vm.PC++
	case ReadNum:
		if len(vm.Stack) == 0 {
			vm.Terminated = true
			return ErrNotEnoughStack
		}
		var n int
		_, err := fmt.Fscanln(in, &n)
		if err != nil {
			vm.Terminated = true
			return err
		}
		l := len(vm.Stack)
		vm.Heap[vm.Stack[l-1]] = n
		vm.Stack = vm.Stack[:l-1]
		vm.PC++

	default:
		vm.Terminated = true
		return ErrUnknownOpCode
	}

	return nil
}

func (vm *VM) appendOpCode(cmd Command, pos int) {
	vm.Program = append(vm.Program, OpCode{Cmd: cmd, Seg: vm.Seg, Pos: pos})
}

func (vm *VM) appendOpCodeNumber(cmd Command, n int, pos int) {
	vm.Program = append(vm.Program, OpCode{Cmd: cmd, Param: n, Seg: vm.Seg, Pos: pos})
}

func (vm *VM) appendOpCodeLabel(cmd Command, label string, pos int) {
	vm.Program = append(vm.Program, OpCode{Cmd: cmd, Param: label, Seg: vm.Seg, Pos: pos})
}

func findWhite(code []byte) (byte, int) {
	for pos, c := range code {
		if c == ' ' || c == '\t' || c == '\n' {
			return c, pos
		}
	}
	return 0, -1
}

func read3code(code []byte) (string, int) {
	r := []byte{0, 0, 0}
	cur := 0
	for i := 0; i < 3; i++ {
		c, p := findWhite(code[cur:])
		if p < 0 {
			return "", 0
		}
		r[i] = c
		cur += p + 1
	}
	return string(r), cur
}

func readNum(code []byte) (int, int, error) {
	c, p := findWhite(code)
	if p < 0 {
		return 0, 0, ErrIncompleteCode
	}
	cur := p + 1
	if c == '\n' {
		return 0, cur, nil
	}
	neg := c == '\t'
	n := 0
	for {
		c, p := findWhite(code[cur:])
		if p < 0 {
			return 0, 0, ErrIncompleteCode
		}
		cur += p + 1
		if c == '\n' {
			break
		}
		if n > (math.MaxInt >> 1) {
			return 0, 0, ErrOverflow
		}
		n <<= 1
		if c == '\t' {
			n |= 1
		}
	}
	if neg {
		n *= -1
	}
	return n, cur, nil
}

func readLabel(code []byte) (string, int, error) {
	l := make([]byte, 0)
	read := 0
	for {
		c, p := findWhite(code[read:])
		if p < 0 {
			return "", 0, ErrIncompleteCode
		}
		read += p + 1
		if c == '\n' {
			return string(l), read, nil
		}
		l = append(l, c)
	}
}

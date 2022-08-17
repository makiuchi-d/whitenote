// Whitespace REPL binary
//
// Usage:
//   wspace <file>
//     Evaluate the file
//   wspace
//     Launch an interactive interpreter
//
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/makiuchi-d/whitenote/wspace"
)

func main() {
	if len(os.Args) >= 2 {
		evalFile(os.Args[1])
		return
	}
	interactive()
}

func evalFile(fname string) {
	code, err := os.ReadFile(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(-1)
	}

	vm := wspace.New()

	_, p, err := vm.Load(code)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s:%v: %+v\n", fname, p, err)
		os.Exit(-1)
	}

	err = vm.Run(context.Background(), os.Stdin, os.Stdout)
	if err != nil {
		op := vm.CurrentOpCode()
		fmt.Fprintf(os.Stderr, "%v:%v: %v: %+v\n", fname, op.Pos, op.Cmd, err)
		os.Exit(-1)
	}

	if !vm.Terminated {
		fmt.Fprintln(os.Stderr, "program is not terminated")
		os.Exit(-1)
	}
}

func interactive() {
	vm := wspace.New()

	rd := NewSwitchBufReader(os.Stdin, 2)
	ctx := context.Background()

	var code []byte
	for !vm.Terminated {

		rd.Switch(0)
		fmt.Printf("[%v]%s>", vm.Seg, visualize(code))
		c, err := rd.ReadLine()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(-1)
		}
		if c[0] == '%' {
			if string(c) == "%debug\n" {
				showVM(vm)
			}
			continue
		}

		code = append(code, c...)
		s, l, err := vm.Load(code)
		if err != nil && !errors.Is(err, wspace.ErrIncompleteCode) {
			fmt.Fprintf(os.Stderr, "%v:%v: %v\n", s, l, err)
			code = code[:0]
		} else {
			code = code[l:]
		}
		if l == 0 {
			continue
		}

		rd.Switch(1)
		err = vm.Run(ctx, rd, os.Stdout)
		if err != nil {
			op := vm.CurrentOpCode()
			fmt.Fprintf(os.Stderr, "%v:%v: %v: %v\n", op.Seg, op.Pos, op.Cmd, err)
			vm.PC = len(vm.Program)
			vm.Terminated = false
		}
	}
}

func visualize(code []byte) string {
	s := make([]byte, 0, len(code))
	for _, c := range code {
		switch c {
		case ' ':
			s = append(s, '.')
		case '\t':
			s = append(s, '_')
		case '\n':
			s = append(s, ',')
		}
	}
	return string(s)
}

func showVM(vm *wspace.VM) {
	fmt.Fprintln(os.Stderr, "program:")
	for _, op := range vm.Program {
		fmt.Fprintln(os.Stderr, "", op)
	}
	fmt.Fprintln(os.Stderr, "stack:", vm.Stack)
	fmt.Fprintln(os.Stderr, "heap:", vm.Heap)
}

type SwitchBufReader struct {
	rd   reader
	bufs []*bytes.Buffer
	sel  int
}

var _ wspace.InputReader = &SwitchBufReader{}

type reader interface {
	ReadBytes(delim byte) ([]byte, error)
}

func NewSwitchBufReader(rd io.Reader, n int) *SwitchBufReader {
	r, ok := rd.(reader)
	if !ok {
		r = bufio.NewReader(rd)
	}
	return &SwitchBufReader{
		rd:   r,
		bufs: make([]*bytes.Buffer, n),
		sel:  0,
	}
}

func (r *SwitchBufReader) Switch(n int) bool {
	if n < 0 || n >= len(r.bufs) {
		return false
	}
	r.sel = n
	return true
}

func (r *SwitchBufReader) Read(p []byte) (int, error) {
	return switchBufReadFn(r, func(buf *bytes.Buffer) (int, error) {
		return buf.Read(p)
	})
}

func (r *SwitchBufReader) ReadByte() (byte, error) {
	return switchBufReadFn(r, func(b *bytes.Buffer) (byte, error) {
		return b.ReadByte()
	})
}

func (r *SwitchBufReader) ReadLine() ([]byte, error) {
	return switchBufReadFn(r, func(b *bytes.Buffer) ([]byte, error) {
		return b.ReadBytes('\n')
	})
}

func switchBufReadFn[T interface{ int | byte | []byte }](r *SwitchBufReader, f func(*bytes.Buffer) (T, error)) (def T, _ error) {
	if buf := r.bufs[r.sel]; buf != nil {
		b, err := f(buf)
		if err == nil {
			return b, nil
		}
		if !errors.Is(err, io.EOF) {
			return b, err
		}
	}
	bs, err := r.rd.ReadBytes('\n')
	if err != nil {
		return def, err
	}
	buf := bytes.NewBuffer(bs)
	r.bufs[r.sel] = buf
	return f(buf)
}

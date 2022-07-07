package main

import (
	"context"
	"fmt"
	"os"

	"github.com/makiuchi-d/whitenote/wspace"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: wspace [file]")
	}
	fname := os.Args[1]

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

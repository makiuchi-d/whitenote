package wspace

import (
	"bytes"
	"reflect"
	"testing"
)

func TestReadNum(t *testing.T) {
	tests := map[string]struct {
		n int
		p int
		e error
	}{
		"aaa":        {0, 0, ErrIncompleteCode},
		" \t ":       {0, 0, ErrIncompleteCode},
		"\n":         {0, 1, nil},
		"aa \n":      {0, 4, nil},
		"aa a\n":     {0, 5, nil},
		"\tabc\n":    {0, 5, nil},
		" \t \t \n":  {10, 6, nil},
		"aa\t\t  \n": {-4, 7, nil},

		" \t                                                               \n": {0, 0, ErrOverflow},
	}
	for c, test := range tests {
		n, p, e := readNum([]byte(c))
		if e != test.e {
			t.Fatalf("%q => error=%v (%v, %v), wants %v", c, e, n, p, test.e)
		}

		if n != test.n || p != test.p {
			t.Fatalf("%q => (%v, %v), wants (%v, %v)", c, n, p, test.n, test.p)
		}
	}
}

func TestStackManipulation(t *testing.T) {
	tests := map[string]struct {
		op     OpCode
		stack  []int
		expect []int
	}{
		"Push":    {OpCode{Cmd: Push, Param: 1}, []int{}, []int{1}},
		"Dup":     {OpCode{Cmd: Dup}, []int{1, 2}, []int{1, 2, 2}},
		"Copy":    {OpCode{Cmd: Copy, Param: 3}, []int{1, 2, 3, 4, 5}, []int{1, 2, 3, 4, 5, 2}},
		"Swap":    {OpCode{Cmd: Swap}, []int{1, 2, 3, 4}, []int{1, 2, 4, 3}},
		"Discard": {OpCode{Cmd: Discard}, []int{1, 2, 3}, []int{1, 2}},
		"Slide":   {OpCode{Cmd: Slide, Param: 2}, []int{1, 2, 3, 4, 5}, []int{1, 2, 5}},
	}
	for k, test := range tests {
		vm := New()
		vm.Program = []OpCode{test.op}
		vm.Stack = test.stack

		err := vm.Step(nil, nil)
		if err != nil {
			t.Fatalf("%v: %v", k, err)
		}
		if !reflect.DeepEqual(vm.Stack, test.expect) {
			t.Fatalf("%v: Stack: %v, wants %v", k, vm.Stack, test.expect)
		}
		if vm.PC != 1 {
			t.Fatalf("%v: PC: %v, wants 1", k, vm.PC)
		}
	}
}

func TestArithmetic(t *testing.T) {
	tests := map[string]struct {
		op     OpCode
		stack  []int
		expect []int
	}{
		"Add": {OpCode{Cmd: Add}, []int{1, 2, 3}, []int{1, 5}},
		"Sub": {OpCode{Cmd: Sub}, []int{1, 2, 3}, []int{1, -1}},
		"Mul": {OpCode{Cmd: Mul}, []int{1, 2, 3}, []int{1, 6}},
		"Div": {OpCode{Cmd: Div}, []int{5, 7, 3}, []int{5, 2}},
		"Mod": {OpCode{Cmd: Mod}, []int{5, 7, 3}, []int{5, 1}},
	}
	for k, test := range tests {
		vm := New()
		vm.Program = []OpCode{test.op}
		vm.Stack = test.stack

		err := vm.Step(nil, nil)
		if err != nil {
			t.Fatalf("%v: %v", k, err)
		}
		if !reflect.DeepEqual(vm.Stack, test.expect) {
			t.Fatalf("%v: Stack: %v, wants %v", k, vm.Stack, test.expect)
		}
		if vm.PC != 1 {
			t.Fatalf("%v: PC: %v, wants 1", k, vm.PC)
		}
	}
}

func TestHeapAccess(t *testing.T) {
	addr := 10
	value := 12345

	vm := New()
	vm.Stack = []int{addr, addr, value}
	vm.Program = []OpCode{{Cmd: Store}, {Cmd: Retrieve}}

	err := vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if v, ok := vm.Heap[addr]; !ok || v != value {
		t.Fatalf("Store: Heap[%v]=%v (%v), wants %v", addr, v, ok, value)
	}
	if !reflect.DeepEqual(vm.Stack, []int{addr}) {
		t.Fatalf("Store: Stack=%v, wants %v", vm.Stack, []int{addr})
	}
	if vm.PC != 1 {
		t.Fatalf("Store: PC=%v, wants 1", vm.PC)
	}

	err = vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if !reflect.DeepEqual(vm.Stack, []int{value}) {
		t.Fatalf("Retrieve: Stack=%v, wants %v", vm.Stack, []int{value})
	}
	if vm.PC != 2 {
		t.Fatalf("Retrieve: PC=%v, wants 1", vm.PC)
	}
}

func TestFlowControll(t *testing.T) {
	vm := New()
	vm.Program = []OpCode{
		{Cmd: End},
		{Cmd: Ret},
		{Cmd: Jump, Param: " "},
		{Cmd: Call, Param: "\t"},
		{Cmd: JZero, Param: " "},
		{Cmd: JZero, Param: " "},
		{Cmd: JNeg, Param: " "},
		{Cmd: JNeg, Param: " "},
	}
	vm.Labels[" "] = 0
	vm.Labels["\t"] = 1

	// Jump
	vm.PC = 2
	err := vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("Jump: %v", err)
	}
	if vm.PC != 0 {
		t.Fatalf("Jump: PC=%v, wants %v", vm.PC, 0)
	}

	// Call, Ret
	vm.PC = 3
	err = vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if vm.PC != 1 {
		t.Fatalf("Call: PC=%v, wants %v", vm.PC, 1)
	}
	if !reflect.DeepEqual(vm.CallStack, []int{4}) {
		t.Fatalf("Call: CallStack=%v, wants %v", vm.CallStack, []int{4})
	}
	err = vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("Ret: %v", err)
	}
	if vm.PC != 4 {
		t.Fatalf("Ret: PC=%v, wants %v", vm.PC, 4)
	}
	if !reflect.DeepEqual(vm.CallStack, []int{}) {
		t.Fatalf("Ret: CallStack=%v, wants %v", vm.CallStack, []int{})
	}

	// JZero
	vm.Stack = []int{0, -1}
	vm.PC = 4
	err = vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("JZero: %v", err)
	}
	if vm.PC != 5 {
		t.Fatalf("JZero: PC=%v, wants 5", vm.PC)
	}
	if !reflect.DeepEqual(vm.Stack, []int{0}) {
		t.Fatalf("JZero: Stack=%v, wants %v", vm.Stack, []int{0})
	}
	err = vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("JZero: %v", err)
	}
	if vm.PC != 0 {
		t.Fatalf("JZero: PC=%v, wants 0", vm.PC)
	}
	if !reflect.DeepEqual(vm.Stack, []int{}) {
		t.Fatalf("JZero: Stack=%v, wants %v", vm.Stack, []int{})
	}

	// JNeg
	vm.Stack = []int{-1, 0}
	vm.PC = 6
	err = vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("JNeg: %v", err)
	}
	if vm.PC != 7 {
		t.Fatalf("JNeg: PC=%v, wants 7", vm.PC)
	}
	if !reflect.DeepEqual(vm.Stack, []int{-1}) {
		t.Fatalf("JNeg: Stack=%v, wants %v", vm.Stack, []int{-1})
	}
	err = vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("JNeg: %v", err)
	}
	if vm.PC != 0 {
		t.Fatalf("JNeg: PC=%v, wants 0", vm.PC)
	}
	if !reflect.DeepEqual(vm.Stack, []int{}) {
		t.Fatalf("JNeg: Stack=%v, wants %v", vm.Stack, []int{})
	}

	// End
	err = vm.Step(nil, nil)
	if err != nil {
		t.Fatalf("End: %v", err)
	}
	if !vm.Terminated {
		t.Fatalf("End: Terminated must be true")
	}
	if vm.PC != 0 {
		t.Fatalf("End: PC=%v, wants 0", vm.PC)
	}
}

func TestIO(t *testing.T) {
	vm := New()

	// input
	r := bytes.NewBufferString("a123\n")
	vm.Program = []OpCode{{Cmd: ReadChar}, {Cmd: ReadNum}}
	vm.Stack = []int{1, 0}

	err := vm.Step(r, nil)
	if err != nil {
		t.Fatalf("ReadChar: %v", err)
	}
	if v, ok := vm.Heap[0]; !ok || v != 'a' {
		t.Fatalf("ReadChar: Heap[0]=%v (%v), wants %v", v, ok, 'a')
	}
	if vm.PC != 1 {
		t.Fatalf("ReadChar: PC=%v, wants 1", vm.PC)
	}

	err = vm.Step(r, nil)
	if err != nil {
		t.Fatalf("ReadNum: %v", err)
	}
	if v, ok := vm.Heap[1]; !ok || v != 123 {
		t.Fatalf("ReadNum: Heap[1]=%v (%v), wants %v", v, ok, 'a')
	}
	if vm.PC != 2 {
		t.Fatalf("ReadChar: PC=%v, wants 2", vm.PC)
	}

	// output
	w := bytes.NewBuffer(nil)
	vm = New()
	vm.Program = []OpCode{{Cmd: WriteChar}, {Cmd: WriteNum}}
	vm.Stack = []int{123, 'c'}

	err = vm.Step(nil, w)
	if err != nil {
		t.Fatalf("WriteChar: %v", err)
	}
	if s := string(w.Bytes()); s != "c" {
		t.Fatalf("WriteChar: out=%q, wants %q", s, "c")
	}
	if !reflect.DeepEqual(vm.Stack, []int{123}) {
		t.Fatalf("WriteChar: Stack=%v, wants %v", vm.Stack, []int{123})
	}
	if vm.PC != 1 {
		t.Fatalf("WriteChar: PC=%v, wants 1", vm.PC)
	}

	err = vm.Step(nil, w)
	if err != nil {
		t.Fatalf("WriteNum: %v", err)
	}
	if s := string(w.Bytes()); s != "c123" {
		t.Fatalf("WriteNum: out=%q, wants %q", s, "c123")
	}
	if !reflect.DeepEqual(vm.Stack, []int{}) {
		t.Fatalf("WriteNum: Stack=%v, wants %v", vm.Stack, []int{})
	}
	if vm.PC != 2 {
		t.Fatalf("WriteNum: PC=%v, wants 2", vm.PC)
	}
}

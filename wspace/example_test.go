package wspace_test

import (
	"context"
	"os"

	"github.com/makiuchi-d/whitenote/wspace"
)

func ExampleVM() {
	code := "   \t  \t   \n\t\n     \t\t  \t \t\n\t\n     \t\t \t\t  \n \n" +
		" \t\n  \t\n     \t\t \t\t\t\t\n\t\n     \t    \t\n\t\n  \n\n\n"

	vm := wspace.New()
	vm.Load([]byte(code))
	vm.Run(context.Background(), os.Stdin, os.Stdout)
	// Output: Hello!
}

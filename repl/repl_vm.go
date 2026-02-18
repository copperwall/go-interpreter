package repl

import (
	"bufio"
	"fmt"
	"io"
	"monkey/compiler"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
	"monkey/vm"
)

func StartVMRepl(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalsSize)
	symbolTable := compiler.NewSymbolTable()

	for {
		fmt.Fprintf(out, PROMPT)
		scanned := scanner.Scan()

		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()

		errs := p.Errors()

		if len(errs) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		// TODO: Repl not working with function let statements
		// when calling because
		// constants in repl is being reset to empty slice.
		// Does this mean the repl is missing an instruction to load in a
		// global set in the previous line?
		fmt.Println("constants")
		fmt.Println(constants)

		c := compiler.NewWithState(symbolTable, constants)
		err := c.Compile(program)

		if err != nil {
			fmt.Fprintf(out, "Woops! Compilation failed:\n %s\n", err)
			continue
		}

		machine := vm.NewWithGlobalsStore(c.Bytecode(), globals)
		err = machine.Run()

		if err != nil {
			fmt.Fprintf(out, "Woops! Executing bytecode failed:\n %s\n", err)
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		io.WriteString(out, lastPopped.Inspect())
		io.WriteString(out, "\n")
	}

}

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

		c := compiler.NewWithState(symbolTable, constants)
		err := c.Compile(program)

		if err != nil {
			fmt.Fprintf(out, "Woops! Compilation failed:\n %s\n", err)
			continue
		}

		code := c.Bytecode()
		constants = code.Constants

		machine := vm.NewWithGlobalsStore(code, globals)
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

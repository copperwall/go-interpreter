package run

import (
	"fmt"
	"io"
	"monkey/compiler"
	"monkey/lexer"
	"monkey/parser"
	"monkey/vm"
	"os"
)

func RunProgramFromFile(filename string) {
	text, err := os.ReadFile(filename)

	if err != nil {
		panic("Failed to read file: " + err.Error())
	}

	l := lexer.New(string(text))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		printParserErrors(os.Stderr, p.Errors())
		return
	}

	c := compiler.New()
	c.Compile(program)

	bytecode := c.Bytecode()

	for _, constant := range bytecode.Constants {
		fmt.Println(constant.Inspect())
	}
	// fmt.Println(bytecode.Instructions)

	v := vm.New(bytecode)
	v.Run()
	fmt.Println(v.LastPoppedStackElem().Inspect())
}

func printParserErrors(out io.Writer, errors []string) {
	for _, error := range errors {
		io.WriteString(out, "\t"+error+"\t")
	}
}

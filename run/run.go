package run

import (
	"fmt"
	"io"
	"monkey/evaluator"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
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

	evaluated := evaluator.Eval(program, object.NewEnvironment())
	fmt.Println(evaluated.Inspect())
}

func printParserErrors(out io.Writer, errors []string) {
	for _, error := range errors {
		io.WriteString(out, "\t"+error+"\t")
	}
}

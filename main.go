package main

import (
	"fmt"
	"monkey/repl"
	"monkey/run"
	"os"
	"os/user"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		replMode()
	} else {
		run.RunProgramFromFile(args[0])
	}
}

func replMode() {
	user, err := user.Current()

	if err != nil {
		panic(err)
	}

	fmt.Printf("Hello %s! This is the Monkey programming language!\n", user.Username)
	fmt.Printf("Feel free to type in commands\n")
	repl.Start(os.Stdin, os.Stdout)
}

package compiler

import (
	"fmt"
	"monkey/ast"
	"monkey/code"
	"monkey/object"
)

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

type Compiler struct {
	instructions        code.Instructions
	constants           []object.Object
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

func New() *Compiler {
	return &Compiler{
		instructions:        code.Instructions{},
		constants:           []object.Object{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)

			if err != nil {
				return err
			}
		}
	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(code.OpPop)
	case *ast.InfixExpression:
		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}

			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			c.emit(code.OpGreaterThan)
			return nil
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		case ">":
			c.emit(code.OpGreaterThan)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))
	case *ast.PrefixExpression:
		err := c.Compile(node.Right)

		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
			// something
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown operator: %s", node.Operator)
		}
	case *ast.IfExpression:
		// Compile condition to generate that info
		err := c.Compile(node.Condition)

		if err != nil {
			return err
		}

		// Dummy value because we don't know where we'll jump to yet
		jumpNotTruthPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Consequence)

		if err != nil {
			return err
		}

		// This is necessary for the last expression value of the consequence block to not get popped
		// Necessary for syntax like `let thing = if (true) { 1; 2; } (thing should be 2)
		if c.lastInstructionIsPop() {
			c.removeLastPop()
		}

		jumpPos := c.emit(code.OpJump, 9999)
		// Update JMPNotTruthy to point to end of consequence instructions
		endOfConsequencePos := len(c.instructions)
		c.changeOperand(jumpNotTruthPos, endOfConsequencePos)
		if node.Alternative == nil {
			c.emit(code.OpNull)

		} else {
			// Update JMPNotTruthy to point to end of consequence instructions (after OpJump after consequence)
			err := c.Compile(node.Alternative)

			if err != nil {
				return err
			}

			// This is necessary for the last expression value of the consequence block to not get popped
			// Necessary for syntax like `let thing = if (true) { 1; 2; } (thing should be 2)
			if c.lastInstructionIsPop() {
				c.removeLastPop()
			}
		}
		endOfAlternativePos := len(c.instructions)
		c.changeOperand(jumpPos, endOfAlternativePos)

		// If no alternative don't add a non-conditional jump
	case *ast.BlockStatement:
		// Compile all statements
		for _, v := range node.Statements {
			err := c.Compile(v)
			if err != nil {
				return err
			}
		}

	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	}

	return nil
}

// append constant and return the index
func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) lastInstructionIsPop() bool {
	return c.lastInstruction.Opcode == code.OpPop
}

func (c *Compiler) removeLastPop() {
	// Reduce slice to not include last instruction's (OpPop) position
	c.instructions = c.instructions[:c.lastInstruction.Position]
	// Set last instruction to 2nd to last
	c.lastInstruction = c.previousInstruction
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	c.previousInstruction = c.lastInstruction
	c.lastInstruction = EmittedInstruction{
		Opcode:   op,
		Position: pos,
	}
}

// Replaces bytes length of newInstruction at pos
func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	for i := 0; i < len(newInstruction); i++ {
		c.instructions[pos+i] = newInstruction[i]
	}
}

// Create a new instruction and place it using replaceInstruction
func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.instructions[opPos])
	ins := code.Make(op, operand)

	c.replaceInstruction(opPos, ins)
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.instructions)

	c.instructions = append(c.instructions, ins...)
	return posNewInstruction
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

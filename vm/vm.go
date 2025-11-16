package vm

import (
	"fmt"
	"monkey/code"
	"monkey/compiler"
	"monkey/object"
)

const StackSize = 2048

type VM struct {
	constants    []object.Object
	instructions code.Instructions
	stack        []object.Object
	sp           int // This points to the next value, the top value in the stack is always at sp - 1.
}

func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		instructions: bytecode.Instructions,
		constants:    bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,
	}
}

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}

	return vm.stack[vm.sp-1]
}

func (vm *VM) Run() error {
	// ip is instruction pointer
	// it starts at the beginning an goes until there are no instructions left.
	// vm.instructions is a []byte, meaning we need to parse instructions correctly
	// otherwise we'll end up at the beginning of a loop on a byte that isn't an opcode.
	for ip := 0; ip < len(vm.instructions); ip++ {
		// Fetch the instruction
		op := code.Opcode(vm.instructions[ip])

		// Decode the opcode
		switch op {
		case code.OpConstant:
			// If it's a constant, read the next two bytes after the ip
			constIndex := code.ReadUint16(vm.instructions[ip+1:])
			// increment the ip past the two bytes we read
			ip += 2

			// The operand in a OpConstant instruction is an index into the vm's constants table,
			// not the constant value itself.
			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vm *VM) push(obj object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow, oh no")
	}

	vm.stack[vm.sp] = obj
	vm.sp++

	return nil
}

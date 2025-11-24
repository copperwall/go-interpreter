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
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}
		case code.OpPop:
			vm.pop()
		}
	}

	return nil
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	if leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ {
		return vm.executeBinaryIntegerOperation(op, left, right)
	}

	return fmt.Errorf("Unsupported types for binary operation: %s %s", leftType, rightType)
}

func (vm *VM) executeBinaryIntegerOperation(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	var result int64
	switch op {
	case code.OpAdd:
		result = leftValue + rightValue
	case code.OpSub:
		result = leftValue - rightValue
	case code.OpMul:
		result = leftValue * rightValue
	case code.OpDiv:
		result = leftValue / rightValue
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

	fmt.Printf("Binary operation: %d %d %d, %d\n", leftValue, op, rightValue, result)
	return vm.push(&object.Integer{Value: result})
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) push(obj object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow, oh no")
	}

	vm.stack[vm.sp] = obj
	vm.sp++

	return nil
}

// Kind of seems like we should have an error case here
func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

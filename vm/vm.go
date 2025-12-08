package vm

import (
	"fmt"
	"monkey/code"
	"monkey/compiler"
	"monkey/object"
)

const StackSize = 2048
const GlobalsSize = 65536

// Global boolean objects
var True = &object.Boolean{
	Value: true,
}
var False = &object.Boolean{
	Value: false,
}
var Null = &object.Null{}

type VM struct {
	constants    []object.Object
	instructions code.Instructions
	stack        []object.Object
	sp           int // This points to the next value, the top value in the stack is always at sp - 1.
	globals      []object.Object
}

func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		instructions: bytecode.Instructions,
		constants:    bytecode.Constants,

		stack:   make([]object.Object, StackSize),
		sp:      0,
		globals: make([]object.Object, GlobalsSize),
	}
}

func NewWithGlobalsStore(bytecode *compiler.Bytecode, s []object.Object) *VM {
	vm := New(bytecode)
	vm.globals = s
	return vm
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
		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}
		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}
		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}
		case code.OpBang:
			err := vm.executeBangOperation()

			if err != nil {
				return err
			}
		case code.OpMinus:
			err := vm.executeMinusOperation()
			if err != nil {
				return err
			}
		case code.OpJump:
			// Read next uint16 after opcode (current ip)
			pos := int(code.ReadUint16(vm.instructions[ip+1:]))
			ip = pos - 1
		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(vm.instructions[ip+1:]))
			ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				ip = pos - 1
			}
		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}
		case code.OpSetGlobal:
			index := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2

			vm.globals[index] = vm.pop()

		case code.OpGetGlobal:
			// Read index off of instruction
			index := int(code.ReadUint16(vm.instructions[ip+1:]))

			ip += 2
			err := vm.push(vm.globals[index])
			if err != nil {
				return err
			}

		case code.OpArray:
			size := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2
			arr := make([]object.Object, size)

			// These are gonna be right to left
			// Slightly different impl than book, but tests pass
			// Starting from size so that we don't have to change
			// size from uint16 to get to < 0.
			for i := size; i > 0; i-- {
				arr[i-1] = vm.pop()
			}

			arrObj := &object.Array{Elements: arr}
			err := vm.push(arrObj)
			if err != nil {
				return err
			}

		case code.OpHash:
			size := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2

			pairs := make(map[object.HashKey]object.HashPair)

			// size is amount of pops to do, not pairs
			for i := uint16(0); i < size; i += 2 {
				val := vm.pop()
				key := vm.pop()

				pair := object.HashPair{Key: key, Value: val}

				hashKey, ok := key.(object.Hashable)

				if !ok {
					return fmt.Errorf("Value not hashable as key: %s", key.Type())
				}

				pairs[hashKey.HashKey()] = pair
			}

			err := vm.push(&object.Hash{Pairs: pairs})

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

	switch {
	case leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ:
		return vm.executeBinaryIntegerOperation(op, left, right)
	case leftType == object.STRING_OBJ && rightType == object.STRING_OBJ:
		return vm.executeBinaryStringOperation(op, left, right)
	default:
		return fmt.Errorf("Unsupported types for binary operation: %s %s", leftType, rightType)
	}

}

// Everything not False is True with ! (i.e. !5 is False, !true is False)
func (vm *VM) executeBangOperation() error {
	right := vm.pop()
	if right == False || right == Null {
		return vm.push(True)
	}

	return vm.push(False)
}

func (vm *VM) executeMinusOperation() error {
	right := vm.pop()
	if right.Type() != object.INTEGER_OBJ {
		return fmt.Errorf("Minus operator only works on integers, got %s", right.Type())
	}

	value := right.(*object.Integer).Value

	return vm.push(&object.Integer{Value: -value})
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

	return vm.push(&object.Integer{Value: result})
}

func (vm *VM) executeBinaryStringOperation(op code.Opcode, left object.Object, right object.Object) error {
	if op != code.OpAdd {
		return fmt.Errorf("unknown string operator: %d", op)
	}

	rightStr := right.(*object.String).Value
	leftStr := left.(*object.String).Value

	newStr := &object.String{Value: leftStr + rightStr}

	return vm.push(newStr)
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	if right.Type() == object.INTEGER_OBJ && left.Type() == object.INTEGER_OBJ {
		return vm.executeIntegerComparison(op, left, right)
	}

	// If right and left are integers, do eq, noteq, gt
	switch op {
	case code.OpEqual:
		// We don't need to unwrap these because booleans are singleton values.
		return vm.push(nativeBoolToBooleanObject(right == left))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(right != left))
	default:
		return fmt.Errorf("Unsupported operator: %d (%s %s)", op, left.Type(), right.Type())
	}
}

func (vm *VM) executeIntegerComparison(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(leftValue == rightValue))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(leftValue != rightValue))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBooleanObject(leftValue > rightValue))
	default:
		return fmt.Errorf("Unknown operator: %d", op)
	}
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

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return True
	}

	return False
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value
	case *object.Null:
		return false
	default:
		return true
	}
}

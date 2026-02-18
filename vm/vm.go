package vm

import (
	"fmt"
	"monkey/code"
	"monkey/compiler"
	"monkey/object"
)

const StackSize = 2048
const GlobalsSize = 65536

const MaxFrames = 1024

// Global boolean objects
var True = &object.Boolean{
	Value: true,
}
var False = &object.Boolean{
	Value: false,
}
var Null = &object.Null{}

type VM struct {
	constants []object.Object
	stack     []object.Object
	sp        int // This points to the next value, the top value in the stack is always at sp - 1.
	globals   []object.Object

	frames      []*Frame
	framesIndex int
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainFrame := NewFrame(mainFn)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	return &VM{
		constants: bytecode.Constants,

		stack:   make([]object.Object, StackSize),
		sp:      0,
		globals: make([]object.Object, GlobalsSize),

		frames:      frames,
		framesIndex: 1,
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
	var ip int
	var ins code.Instructions
	var op code.Opcode

	fmt.Printf("ins (%+v)\n", ins)

	// ip is instruction pointer
	// it starts at the beginning an goes until there are no instructions left.
	// vm.instructions is a []byte, meaning we need to parse instructions correctly
	// otherwise we'll end up at the beginning of a loop on a byte that isn't an opcode.
	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		op = code.Opcode(ins[ip])

		fmt.Println(op)
		fmt.Println(vm.constants)
		// Decode the opcode
		switch op {
		case code.OpConstant:
			// If it's a constant, read the next two bytes after the ip
			constIndex := code.ReadUint16(ins[ip+1:])
			// increment the ip past the two bytes we read
			vm.currentFrame().ip += 2

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
			pos := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip = pos - 1
		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}
		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}
		case code.OpSetGlobal:
			index := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			vm.globals[index] = vm.pop()

		case code.OpGetGlobal:
			// Read index off of instruction
			index := int(code.ReadUint16(ins[ip+1:]))

			vm.currentFrame().ip += 2
			err := vm.push(vm.globals[index])
			if err != nil {
				return err
			}

		case code.OpArray:
			size := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2
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
			size := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			pairs := make(map[object.HashKey]object.HashPair)

			// size is amount of pops to do, not pairs
			// This means that printing a hash in the runtime will
			// show the pairs in a different order, but they don't support
			// iteration so it doesn't matter right now
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
		case code.OpIndex:
			err := vm.executeIndexOperation()
			if err != nil {
				return err
			}
		case code.OpReturnValue:
			returnValue := vm.pop()

			vm.popFrame()
			vm.pop()

			err := vm.push(returnValue)

			if err != nil {
				return err
			}
		case code.OpCall:
			// Take value off of stack
			// Execute the instructions
			// Place value from the function back on the stack
			fn, ok := vm.stack[vm.sp-1].(*object.CompiledFunction)

			if !ok {
				return fmt.Errorf("calling non-function")
			}

			frame := NewFrame(fn)
			vm.pushFrame(frame)

		case code.OpPop:
			vm.pop()
		}

	}

	return nil
}

func (vm *VM) executeIndexOperation() error {
	index := vm.pop()
	container := vm.pop()

	switch {

	case container.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndexOperation(container, index)
	case container.Type() == object.HASH_OBJ:
		return vm.executeHashIndexOperation(container, index)
	default:
		return fmt.Errorf("Unknown operands for index, %q and %q", container.Type(), index.Type())
	}
}

func (vm *VM) executeHashIndexOperation(hashObj object.Object, index object.Object) error {
	hash, ok := hashObj.(*object.Hash)

	if !ok {
		return fmt.Errorf("Value is not hash, %q", hashObj.Inspect())
	}

	idx, ok := index.(object.Hashable)

	if !ok {
		return fmt.Errorf("Index is not hashable, %q", index.Inspect())
	}

	pair, ok := hash.Pairs[idx.HashKey()]

	if !ok {
		return vm.push(Null)
	} else {
		return vm.push(pair.Value)
	}
}

func (vm *VM) executeArrayIndexOperation(array object.Object, index object.Object) error {
	arr, ok := array.(*object.Array)

	if !ok {
		return fmt.Errorf("Value is not array, %q", array.Inspect())
	}

	idx, ok := index.(*object.Integer)

	if !ok {
		return fmt.Errorf("Index is not number, %q", index.Inspect())
	}

	if idx.Value < 0 || int(idx.Value) >= len(arr.Elements) {
		return vm.push(Null)
	} else {
		return vm.push(arr.Elements[idx.Value])
	}
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

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
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

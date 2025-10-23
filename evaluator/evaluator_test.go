package evaluator

import (
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
	"testing"
)

func TestEvalIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"-10", -10},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"-50 + 100 + -50", 0},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"20 + 2 * -10", 0},
		{"50 / 2 * 2 + 10", 60},
		{"2 * (5 + 10)", 30},
		{"3 * 3 * 3 + 10", 37},
		{"3 * (3 * 3) + 10", 37},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}

}

func testEval(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	return Eval(program, object.NewEnvironment())
}

func testIntegerObject(t *testing.T, obj object.Object, expected int64) bool {
	result, ok := obj.(*object.Integer)

	if !ok {
		t.Errorf("object is not Integer. got %T (%+v)", obj, obj)
		return false
	}

	if result.Value != expected {
		t.Errorf("object has wrong value. expected %d, but got %d", expected, result.Value)
		return false
	}

	return true
}

func TestEvalBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"false != true", true},
		{"(1 < 2) == true", true},
		{"(1 < 2) == false", false},
		{"(1 > 2) == true", false},
		{"(1 > 2) == false", true},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestBangOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"!false", true},
		{"!true", false},
		{"!5", false},
		{"!!true", true},
		{"!!false", false},
		{"!!5", true},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestIfElseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"if (true) { 10 }", 10},
		{"if (false) { 10 }", nil},
		{"if (1) { 10 }", 10},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 > 2) { 10 }", nil},
		{"if (1 > 2) { 10 } else { 20 }", 20},
		{"if (1 < 2) { 10 } else { 20 }", 10},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		integer, ok := tt.expected.(int)

		if ok {
			testIntegerObject(t, evaluated, int64(integer))
		} else {
			testNullObject(t, evaluated)
		}
	}
}

func TestReturnStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"return 10;", 10},
		{"return 10; 9;", 10},
		{"return 2 * 5; 9;", 10},
		{"9; return 2 * 5; 9;", 10},
		{`
			if (10 > 1) {
				if (10 > 1) {
					return 10;
				}
				return 1;
			}
		`, 10},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		input           string
		expectedMessage string
	}{
		// Integer boolean mismatch
		{"5 + true;",
			"type mismatch: INTEGER + BOOLEAN"},
		// Stop execution after error
		{"5 + true; 5",
			"type mismatch: INTEGER + BOOLEAN"},
		// Unary boolean mismatch
		{"-true",
			"unknown operator: -BOOLEAN"},
		// Boolean add
		{"true + false;",
			"unknown operator: BOOLEAN + BOOLEAN"},
		// Boolean add in between expressions
		{"4; true + false; 10;",
			"unknown operator: BOOLEAN + BOOLEAN"},
		// Boolean add in block statement
		{"if (10 > 1) { true + false }",
			"unknown operator: BOOLEAN + BOOLEAN"},
		// Boolean add in nested block statement
		{`
			if (10 > 1) {
				if (10 > 1) {
					return true + false
				}
				return 1;
			}
		`,
			"unknown operator: BOOLEAN + BOOLEAN"},
		// Evaluate unknown identifier
		{"a;", "identifier not found: \"a\""},
		{`"Hello" - "World"`, "unknown operator: STRING - STRING"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)

		errObj, ok := evaluated.(*object.Error)

		if !ok {
			t.Errorf("Expected error, got %T(%+v)", evaluated, evaluated)
			continue
		}

		if errObj.Message != tt.expectedMessage {
			t.Errorf("Expected error to have message %q, got %q", tt.expectedMessage, errObj.Message)
		}
	}
}

func TestLetStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let a = 5; a;", 5},
		{"let a = 5 * 5; a;", 25},
		{"let a = 5; let b = a; b;", 5},
		{"let a = 5; let b = a; let c = a + b + 5; c;", 15},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(tt.input), tt.expected)
	}
}

func TestFunctionObject(t *testing.T) {
	input := `fn(x) { x + 2; };`

	evaluated := testEval(input)

	fn, ok := evaluated.(*object.FunctionValue)

	if !ok {
		t.Fatalf("object is not a function, got %T", evaluated)
	}

	if len(fn.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(fn.Parameters))
	}

	if fn.Parameters[0].String() != "x" {
		t.Fatalf("Expected parameter name to be 'x', got %q", fn.Parameters[0])
	}

	expectedBody := "(x + 2)"

	if fn.Body.String() != expectedBody {
		t.Fatalf("body is not %q. got = %q", expectedBody, fn.Body.String())
	}
}

func TestFunctionCalls(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		// Implicit return
		{"let identity = fn(x) { x }; identity(5);", 5},
		// Explicit return
		{"let identity = fn(x) { return x }; identity(5);", 5},
		{"let double = fn(x) { x * 2 }; double(5);", 10},
		{"let add = fn(x,y) { x + y }; add(5, 5);", 10},
		// Nested calls
		{"let add = fn(x,y) { x + y }; add(5 + 5, add(5, 5));", 20},
		// Call function expression
		{"fn(x) { x; }(5)", 5},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestClosures(t *testing.T) {
	input := `
		let newAdder = fn(x) {
			fn(y) { x + y };
		};

		let addTwo = newAdder(2);
		addTwo(2);
	`

	testIntegerObject(t, testEval(input), 4)
}

func TestStringLiteral(t *testing.T) {
	input := `"Hello World!"`

	evaluated := testEval(input)

	str, ok := evaluated.(*object.String)

	if !ok {
		t.Fatalf("object is not a string, got %T(%+v)", evaluated, evaluated)
	}

	if str.Value != "Hello World!" {
		t.Errorf("String has wrong value. got %q", str.Value)
	}
}

func TestStringConcatenation(t *testing.T) {
	input := `"Hello" + " " + "World!"`

	evaluated := testEval(input)

	str, ok := evaluated.(*object.String)

	if !ok {
		t.Fatalf("Expected result to be string, got %T (%+v)", evaluated, evaluated)
	}

	if str.Value != "Hello World!" {
		t.Fatalf("Expected string to be Hello World, got %q", str.Value)
	}
}

func TestStringComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`"hello" == "hello"`, true},
		{`"hello" == "hellooo"`, false},
		{`"foo" != "bar"`, true},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{`len("")`, 0},
		{`len("four")`, 4},
		{`len("hello world")`, 11},
		{`len(1)`, "argument to `len` not supported, got INTEGER"},
		{`len("one", "two")`, "wrong number of arguments. got=2, want=1"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)

		switch expected := tt.expected.(type) {
		case int:
			testIntegerObject(t, evaluated, int64(expected))
		case string:
			errObj, ok := evaluated.(*object.Error)
			if !ok {
				t.Errorf("object is not Error. got %T (%+v)", evaluated, evaluated)
				continue
			}

			if errObj.Message != expected {
				t.Errorf("wrong error message. Expected %q, got %q", expected, errObj.Message)
			}
		}
	}
}

func TestArrayLiterals(t *testing.T) {
	input := "[1 + 2, 10, true]"

	evaluated := testEval(input)

	array, ok := evaluated.(*object.Array)

	if !ok {
		t.Fatalf("Expected array object, got %T (%+v)", evaluated, evaluated)
	}

	if len(array.Elements) != 3 {
		t.Fatalf("Expected three elements, got %d", len(array.Elements))
	}

	testIntegerObject(t, array.Elements[0], 3)
	testIntegerObject(t, array.Elements[1], 10)
	testBooleanObject(t, array.Elements[2], true)
}

func TestArrayIndexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{
			"[1,2,3][0]",
			1,
		},
		{
			"[1,2,3][1]",
			2,
		},
		{
			"[1,2,3][2]",
			3,
		},
		{
			"[1,2,3][1 + 1]",
			3,
		},
		{
			"let x = [1,2,3]; x[2]",
			3,
		},
		{
			"let x = [1,2,3]; x[0] + x[1] + x[2];",
			6,
		},
		{
			"let i = 0; [1][i]",
			1,
		},
		{
			// Invalid access over
			"[1,2,3][3]",
			nil,
		},
		{
			// Invalid access under
			"[1,2,3][-1]",
			nil,
		},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)

		integer, ok := tt.expected.(int)

		if ok {
			// Check for integer value
			testIntegerObject(t, evaluated, int64(integer))
		} else {
			// check for nil
			if evaluated != NULL {
				testNullObject(t, evaluated)
			}
		}
	}
}

func TestHashLiterals(t *testing.T) {
	input := `let two = "two";
		{
			"one": 10 - 9,
			two: 1 + 1,
			"thr" + "ee": 6 / 2,
			4: 4,
			true: 5,
			false: 6
		}`

	evaluated := testEval(input)
	result, ok := evaluated.(*object.Hash)

	if !ok {
		t.Fatalf("Expected hash, got %T (%+v)", evaluated, evaluated)
	}

	expected := map[object.HashKey]int64{
		(&object.String{Value: "one"}).HashKey():   1,
		(&object.String{Value: "two"}).HashKey():   2,
		(&object.String{Value: "three"}).HashKey(): 3,
		(&object.Integer{Value: 4}).HashKey():      4,
		TRUE.HashKey():                             5,
		FALSE.HashKey():                            6,
	}

	if len(result.Pairs) != len(expected) {
		t.Fatalf("Expected %d pairs, got %d", len(expected), len(result.Pairs))
	}

	for expectedKey, expectedValue := range expected {
		pair, ok := result.Pairs[expectedKey]

		if !ok {
			t.Errorf("no pair for key in Pairs")
		}

		testIntegerObject(t, pair.Value, expectedValue)
	}
}

func testNullObject(t *testing.T, evaluated object.Object) bool {
	if evaluated != NULL {
		t.Errorf("object is not NULL. Got %T (%+v)", evaluated, evaluated)
		return false
	}
	return true
}

func testBooleanObject(t *testing.T, obj object.Object, expected bool) bool {
	result, ok := obj.(*object.Boolean)

	if !ok {
		t.Errorf("Expected object to be Boolean, got %T", obj)
		return false
	}

	if result.Value != expected {
		t.Errorf("Expected value to be %t, got %t", expected, result.Value)
		return false

	}

	return true
}

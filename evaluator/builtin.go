package evaluator

import (
	"monkey/object"
)

var builtins = map[string]*object.Builtin{
	"first": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("Expected one argument, got %d, want 1", len(args))
			}

			if args[0].Type() != object.ARRAY_OBJ {
				return newError("Only supports array, got %s", args[0].Type())
			}

			arr := args[0].(*object.Array)

			if len(arr.Elements) == 0 {
				return NULL
			}

			return arr.Elements[0]
		},
	},
	"last": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("Expected one argument, got %d, want 1", len(args))
			}

			if args[0].Type() != object.ARRAY_OBJ {
				return newError("Only supports array, got %s", args[0].Type())
			}

			arr := args[0].(*object.Array)

			if len(arr.Elements) == 0 {
				return NULL
			}

			return arr.Elements[len(arr.Elements)-1]
		},
	},
	"rest": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("Expected one argument, got %d, want 1", len(args))
			}

			if args[0].Type() != object.ARRAY_OBJ {
				return newError("Only supports array, got %s", args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)

			if length == 0 {
				return NULL
			}

			newElements := make([]object.Object, length-1, length-1)
			copy(newElements, arr.Elements[1:length])
			return &object.Array{Elements: newElements}
		},
	},
	"push": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("Expected one argument, got %d, want 2", len(args))
			}

			if args[0].Type() != object.ARRAY_OBJ {
				return newError("Only supports array, got %s", args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)

			newElements := make([]object.Object, length+1)
			copy(newElements, arr.Elements)

			newElements[length] = args[1]
			return &object.Array{Elements: newElements}
		},
	},
	"len": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch arg := args[0].(type) {
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			default:
				return newError("argument to `len` not supported, got %s", args[0].Type())

			}
		},
	},
}

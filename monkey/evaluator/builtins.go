// builtins.go は Monkey言語の組み込み関数を定義する。
// これらの関数はユーザーが定義しなくても最初から使える。
// 4章で追加。
//
// 組み込み関数一覧:
// - len: 文字列の長さまたは配列の要素数を返す
// - puts: 引数を標準出力に出力する（デバッグ用）
// - first: 配列の最初の要素を返す
// - last: 配列の最後の要素を返す
// - rest: 配列の最初の要素を除いた新しい配列を返す
// - push: 配列の末尾に要素を追加した新しい配列を返す（元の配列は変更しない）
package evaluator

import (
	"fmt"
	"monkey/object"
)

// builtins は組み込み関数名からBuiltinオブジェクトへのマップ。
// evalIdentifier から参照される。
var builtins = map[string]*object.Builtin{
	// len は文字列の長さまたは配列の要素数を返す。
	// 引数は1つだけ受け取り、STRING または ARRAY 型のみ対応。
	"len": {Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		switch arg := args[0].(type) {
		case *object.Array:
			return &object.Integer{Value: int64(len(arg.Elements))}
		case *object.String:
			return &object.Integer{Value: int64(len(arg.Value))}
		default:
			return newError("argument to `len` not supported, got %s",
				args[0].Type())
		}
	},
	},

	// puts は引数を標準出力に出力する。デバッグ用。
	// 常にNULLを返す。
	"puts": {
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}

			return NULL
		},
	},

	// first は配列の最初の要素を返す。
	// 空配列の場合はNULLを返す。
	"first": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `first` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			if len(arr.Elements) > 0 {
				return arr.Elements[0]
			}

			return NULL
		},
	},

	// last は配列の最後の要素を返す。
	// 空配列の場合はNULLを返す。
	"last": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `last` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				return arr.Elements[length-1]
			}

			return NULL
		},
	},

	// rest は配列の最初の要素を除いた新しい配列を返す。
	// 元の配列は変更しない（イミュータブル）。
	// 空配列の場合はNULLを返す。
	"rest": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `rest` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				newElements := make([]object.Object, length-1, length-1)
				copy(newElements, arr.Elements[1:length])
				return &object.Array{Elements: newElements}
			}

			return NULL
		},
	},

	// push は配列の末尾に要素を追加した新しい配列を返す。
	// 元の配列は変更しない（イミュータブル）。
	// 関数型プログラミングのスタイルで、元のデータを壊さない。
	"push": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `push` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)

			newElements := make([]object.Object, length+1, length+1)
			copy(newElements, arr.Elements)
			newElements[length] = args[1]

			return &object.Array{Elements: newElements}
		},
	},
}

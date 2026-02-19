// Package evaluator は Monkey言語のTree-walking評価器を実装するパッケージ。
// ASTを再帰的にたどりながら（tree-walking）、各ノードを評価して
// object.Object としての結果を返す。
//
// これはインタプリタの核心部分であり、パーサーが構築したASTを
// 実際に「実行」する役割を担う。
package evaluator

import (
	"fmt"
	"monkey/ast"
	"monkey/object"
)

// シングルトンオブジェクト。
// true, false, null は常に同じオブジェクトを使い回すことで、
// メモリ効率を上げ、ポインタ比較で等値判定できるようにする。
var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

// Eval はASTノードを評価してオブジェクトを返す、評価器のメイン関数。
// ノードの型に応じたswitch文で処理を分岐する。
// 全ての評価はこの関数を通じて再帰的に行われる。
func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {

	// === 文（Statements）===

	// Program: プログラム全体を評価する
	case *ast.Program:
		return evalProgram(node, env)

	// BlockStatement: ブロック内の文を順に評価する
	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	// ExpressionStatement: 式文の内部の式を評価する
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	// ReturnStatement: 戻り値を評価し、ReturnValueでラップする
	// これにより呼び出しスタックを巻き戻せる
	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	// LetStatement: 右辺を評価し、環境に変数を束縛する
	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)

	// === 式（Expressions）===

	// IntegerLiteral: 整数リテラルをIntegerオブジェクトに変換
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	// Boolean: 真偽値をシングルトンのBooleanオブジェクトに変換
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	// PrefixExpression: 前置演算子式を評価する（!, -）
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	// InfixExpression: 中置演算子式を評価する（+, -, *, /, ==, != など）
	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalInfixExpression(node.Operator, left, right)

	// IfExpression: 条件式を評価し、真偽に応じたブロックを実行
	case *ast.IfExpression:
		return evalIfExpression(node, env)

	// Identifier: 環境から変数の値を取得する
	case *ast.Identifier:
		return evalIdentifier(node, env)

	// FunctionLiteral: 関数オブジェクトを生成する（クロージャ）
	// 定義時の環境を保持することがクロージャのポイント
	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Env: env, Body: body}

	// CallExpression: 関数呼び出しを評価する
	case *ast.CallExpression:
		// まず関数自体を評価する
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		// 引数を左から右に評価する
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		// 関数を適用する
		return applyFunction(function, args)
	}

	return nil
}

// evalProgram はプログラム全体（文のリスト）を評価する。
// 各文を順に評価し、ReturnValueまたはErrorに遭遇したら即座に返す。
// ReturnValueの場合は中身を取り出して返す（プログラムレベルではアンラップ）。
func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value // ReturnValueをアンラップ
		case *object.Error:
			return result // エラーはそのまま返す
		}
	}

	return result
}

// evalBlockStatement はブロック内の文を評価する。
// evalProgram との違い: ReturnValueをアンラップしない。
// これにより、ネストされたブロックからのreturnが正しく伝播する。
// 例: if (true) { if (true) { return 10; } return 1; } → 10
func evalBlockStatement(
	block *ast.BlockStatement,
	env *object.Environment,
) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			// ReturnValueまたはErrorならそのまま返す（アンラップしない）
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

// nativeBoolToBooleanObject はGoのbool値をシングルトンのBooleanオブジェクトに変換する。
func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

// =====================
// 前置演算子の評価
// =====================

// evalPrefixExpression は前置演算子式を評価する。
func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

// evalBangOperatorExpression は ! 演算子を評価する。
// !true → false, !false → true, !null → true, それ以外 → false
func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		// 整数値など、truthyな値に対して ! を適用すると false になる
		return FALSE
	}
}

// evalMinusPrefixOperatorExpression は - 前置演算子を評価する。
// 整数にのみ適用可能。-5 は 5 の符号を反転させる。
func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

// =====================
// 中置演算子の評価
// =====================

// evalInfixExpression は中置演算子式を評価する。
// 両辺の型に応じて処理を分岐する。
func evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	// 両辺が整数の場合: 算術演算・比較演算
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	// == と != はポインタ比較（シングルトンなので正しく動く）
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	// 型が異なる場合（例: INTEGER + BOOLEAN）
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

// evalIntegerInfixExpression は整数同士の中置演算を評価する。
// 四則演算（+, -, *, /）と比較演算（<, >, ==, !=）をサポート。
func evalIntegerInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

// =====================
// if式の評価
// =====================

// evalIfExpression は if式を評価する。
// 条件がtruthyならConsequenceを、falsyでAlternativeがあればAlternativeを評価する。
// どちらにも当てはまらなければNULLを返す。
func evalIfExpression(
	ie *ast.IfExpression,
	env *object.Environment,
) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

// =====================
// 識別子と変数
// =====================

// evalIdentifier は識別子（変数名）を評価する。
// 環境から変数の値を検索し、見つからなければエラーを返す。
func evalIdentifier(
	node *ast.Identifier,
	env *object.Environment,
) object.Object {
	val, ok := env.Get(node.Value)
	if !ok {
		return newError("identifier not found: %s", node.Value)
	}

	return val
}

// =====================
// ユーティリティ関数
// =====================

// isTruthy はオブジェクトが「真」とみなされるか判定する。
// Monkey言語では: null → false, false → false, それ以外 → true
func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		// 整数値などは全てtruthy
		return true
	}
}

// newError はエラーオブジェクトを生成するヘルパー関数。
func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

// isError はオブジェクトがエラーかどうか判定する。
// 各評価関数でエラーチェックに使用し、エラーの伝播を実現する。
func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

// =====================
// 関数呼び出し
// =====================

// evalExpressions は式のリスト（関数引数など）を左から右に評価する。
// 途中でエラーが発生したら、エラーだけを含むスライスを返す。
func evalExpressions(
	exps []ast.Expression,
	env *object.Environment,
) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

// applyFunction は関数オブジェクトに引数を適用して実行する。
// 1. 関数の定義時環境を外側スコープとする新しい環境を作成
// 2. 引数をパラメータ名に束縛
// 3. 関数本体を新しい環境で評価
// 4. ReturnValueをアンラップして結果を返す
func applyFunction(fn object.Object, args []object.Object) object.Object {
	function, ok := fn.(*object.Function)
	if !ok {
		return newError("not a function: %s", fn.Type())
	}

	extendedEnv := extendFunctionEnv(function, args)
	evaluated := Eval(function.Body, extendedEnv)
	return unwrapReturnValue(evaluated)
}

// extendFunctionEnv は関数呼び出し用の新しい環境を作成する。
// 関数の定義時環境を外側として、引数をパラメータ名に束縛する。
// これがクロージャの仕組みの核心部分。
func extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
}

// unwrapReturnValue はReturnValueオブジェクトの中身を取り出す。
// 関数の本体評価後に呼ばれ、ReturnValueのラップを外して値だけを返す。
// これにより、returnが関数の外側まで伝播しないようにする。
func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

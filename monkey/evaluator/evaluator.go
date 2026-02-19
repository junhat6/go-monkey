// Package evaluator は Monkey言語のTree-walking評価器を実装するパッケージ。
// ASTを再帰的にたどりながら（tree-walking）、各ノードを評価して
// object.Object としての結果を返す。
//
// 4章で追加: 文字列リテラル・配列リテラル・インデックス式・ハッシュリテラルの評価、
// 文字列の連結（+演算子）、組み込み関数のサポート、
// 配列/ハッシュのインデックスアクセス。
package evaluator

import (
	"fmt"
	"monkey/ast"
	"monkey/object"
	"monkey/token"
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
//
// 4章で追加された分岐:
// - StringLiteral: 文字列リテラルの評価
// - ArrayLiteral: 配列リテラルの評価
// - IndexExpression: インデックスアクセスの評価
// - HashLiteral: ハッシュリテラルの評価
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

	// StringLiteral: 文字列リテラルをStringオブジェクトに変換（4章で追加）
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

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

	// Identifier: 環境から変数の値を取得する（組み込み関数も検索）
	case *ast.Identifier:
		return evalIdentifier(node, env)

	// FunctionLiteral: 関数オブジェクトを生成する（クロージャ）
	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Env: env, Body: body}

	// CallExpression: 関数呼び出しを評価する
	// 付録で追加: quote() は特別扱い（引数を評価しない）
	case *ast.CallExpression:
		if node.Function.TokenLiteral() == "quote" {
			return quote(node.Arguments[0], env)
		}

		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)

	// ArrayLiteral: 配列リテラルの要素を評価し、Arrayオブジェクトを生成（4章で追加）
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}

	// IndexExpression: インデックスアクセスを評価する（4章で追加）
	// 左辺（配列/ハッシュ）とインデックスを評価し、要素を取得
	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)

	// HashLiteral: ハッシュリテラルを評価する（4章で追加）
	case *ast.HashLiteral:
		return evalHashLiteral(node, env)

	}

	return nil
}

// evalProgram はプログラム全体（文のリスト）を評価する。
// 各文を順に評価し、ReturnValueまたはErrorに遭遇したら即座に返す。
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
func evalBlockStatement(
	block *ast.BlockStatement,
	env *object.Environment,
) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
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
func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

// evalMinusPrefixOperatorExpression は - 前置演算子を評価する。
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
// 4章で追加: 文字列同士の場合は evalStringInfixExpression に分岐。
func evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	// 4章で追加: 文字列同士の演算（連結 "hello" + " world"）
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

// evalIntegerInfixExpression は整数同士の中置演算を評価する。
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

// evalStringInfixExpression は文字列同士の中置演算を評価する。
// 現在は + 演算子（文字列連結）のみサポート。
// 4章で追加。
func evalStringInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	if operator != "+" {
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}

	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value
	return &object.String{Value: leftVal + rightVal}
}

// =====================
// if式の評価
// =====================

// evalIfExpression は if式を評価する。
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
// まずユーザー定義の変数を検索し、見つからなければ組み込み関数を検索する。
// どちらにもなければエラーを返す。
// 4章で変更: 組み込み関数（builtins）の検索を追加。
func evalIdentifier(
	node *ast.Identifier,
	env *object.Environment,
) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return newError("identifier not found: %s", node.Value)
}

// =====================
// ユーティリティ関数
// =====================

// isTruthy はオブジェクトが「真」とみなされるか判定する。
func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

// newError はエラーオブジェクトを生成するヘルパー関数。
func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

// isError はオブジェクトがエラーかどうか判定する。
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
// 4章で変更: switch文でユーザー定義関数（Function）と組み込み関数（Builtin）を
// 区別して処理するようになった。
func applyFunction(fn object.Object, args []object.Object) object.Object {
	switch fn := fn.(type) {

	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *object.Builtin:
		return fn.Fn(args...)

	default:
		return newError("not a function: %s", fn.Type())
	}
}

// extendFunctionEnv は関数呼び出し用の新しい環境を作成する。
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
func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

// =====================
// インデックスアクセス（4章で追加）
// =====================

// evalIndexExpression はインデックスアクセス式を評価する。
// 左辺の型に応じて配列アクセスとハッシュアクセスを分岐する。
// 4章で追加。
func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

// evalArrayIndexExpression は配列のインデックスアクセスを評価する。
// 範囲外アクセスの場合はNULLを返す（エラーにはしない）。
// 4章で追加。
func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return NULL
	}

	return arrayObject.Elements[idx]
}

// =====================
// ハッシュ（4章で追加）
// =====================

// evalHashLiteral はハッシュリテラルを評価する。
// 各キーと値のペアを評価し、キーが Hashable インターフェースを
// 実装しているか確認してからハッシュに格納する。
// 4章で追加。
func evalHashLiteral(
	node *ast.HashLiteral,
	env *object.Environment,
) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if isError(key) {
			return key
		}

		// キーが Hashable でなければエラー（例: 関数をキーにはできない）
		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", key.Type())
		}

		value := Eval(valueNode, env)
		if isError(value) {
			return value
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	return &object.Hash{Pairs: pairs}
}

// evalHashIndexExpression はハッシュのインデックスアクセスを評価する。
// キーが Hashable でなければエラー、キーが存在しなければNULLを返す。
// 4章で追加。
func evalHashIndexExpression(hash, index object.Object) object.Object {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return NULL
	}

	return pair.Value
}

// =====================
// quote/unquote（付録で追加）
// =====================

// quote はASTノードを評価せずにデータとして保持する。
// 内部で unquote() 呼び出しがあれば、その部分だけ評価して結果のASTノードに置換する。
// 付録で追加。
func quote(node ast.Node, env *object.Environment) object.Object {
	node = evalUnquoteCalls(node, env)
	return &object.Quote{Node: node}
}

// evalUnquoteCalls は quote されたAST内の unquote() 呼び出しを見つけて評価する。
// ast.Modify を使ってASTを走査し、unquote() の引数を評価した結果で置換する。
// 付録で追加。
func evalUnquoteCalls(quoted ast.Node, env *object.Environment) ast.Node {
	return ast.Modify(quoted, func(node ast.Node) ast.Node {
		if !isUnquoteCall(node) {
			return node
		}

		call, ok := node.(*ast.CallExpression)
		if !ok {
			return node
		}

		if len(call.Arguments) != 1 {
			return node
		}

		unquoted := Eval(call.Arguments[0], env)
		return convertObjectToASTNode(unquoted)
	})
}

// isUnquoteCall はノードが unquote() 関数呼び出しかどうか判定する。
// 付録で追加。
func isUnquoteCall(node ast.Node) bool {
	callExpression, ok := node.(*ast.CallExpression)
	if !ok {
		return false
	}

	return callExpression.Function.TokenLiteral() == "unquote"
}

// convertObjectToASTNode はオブジェクトをASTノードに変換する。
// unquote() で評価した結果をASTに埋め戻すために使う。
// 付録で追加。
func convertObjectToASTNode(obj object.Object) ast.Node {
	switch obj := obj.(type) {
	case *object.Integer:
		t := token.Token{
			Type:    token.INT,
			Literal: fmt.Sprintf("%d", obj.Value),
		}
		return &ast.IntegerLiteral{Token: t, Value: obj.Value}

	case *object.Boolean:
		var t token.Token
		if obj.Value {
			t = token.Token{Type: token.TRUE, Literal: "true"}
		} else {
			t = token.Token{Type: token.FALSE, Literal: "false"}
		}
		return &ast.Boolean{Token: t, Value: obj.Value}

	case *object.Quote:
		return obj.Node

	default:
		return nil
	}
}

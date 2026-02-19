// Package object は Monkey言語のランタイムオブジェクトシステムを定義するパッケージ。
// 評価器（Evaluator）がASTを評価した結果はすべてこのパッケージの Object として表現される。
// 全てのオブジェクトは Object インターフェースを実装する。
package object

import (
	"bytes"
	"fmt"
	"monkey/ast"
	"strings"
)

// ObjectType はオブジェクトの種類を識別する文字列型。
type ObjectType string

// オブジェクトの種類を表す定数。
const (
	NULL_OBJ  = "NULL"  // null値
	ERROR_OBJ = "ERROR" // エラーオブジェクト

	INTEGER_OBJ = "INTEGER" // 整数
	BOOLEAN_OBJ = "BOOLEAN" // 真偽値

	RETURN_VALUE_OBJ = "RETURN_VALUE" // return文の戻り値をラップするオブジェクト

	FUNCTION_OBJ = "FUNCTION" // 関数オブジェクト
)

// Object はMonkey言語の全ての値が実装するインターフェース。
// Type() はオブジェクトの種類を返し、Inspect() は値の文字列表現を返す。
type Object interface {
	Type() ObjectType
	Inspect() string
}

// Integer は整数値を表すオブジェクト。
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

// Boolean は真偽値を表すオブジェクト。
// 評価器ではシングルトン（TRUE, FALSE）として扱い、メモリ効率を上げている。
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

// Null はnull値を表すオブジェクト。
// 値が存在しないことを表す。評価器ではシングルトン（NULL）として扱う。
type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

// ReturnValue はreturn文の戻り値をラップするオブジェクト。
// 評価器がreturn文に遭遇すると、このオブジェクトでラップして
// 呼び出しスタックを巻き戻す。最終的に unwrapReturnValue() で取り出される。
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// Error はエラーを表すオブジェクト。
// 型の不一致や未定義の識別子などのエラー情報をメッセージとして持つ。
// エラーは評価中に伝播し、以降の評価を停止させる。
type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

// Function は関数オブジェクト。
// Parameters は仮引数リスト、Body は関数本体、Env は定義時の環境。
// Env を保持することでクロージャ（外側のスコープの変数を参照する関数）を実現する。
type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }

// Inspect は関数の文字列表現を返す。
// `fn(params) { body }` の形式。
func (f *Function) Inspect() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("fn")
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n}")

	return out.String()
}

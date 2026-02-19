// Package object は Monkey言語のランタイムオブジェクトシステムを定義するパッケージ。
// 評価器（Evaluator）がASTを評価した結果はすべてこのパッケージの Object として表現される。
// 全てのオブジェクトは Object インターフェースを実装する。
//
// 4章で追加: String（文字列）、Builtin（組み込み関数）、Array（配列）、
// Hash（ハッシュ）、HashPair、HashKey、Hashable インターフェース。
// ハッシュのキーとして使えるのは Hashable を実装した型のみ
// （Integer, Boolean, String）。
package object

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"monkey/ast"
	"strings"
)

// BuiltinFunction は組み込み関数の型。
// 可変長引数を受け取り、Objectを返す。
// 4章で追加。
type BuiltinFunction func(args ...Object) Object

// ObjectType はオブジェクトの種類を識別する文字列型。
type ObjectType string

// オブジェクトの種類を表す定数。
// 4章で追加: STRING_OBJ, BUILTIN_OBJ, ARRAY_OBJ, HASH_OBJ
const (
	NULL_OBJ  = "NULL"  // null値
	ERROR_OBJ = "ERROR" // エラーオブジェクト

	INTEGER_OBJ = "INTEGER" // 整数
	BOOLEAN_OBJ = "BOOLEAN" // 真偽値
	STRING_OBJ  = "STRING"  // 文字列

	RETURN_VALUE_OBJ = "RETURN_VALUE" // return文の戻り値をラップするオブジェクト

	FUNCTION_OBJ = "FUNCTION" // ユーザー定義関数
	BUILTIN_OBJ  = "BUILTIN"  // 組み込み関数

	ARRAY_OBJ = "ARRAY" // 配列
	HASH_OBJ  = "HASH"  // ハッシュ（連想配列）

	QUOTE_OBJ = "QUOTE" // quote（ASTノードをデータとして保持）（付録で追加）
	MACRO_OBJ = "MACRO" // マクロ（付録で追加）
)

// HashKey はハッシュのキーとして使うための構造体。
// Type はオブジェクトの型、Value はハッシュ値。
// 同じ値を持つオブジェクトは同じ HashKey を生成する必要がある。
// 4章で追加。
type HashKey struct {
	Type  ObjectType
	Value uint64
}

// Hashable はハッシュのキーとして使えるオブジェクトが実装するインターフェース。
// HashKey() メソッドで一意なハッシュキーを返す。
// Integer, Boolean, String がこれを実装する。
// 4章で追加。
type Hashable interface {
	HashKey() HashKey
}

// Object はMonkey言語の全ての値が実装するインターフェース。
// Type() はオブジェクトの種類を返し、Inspect() は値の文字列表現を返す。
type Object interface {
	Type() ObjectType
	Inspect() string
}

// Integer は整数値を表すオブジェクト。
// 4章で追加: HashKey() メソッドを実装し、ハッシュのキーとして使えるようになった。
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

// HashKey は整数値をそのままハッシュキーとして返す。
func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: uint64(i.Value)}
}

// Boolean は真偽値を表すオブジェクト。
// 4章で追加: HashKey() メソッドを実装。
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

// HashKey は true なら 1、false なら 0 をハッシュキーとして返す。
func (b *Boolean) HashKey() HashKey {
	var value uint64

	if b.Value {
		value = 1
	} else {
		value = 0
	}

	return HashKey{Type: b.Type(), Value: value}
}

// Null はnull値を表すオブジェクト。
type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

// ReturnValue はreturn文の戻り値をラップするオブジェクト。
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// Error はエラーを表すオブジェクト。
type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

// Function はユーザー定義関数オブジェクト。
// Env を保持することでクロージャを実現する。
type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }

// Inspect は関数の文字列表現を返す。
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

// String は文字列を表すオブジェクト。
// 4章で追加: HashKey() メソッドを実装し、ハッシュのキーとして使えるようになった。
// ハッシュ値の計算には FNV-1a アルゴリズムを使用。
type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

// HashKey は文字列の FNV-1a ハッシュ値をキーとして返す。
func (s *String) HashKey() HashKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))

	return HashKey{Type: s.Type(), Value: h.Sum64()}
}

// Builtin は組み込み関数を表すオブジェクト。
// Fn にGoで実装された関数を保持する。
// 4章で追加。
type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// Array は配列を表すオブジェクト。
// Elements に任意のObjectのスライスを保持する。
// 4章で追加。
type Array struct {
	Elements []Object
}

func (ao *Array) Type() ObjectType { return ARRAY_OBJ }

// Inspect は `[elem1, elem2, ...]` の形式で返す。
func (ao *Array) Inspect() string {
	var out bytes.Buffer

	elements := []string{}
	for _, e := range ao.Elements {
		elements = append(elements, e.Inspect())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// HashPair はハッシュの1エントリ（キーと値のペア）を表す。
// Key は元のオブジェクト（表示用）、Value は対応する値。
// 4章で追加。
type HashPair struct {
	Key   Object
	Value Object
}

// Hash はハッシュ（連想配列）を表すオブジェクト。
// Pairs は HashKey をキーにした HashPair のマップ。
// HashKey で検索することで O(1) のアクセスを実現する。
// 4章で追加。
type Hash struct {
	Pairs map[HashKey]HashPair
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }

// Inspect は `{key1: value1, key2: value2}` の形式で返す。
func (h *Hash) Inspect() string {
	var out bytes.Buffer

	pairs := []string{}
	for _, pair := range h.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s",
			pair.Key.Inspect(), pair.Value.Inspect()))
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

// =====================
// 付録で追加されたオブジェクト
// =====================

// Quote はASTノードをデータとして保持するオブジェクト。
// quote() 関数で生成され、コードをデータとして扱うために使う。
// 付録で追加。
type Quote struct {
	Node ast.Node
}

func (q *Quote) Type() ObjectType { return QUOTE_OBJ }
func (q *Quote) Inspect() string {
	return "QUOTE(" + q.Node.String() + ")"
}

// Macro はマクロオブジェクト。
// ユーザー定義関数と同じくパラメータ、本体、環境を持つが、
// 呼び出し時に引数を評価せず、ASTノードをそのまま受け取る。
// 付録で追加。
type Macro struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (m *Macro) Type() ObjectType { return MACRO_OBJ }

// Inspect はマクロの文字列表現を返す。
func (m *Macro) Inspect() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range m.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("macro")
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(m.Body.String())
	out.WriteString("\n}")

	return out.String()
}

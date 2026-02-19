// Package ast は Monkey言語の抽象構文木（AST）を定義するパッケージ。
// パーサーがソースコードをトークン列から変換した結果がこのASTになる。
// ASTの各ノードは Node インターフェースを実装し、
// 文（Statement）と式（Expression）の2種類に大別される。
package ast

import (
	"bytes"
	"monkey/token"
	"strings"
)

// Node はASTの全ノードが実装する基本インターフェース。
// TokenLiteral() はデバッグ用にトークンのリテラル値を返す。
// String() はノードを人間が読める文字列に変換する。
type Node interface {
	TokenLiteral() string
	String() string
}

// Statement は「文」を表すノードのインターフェース。
// statementNode() はマーカーメソッドで、式と文を型レベルで区別するために使う。
type Statement interface {
	Node
	statementNode()
}

// Expression は「式」を表すノードのインターフェース。
// expressionNode() はマーカーメソッドで、式と文を型レベルで区別するために使う。
type Expression interface {
	Node
	expressionNode()
}

// Program はASTのルートノード。
// Monkey言語のプログラムは文（Statement）の列で構成される。
type Program struct {
	Statements []Statement
}

// TokenLiteral は最初の文のトークンリテラルを返す。
func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

// String はプログラム全体を文字列に変換する。
// 各文のString()を連結して返す。
func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// =====================
// 文（Statements）
// =====================

// LetStatement は `let x = <expression>;` という変数束縛の文を表す。
// Name は束縛先の識別子、Value は束縛する値の式。
type LetStatement struct {
	Token token.Token // token.LET トークン
	Name  *Identifier
	Value Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }

// String は `let <name> = <value>;` の形式で文字列を返す。
func (ls *LetStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")

	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}

// ReturnStatement は `return <expression>;` というreturn文を表す。
type ReturnStatement struct {
	Token       token.Token // 'return' トークン
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }

// String は `return <value>;` の形式で文字列を返す。
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	out.WriteString(";")

	return out.String()
}

// ExpressionStatement は式だけからなる文を表す。
// Monkey言語では `x + 10;` のように式を文として扱える。
type ExpressionStatement struct {
	Token      token.Token // その式の最初のトークン
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

// String は内部の式を文字列化して返す。
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement は `{ ... }` で囲まれたブロック（文の列）を表す。
// if式やfunction literalの本体部分で使われる。
type BlockStatement struct {
	Token      token.Token // '{' トークン
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }

// String はブロック内の全文を連結して返す。
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// =====================
// 式（Expressions）
// =====================

// Identifier は変数名などの識別子を表す。
// 式としても扱われる（例: `foobar` を評価するとその値が返る）。
type Identifier struct {
	Token token.Token // token.IDENT トークン
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// Boolean は true/false のブーリアンリテラルを表す。
type Boolean struct {
	Token token.Token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

// IntegerLiteral は整数リテラル（例: 5, 100）を表す。
type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// PrefixExpression は前置演算子式（例: !true, -5）を表す。
// Operator は演算子（"!" や "-"）、Right は右辺の式。
type PrefixExpression struct {
	Token    token.Token // 前置演算子のトークン（例: !）
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }

// String は `(<operator><right>)` の形式で返す（例: "(-5)"）。
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

// InfixExpression は中置演算子式（例: 5 + 10, a == b）を表す。
// Left は左辺、Operator は演算子、Right は右辺。
type InfixExpression struct {
	Token    token.Token // 演算子トークン（例: +）
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }

// String は `(<left> <operator> <right>)` の形式で返す（例: "(5 + 10)"）。
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")

	return out.String()
}

// IfExpression は `if (<condition>) <consequence> else <alternative>` を表す。
// Condition は条件式、Consequence は真の場合のブロック、
// Alternative は偽の場合のブロック（省略可能）。
type IfExpression struct {
	Token       token.Token // 'if' トークン
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }

// String は if式を人間が読める形式に変換する。
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

// FunctionLiteral は関数リテラル `fn(<params>) <body>` を表す。
// Monkey言語では関数は第一級オブジェクト（値として扱える）。
type FunctionLiteral struct {
	Token      token.Token // 'fn' トークン
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }

// String は `fn(<params>) <body>` の形式で返す。
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}

	out.WriteString(fl.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fl.Body.String())

	return out.String()
}

// CallExpression は関数呼び出し `<function>(<args>)` を表す。
// Function は呼び出す関数（識別子またはFunctionLiteral）。
// Arguments は引数のリスト。
type CallExpression struct {
	Token     token.Token // '(' トークン
	Function  Expression  // Identifier または FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }

// String は `<function>(<args>)` の形式で返す。
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

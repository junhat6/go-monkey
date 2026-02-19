package ast

import (
	"monkey/token"
	"testing"
)

// TestString はASTノードのString()メソッドが正しく動作するかテストする。
// `let myVar = anotherVar;` というプログラムを手動でAST構築し、
// String()の出力が期待通りかを検証する。
func TestString(t *testing.T) {
	// ASTを手動で組み立てる（パーサーを使わずに直接構築）
	program := &Program{
		Statements: []Statement{
			&LetStatement{
				Token: token.Token{Type: token.LET, Literal: "let"},
				Name: &Identifier{
					Token: token.Token{Type: token.IDENT, Literal: "myVar"},
					Value: "myVar",
				},
				Value: &Identifier{
					Token: token.Token{Type: token.IDENT, Literal: "anotherVar"},
					Value: "anotherVar",
				},
			},
		},
	}

	if program.String() != "let myVar = anotherVar;" {
		t.Errorf("program.String() wrong. got=%q", program.String())
	}
}

// Package token は Monkey言語のトークン（字句）を定義するパッケージ。
// レキサーがソースコードを分割した最小単位がトークンであり、
// パーサーはこのトークン列を入力として構文解析を行う。
package token

// TokenType はトークンの種類を文字列で表す型。
type TokenType string

const (
	ILLEGAL = "ILLEGAL" // 未知のトークン
	EOF     = "EOF"     // 入力の終端

	// 識別子 + リテラル
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	INT    = "INT"    // 1343456
	STRING = "STRING" // "foobar"

	// 演算子
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"

	LT = "<"
	GT = ">"

	EQ     = "=="
	NOT_EQ = "!="

	// デリミタ（区切り文字）
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":" // ハッシュリテラルのキーと値の区切り

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "[" // 配列リテラル・インデックスアクセス
	RBRACKET = "]"

	// キーワード
	FUNCTION = "FUNCTION"
	LET      = "LET"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	MACRO    = "MACRO" // マクロ定義（付録で追加）
)

// Token はトークンの型とリテラル値のペア。
type Token struct {
	Type    TokenType
	Literal string
}

// keywords はMonkey言語の予約語マップ。
var keywords = map[string]TokenType{
	"fn":     FUNCTION,
	"let":    LET,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
	"macro":  MACRO,
}

// LookupIdent は識別子が予約語かどうかを判定する。
// 予約語であればそのトークン型を、そうでなければIDENTを返す。
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

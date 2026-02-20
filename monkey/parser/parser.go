// Package parser は Monkey言語のパーサーを実装するパッケージ。
// Pratt Parser（トップダウン演算子順位解析法）を使って、
// トークン列をAST（抽象構文木）に変換する。
//
// Pratt Parserの核心的なアイデア:
// - 各トークンタイプに「前置解析関数」と「中置解析関数」を関連付ける
// - 演算子の優先順位（precedence）に基づいて正しい構文木を構築する
//
// 4章で追加: INDEX優先順位、文字列リテラル・配列リテラル・インデックス式・
// ハッシュリテラルのパース。parseCallArguments を parseExpressionList に
// 汎用化（配列リテラルと関数呼び出しで共有）。
package parser

import (
	"fmt"
	"monkey/ast"
	"monkey/lexer"
	"monkey/token"
	"strconv"
)

// 演算子の優先順位を定数で定義する。
// 数値が大きいほど優先順位が高い。
// 例: * は + より優先順位が高いので、`1 + 2 * 3` は `1 + (2 * 3)` になる。
// 4章で追加: INDEX（配列やハッシュのインデックスアクセス）が最も高い優先順位。
const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > または <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X または !X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

// precedences はトークンタイプから優先順位への対応表。
// この表に基づいてパーサーが演算子の結合順序を決定する。
// 4章で追加: LBRACKET → INDEX（インデックスアクセスの優先順位）。
var precedences = map[token.TokenType]int{
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
}

// prefixParseFn は前置解析関数の型。
// トークンが式の先頭に来た場合に呼ばれる（例: -5, !true, 識別子, 整数リテラル）。
type (
	prefixParseFn func() ast.Expression
	// infixParseFn は中置解析関数の型。
	// 左辺の式を引数に取り、中置演算子の右辺を解析して完全な式を返す。
	infixParseFn func(ast.Expression) ast.Expression
)

// Parser はMonkey言語のパーサー。
// レキサーからトークンを読み取り、ASTを構築する。
type Parser struct {
	l      *lexer.Lexer // トークンを供給するレキサー
	errors []string     // パース中に発生したエラーメッセージ

	curToken  token.Token // 現在見ているトークン
	peekToken token.Token // 次のトークン（先読み用）

	// 各トークンタイプに対応する解析関数を登録するマップ
	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

// New はレキサーからパーサーを生成する。
// 各トークンタイプに対して適切な解析関数を登録し、
// 最初の2トークンを読み込んで curToken と peekToken をセットする。
//
// 4章で追加された登録:
// - STRING → parseStringLiteral（文字列リテラル）
// - LBRACKET → parseArrayLiteral（配列リテラル、前置）
// - LBRACE → parseHashLiteral（ハッシュリテラル、前置）
// - LBRACKET → parseIndexExpression（インデックスアクセス、中置）
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// 前置解析関数の登録
	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.MACRO, p.parseMacroLiteral)
	p.registerPrefix(token.FOR, p.parseForExpression)

	// 中置解析関数の登録
	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)

	// '(' は関数呼び出しの中置演算子として扱う（例: add(1, 2)）
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	// '[' はインデックスアクセスの中置演算子として扱う（例: arr[0]）
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)

	// curToken と peekToken の両方をセットするために2回読む
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken は次のトークンに進む。
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// curTokenIs は現在のトークンが指定された型か判定する。
func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs は次のトークンが指定された型か判定する。
func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek は次のトークンが期待する型であればトークンを進めてtrueを返す。
// 期待と違う場合はエラーを追加してfalseを返す。
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// Errors はパース中に蓄積されたエラーメッセージのスライスを返す。
func (p *Parser) Errors() []string {
	return p.errors
}

// peekError は次のトークンが期待と違った場合にエラーメッセージを追加する。
func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// noPrefixParseFnError はトークンに対応する前置解析関数がない場合のエラー。
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

// =====================
// プログラムと文のパース
// =====================

// ParseProgram はプログラム全体をパースしてASTのルートノードを返す。
// EOF に到達するまで文を1つずつパースしてProgramに追加していく。
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseStatement は現在のトークンに応じて適切な種類の文をパースする。
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseLetStatement は `let <identifier> = <expression>;` をパースする。
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseReturnStatement は `return <expression>;` をパースする。
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseExpressionStatement は式だけからなる文をパースする。
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// =====================
// 式のパース（Pratt Parser の心臓部）
// =====================

// parseExpression はPratt Parserのメインループ。
//  1. まず現在のトークンに対応する前置解析関数を呼んで左辺の式を得る
//  2. 次のトークンの優先順位が現在の優先順位より高い間、
//     中置解析関数を呼んで左辺に演算子と右辺を結合していく
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// peekPrecedence は次のトークンの優先順位を返す。
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}

	return LOWEST
}

// curPrecedence は現在のトークンの優先順位を返す。
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}

	return LOWEST
}

// =====================
// 各種式の解析関数
// =====================

// parseIdentifier は識別子をパースする。
func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parseIntegerLiteral は整数リテラルをパースする。
// 文字列を int64 に変換し、失敗した場合はエラーを追加する。
func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value

	return lit
}

// parseStringLiteral は文字列リテラルをパースする。
// レキサーがクォートを除いた文字列をLiteralに格納済みなので、
// そのまま StringLiteral ノードを生成する。
// 4章で追加。
func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// parsePrefixExpression は前置演算子式（!x, -5 など）をパースする。
func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// parseInfixExpression は中置演算子式（5 + 10 など）をパースする。
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

// parseBoolean はブーリアンリテラル（true/false）をパースする。
func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

// parseGroupedExpression は括弧で囲まれた式 `(expression)` をパースする。
func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

// parseIfExpression は `if (<condition>) <consequence> else <alternative>` をパースする。
func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

// parseBlockStatement は `{ ... }` 内の文をパースする。
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// parseFunctionLiteral は `fn(<params>) <body>` をパースする。
func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

// parseFunctionParameters は関数のパラメータリスト `(x, y, z)` をパースする。
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

// parseCallExpression は関数呼び出し `<expression>(<args>)` をパースする。
// 4章で変更: parseCallArguments → parseExpressionList を使うように汎用化。
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

// parseExpressionList はカンマ区切りの式リストをパースする汎用関数。
// end で指定したトークンが来るまで式を読み続ける。
// 関数呼び出しの引数リスト `(a, b, c)` と
// 配列リテラルの要素リスト `[1, 2, 3]` の両方で使われる。
// 4章で追加（3章の parseCallArguments を汎用化したもの）。
func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	// 要素が0個の場合
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	// 最初の要素
	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	// カンマ区切りで残りの要素を読む
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

// parseArrayLiteral は配列リテラル `[<elements>]` をパースする。
// parseExpressionList を使って要素リストを読み取る。
// 4章で追加。
func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(token.RBRACKET)

	return array
}

// parseIndexExpression はインデックスアクセス式 `<left>[<index>]` をパースする。
// 配列アクセス（arr[0]）やハッシュアクセス（hash["key"]）で使われる。
// 中置解析関数として登録され、左辺（配列やハッシュ）を引数に取る。
// 4章で追加。
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

// parseMacroLiteral はマクロリテラル `macro(<params>) <body>` をパースする。
// FunctionLiteral と同じ構造を持つが、トークンが macro である。
// 付録で追加。
func (p *Parser) parseMacroLiteral() ast.Expression {
	lit := &ast.MacroLiteral{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

// parseHashLiteral はハッシュリテラル `{<key>:<value>, ...}` をパースする。
// キーは任意の式（文字列、整数、ブーリアン等）、値も任意の式。
// 4章で追加。
func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.Expression]ast.Expression)

	// '}' が来るまでキーと値のペアを読み続ける
	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		// キーの後に ':' が来なければならない
		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		hash.Pairs[key] = value

		// '}' でなければ ',' が来なければならない
		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

// registerPrefix は前置解析関数を登録するヘルパー。
func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

// registerInfix は中置解析関数を登録するヘルパー。
func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// for (<init>; <condition>; <update>) { <body> }
func (p *Parser) parseForExpression() ast.Expression {
	expression := &ast.ForExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	// Init部分
	p.nextToken()
	if p.curTokenIs(token.LET) {
		expression.Init = p.parseLetStatement()
	} else if !p.curTokenIs(token.SEMICOLON) {
		expression.Init = p.parseExpressionStatement()
	} else {
	}

	// Condition部分
	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) {
		expression.Condition = p.parseExpression(LOWEST)
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
	}

	// Update部分
	p.nextToken()
	if p.curTokenIs(token.LET) {
		expression.Update = p.parseLetStatement()
	} else if !p.curTokenIs(token.RPAREN) {
		expression.Update = p.parseExpressionStatement()
	}

	if !p.curTokenIs(token.RPAREN) {
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Body = p.parseBlockStatement()
	return expression
}

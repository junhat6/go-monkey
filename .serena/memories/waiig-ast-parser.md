# AST & Parser 詳細

## AST (ast/ast.go)

### 基本インターフェース
```go
type Node interface {
    TokenLiteral() string
    String() string
}

type Statement interface {
    Node
    statementNode()
}

type Expression interface {
    Node
    expressionNode()
}
```

### Program (ルートノード)
```go
type Program struct {
    Statements []Statement
}
```

### Statement種類
| Statement | 説明 | 例 |
|-----------|------|-----|
| LetStatement | 変数束縛 | `let x = 5;` |
| ReturnStatement | 戻り値 | `return 5;` |
| ExpressionStatement | 式文 | `x + y` |
| BlockStatement | ブロック | `{ ... }` |

### Expression種類
| Expression | 説明 | 例 |
|------------|------|-----|
| Identifier | 識別子 | `foobar` |
| IntegerLiteral | 整数 | `5` |
| StringLiteral | 文字列 | `"hello"` |
| Boolean | 真偽値 | `true`, `false` |
| PrefixExpression | 前置式 | `-5`, `!true` |
| InfixExpression | 中置式 | `5 + 5` |
| IfExpression | 条件式 | `if (x) { y }` |
| FunctionLiteral | 関数 | `fn(x) { x }` |
| CallExpression | 関数呼び出し | `add(1, 2)` |
| ArrayLiteral | 配列 | `[1, 2, 3]` |
| IndexExpression | インデックス | `arr[0]` |
| HashLiteral | ハッシュ | `{"a": 1}` |

## Parser (parser/parser.go)

### Pratt Parser の仕組み
- トークン種別ごとにパース関数を登録
- 前置パース関数 (prefixParseFn)
- 中置パース関数 (infixParseFn)

### Parser構造体
```go
type Parser struct {
    l      *lexer.Lexer
    errors []string

    curToken  token.Token   // 現在のトークン
    peekToken token.Token   // 次のトークン

    prefixParseFns map[token.TokenType]prefixParseFn
    infixParseFns  map[token.TokenType]infixParseFn
}
```

### 演算子優先順位
```go
var precedences = map[token.TokenType]int{
    token.EQ:       EQUALS,      // ==
    token.NOT_EQ:   EQUALS,      // !=
    token.LT:       LESSGREATER, // <
    token.GT:       LESSGREATER, // >
    token.PLUS:     SUM,         // +
    token.MINUS:    SUM,         // -
    token.SLASH:    PRODUCT,     // /
    token.ASTERISK: PRODUCT,     // *
    token.LPAREN:   CALL,        // (
    token.LBRACKET: INDEX,       // [
}
```

### 式のパース (Pratt Parser核心)
```go
func (p *Parser) parseExpression(precedence int) ast.Expression {
    prefix := p.prefixParseFns[p.curToken.Type]
    if prefix == nil {
        return nil
    }
    leftExp := prefix()

    // 優先順位が高い限り中置式として解析
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
```

### パース関数の登録
```go
// 前置パース関数
p.registerPrefix(token.IDENT, p.parseIdentifier)
p.registerPrefix(token.INT, p.parseIntegerLiteral)
p.registerPrefix(token.BANG, p.parsePrefixExpression)
p.registerPrefix(token.MINUS, p.parsePrefixExpression)
p.registerPrefix(token.IF, p.parseIfExpression)
p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)

// 中置パース関数
p.registerInfix(token.PLUS, p.parseInfixExpression)
p.registerInfix(token.LPAREN, p.parseCallExpression)
p.registerInfix(token.LBRACKET, p.parseIndexExpression)
```

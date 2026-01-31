# Token & Lexer 詳細

## Token (token/token.go)

### Token構造体
```go
type TokenType string

type Token struct {
    Type    TokenType
    Literal string
}
```

### トークン種別
| カテゴリ | トークン |
|---------|---------|
| リテラル | IDENT, INT, STRING |
| 演算子 | =, +, -, !, *, /, <, >, ==, != |
| デリミタ | ,, ;, :, (, ), {, }, [, ] |
| キーワード | fn, let, true, false, if, else, return |
| 特殊 | ILLEGAL, EOF |

### キーワード判定
```go
var keywords = map[string]TokenType{
    "fn":     FUNCTION,
    "let":    LET,
    "true":   TRUE,
    "false":  FALSE,
    "if":     IF,
    "else":   ELSE,
    "return": RETURN,
}

func LookupIdent(ident string) TokenType {
    if tok, ok := keywords[ident]; ok {
        return tok
    }
    return IDENT
}
```

## Lexer (lexer/lexer.go)

### Lexer構造体
```go
type Lexer struct {
    input        string  // ソースコード
    position     int     // 現在位置
    readPosition int     // 次の読み込み位置
    ch           byte    // 現在の文字
}
```

### 主要メソッド
- `New(input string) *Lexer` - Lexer生成
- `NextToken() Token` - 次のトークンを返す
- `readChar()` - 1文字読み進める
- `peekChar() byte` - 次の文字を覗き見
- `skipWhitespace()` - 空白をスキップ
- `readIdentifier() string` - 識別子を読む
- `readNumber() string` - 数値を読む
- `readString() string` - 文字列を読む

### 2文字演算子の処理
`==`や`!=`は`peekChar()`で先読みして判定:
```go
case '=':
    if l.peekChar() == '=' {
        ch := l.ch
        l.readChar()
        literal := string(ch) + string(l.ch)
        tok = token.Token{Type: token.EQ, Literal: literal}
    } else {
        tok = newToken(token.ASSIGN, l.ch)
    }
```

### 文字判定
```go
func isLetter(ch byte) bool {
    return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
    return '0' <= ch && ch <= '9'
}
```

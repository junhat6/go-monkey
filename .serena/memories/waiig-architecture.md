# WAIIG アーキテクチャ詳細

## インタプリタのパイプライン

```
ソースコード → Lexer → Parser → AST → Evaluator → 結果
```

## 章構成と実装内容

### Chapter 1: Lexer (字句解析)
- ソースコードをトークンに分割
- 実装ファイル: `token/token.go`, `lexer/lexer.go`

### Chapter 2: Parser (構文解析)
- トークン列からAST (抽象構文木) を構築
- **Pratt Parser** (Top-Down Operator Precedence Parser) を使用
- 実装ファイル: `parser/parser.go`, `ast/ast.go`

### Chapter 3: Evaluator (評価)
- ASTを再帰的に評価して値を計算
- Tree-walking interpreter
- 実装ファイル: `evaluator/evaluator.go`, `object/object.go`

### Chapter 4: Extensions (拡張)
- 組み込み関数
- 文字列、配列、ハッシュのサポート
- 実装ファイル: `evaluator/builtins.go`

## パッケージ構成

```
monkey/
├── token/          # トークン定義
├── lexer/          # 字句解析器
├── ast/            # 抽象構文木
├── parser/         # 構文解析器 (Pratt Parser)
├── object/         # ランタイム値とEnvironment
├── evaluator/      # 評価器と組み込み関数
├── repl/           # REPL実装
└── main.go         # エントリポイント
```

## 演算子優先順位 (Pratt Parser)
```go
const (
    LOWEST      // 最低
    EQUALS      // ==
    LESSGREATER // > or <
    SUM         // +
    PRODUCT     // *
    PREFIX      // -X or !X
    CALL        // myFunction(X)
    INDEX       // array[index]
)
```

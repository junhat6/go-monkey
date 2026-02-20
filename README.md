# go-monkey

[Go言語でつくるインタプリタ](https://www.oreilly.co.jp/books/9784873118222/)の実装

## 実装内容

- 字句解析器（Lexer）
- Pratt構文解析器（Parser）
- AST（抽象構文木）
- Tree-Walking評価器（Evaluator）
- REPL
- データ型: 整数、真偽値、文字列、配列、ハッシュ、null
- 変数束縛（`let`文）
- 算術演算子（`+`, `-`, `*`, `/`）
- 比較演算子（`==`, `!=`, `<`, `>`）
- 前置演算子（`!`, `-`）
- 文字列結合（`+`）
- if/else式
- 関数とクロージャ（第一級関数）
- 組み込み関数: `len`, `puts`, `first`, `last`, `rest`, `push`
- インデックス演算子（配列・ハッシュ）
- エラーハンドリング（エラーオブジェクトの伝播）
- マクロシステム（`quote`, `unquote`, `macro`）

## marp preview

```bash
npx @marp-team/marp-cli slides.md --preview
```


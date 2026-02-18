---
marp: true
theme: default
paginate: true
style: |
  section {
    font-family: 'Helvetica Neue', Arial, 'Hiragino Kaku Gothic ProN', sans-serif;
  }
  h1 {
    color: #2d3748;
  }
  h2 {
    color: #4a5568;
  }
  code {
    background: #f7fafc;
  }
  table {
    font-size: 0.85em;
  }
---

# Go言語でつくるインタプリタ
## 振り返り会 - 2026/02/06

---

# 今日のアジェンダ

1. **この取り組みの目的**
2. **書籍の概要** - 何を作っているのか
3. **第1章** - Lexer（字句解析器）
4. **第2章** - Parser & AST（構文解析）
5. **第3章** - Evaluator（評価器）
6. **学んだこと・気づき**
7. **聞きたいこと**

---

# この取り組みの目的

| ゴール | 3章までの関連 |
|--------|-------------|
| コンピュータは何故動くのかを自分なりに理解する | ソースコード → トークン → AST → 実行の流れを体験 |
| CS基礎の組み合わせによる構造を理解する | 字句解析・構文解析・評価という段階的な処理 |
| テスト駆動開発の雰囲気を掴む | 各章でテストを先に書いてから実装する進め方 |
| 学んだことのまとめアウトプットを出す | 今回の振り返り会がまさにこれ |

**ストレッチ課題**: 本に無いオリジナル機能を追加する（今後の目標）

---

# 書籍の概要

**Writing An Interpreter In Go**（Thorsten Ball 著）

- **Monkey言語**というオリジナル言語のインタプリタをGoで実装
- 外部ライブラリ不使用、フルスクラッチで構築
- 全4章構成（今回は1〜3章）

---

# Monkey言語でできること

```javascript
let age = 1;
let name = "Monkey";
let result = 10 * (20 / 2);

let add = fn(a, b) { a + b; };
add(1, 2);

if (age > 0) {
    return "adult";
} else {
    return "child";
}
```

変数束縛、算術演算、関数、条件分岐などを備えた言語

---

# インタプリタの全体像

```
ソースコード（文字列）
       ↓
┌──────────────┐
│   Lexer      │  第1章：文字列 → トークン列
│  （字句解析） │
└──────────────┘
       ↓
┌──────────────┐
│   Parser     │  第2章：トークン列 → AST
│  （構文解析） │
└──────────────┘
       ↓
┌──────────────┐
│  Evaluator   │  第3章：AST → 実行結果
│   （評価）    │
└──────────────┘
```

---

# 第1章：Lexer（字句解析器）

---

# トークンとは

**プログラムの意味のある最小単位**

```
let x = 5 + 10;
```
↓ トークン化

| トークン | 種類 |
|---------|------|
| `let` | キーワード（LET） |
| `x` | 識別子（IDENT） |
| `=` | 代入演算子（ASSIGN） |
| `5` | 整数（INT） |
| `+` | 加算演算子（PLUS） |
| `10` | 整数（INT） |
| `;` | セミコロン（SEMICOLON） |

---

# Lexerの仕組み

文字列を1文字ずつ読み進めてトークンを生成

```go
type Lexer struct {
    input        string
    position     int   // 現在の位置
    readPosition int   // 次の位置
    ch           byte  // 現在の文字
}
```

- `readChar()` で1文字ずつ進む
- `NextToken()` で次のトークンを返す
- `peekChar()` で先読み（`==` や `!=` の判定に使用）

---

# トークン定義

```go
type Token struct {
    Type    TokenType
    Literal string
}
```

キーワード判定の仕組み:

```go
var keywords = map[string]TokenType{
    "fn":     FUNCTION,
    "let":    LET,
    "true":   TRUE,
    "if":     IF,
    "return": RETURN,
    // ...
}

func LookupIdent(ident string) TokenType {
    if tok, ok := keywords[ident]; ok {
        return tok  // キーワードならその型を返す
    }
    return IDENT     // それ以外は識別子
}
```

---

# 第2章：Parser & AST（構文解析）

---

# なぜトークン化だけでは足りないのか

トークン列は**単語が並んでいるだけ**

```
[INT:5] [PLUS] [INT:10] [ASTERISK] [INT:2]
```

**問題**: `5 + 10 * 2` はどっち？
- `(5 + 10) * 2 = 30` ？
- `5 + (10 * 2) = 25` ？

→ トークンの並び順だけでは**関係性・優先順位**がわからない

---

# AST（抽象構文木）

トークン列を**木構造**に変換すると関係性が明確になる

```
  5 + 10 * 2

        [+]           ← 木のルート
       /   \
     [5]   [*]        ← 掛け算が先に結合
          /   \
        [10]  [2]
```

下から評価: `10 * 2 = 20` → `5 + 20 = 25`

---

# ASTの定義（Goのインターフェース）

```go
type Node interface {          // 全ノードの基本
    TokenLiteral() string
}

type Statement interface {     // 文（let, return）
    Node
    statementNode()
}

type Expression interface {    // 式（5+3, x）
    Node
    ExpressionNode()
}

type Program struct {          // プログラム全体
    Statements []Statement
}
```

---

# let文のパース

`let x = 5 + 10;` → 期待するパターンに沿って解析

```
let    x    =    5 + 10    ;
 ↓     ↓    ↓      ↓      ↓
今ここ 変数名 =   式の塊  省略可
       が来る が来る が来る
       はず   はず   はず
```

```go
func (p *Parser) parseLetStatement() *ast.LetStatement {
    // 1. 次は変数名が来るはず → 確認
    // 2. 次は = が来るはず → 確認
    // 3. 次は式が来るはず → parseExpression()
    // 4. セミコロンがあれば読み飛ばす
}
```

---

# 再帰下降構文解析（Pratt Parser）

**トップダウン** + **再帰** でパースする方式

```
ParseProgram()           ← プログラム全体
  ↓
parseStatement()         ← 文を解析
  ↓
parseLetStatement()      ← let文を解析
  ↓
parseExpression()        ← 式を解析
  ↓
parseInfixExpression()   ← 中置式（+）を解析
  ↓
parseExpression()        ← また式を解析（再帰！）
```

**大きな構造から細部へ「下降」していく**

---

# 式の判断方法

トークンの種類に対応した処理関数を登録

| 種類 | 意味 | 例 |
|------|------|-----|
| **prefix** | 式の**先頭**に来るもの | `5`, `x`, `-5`, `!true`, `if` |
| **infix** | 式の**途中**に来る演算子 | `+`, `-`, `*`, `/`, `==` |

演算子の**優先順位（precedence）**で正しい木構造を構築

```
1 + 2 * 3  →  * の方が優先度高い → 先に 2*3 を処理
```

---

# 第3章：Evaluator（評価器）

---

# 評価とは

**ASTを木の下から順に計算して値を求めること**

```
InfixExpression{5, +, 10}
       ↓ Eval
Integer{15}
```

- `Eval` 関数がASTノードの種類ごとに処理を振り分け
- **再帰的**に子ノードを評価
- 結果を `Object` として返す

---

# Eval関数の核心

```go
func Eval(node ast.Node, env *object.Environment) object.Object {
    switch node := node.(type) {

    case *ast.IntegerLiteral:
        return &object.Integer{Value: node.Value}

    case *ast.InfixExpression:
        left := Eval(node.Left, env)     // 左を再帰評価
        right := Eval(node.Right, env)   // 右を再帰評価
        return evalInfixExpression(node.Operator, left, right)

    case *ast.LetStatement:
        val := Eval(node.Value, env)     // 値を評価
        env.Set(node.Name.Value, val)    // 環境に保存
    }
}
```

---

# 環境（Environment）

**変数名と値の対応表**

```go
type Environment struct {
    store map[string]Object
    outer *Environment  // 外側のスコープ
}
```

```
let x = 5;
let y = 10;

env = {
    "x": Integer{5},
    "y": Integer{10},
}
```

`outer` でスコープのネスト（クロージャ）を実現

---

# `let x = 5 + 10;` 全処理の流れ

```
"let x = 5 + 10;"

    ↓ Lexer

[LET][x][=][5][+][10][;]

    ↓ Parser

LetStatement { Name: "x", Value: InfixExpr{5, +, 10} }

    ↓ Evaluator

1. LetStatement を見る
2. Value を評価 → 5 + 10 = 15
3. 環境に x = 15 を保存

    ↓ 完了！

env = { "x": Integer{15} }
```

---

# 現在の進捗

| 章 | 内容 | 状態 |
|---|------|------|
| 第1章 | Lexer / Token | 実装完了 |
| 第2章 | Parser / AST | 構造定義・Parser骨格まで |
| 第3章 | Evaluator / Object | 概要理解済み |

**実装済みパッケージ**: `token`, `lexer`, `repl`, `ast`(一部), `parser`(骨格)

---

# 学んだこと - コンピュータが動く仕組み

人間が書いた文字列が**段階的に変換**されて実行される

```
"let x = 5 + 10;"   ← 人間が読める
       ↓ Lexer
[LET][x][=][5][+][10][;]   ← 意味のある単位に分解
       ↓ Parser
LetStatement{x, Add{5,10}}  ← 構造として理解
       ↓ Evaluator
env = { "x": 15 }   ← 計算して結果を得る
```

コンピュータは「魔法」ではなく、**小さな変換の積み重ね**で動いている

---

# 学んだこと - CS基礎の組み合わせ

各パッケージが**1つの責務**を持ち、組み合わさって動く

| パッケージ | 責務 | CS概念 |
|-----------|------|--------|
| `token` | 定義だけ | データ型の設計 |
| `lexer` | 文字 → トークン | 字句解析（有限オートマトン） |
| `ast` | 木構造の定義 | データ構造（木） |
| `parser` | トークン → AST | 構文解析（再帰下降） |
| `evaluator` | AST → 実行 | 木の走査（再帰） |

**それぞれは単純。組み合わせることで「言語」になる。**

---

# 学んだこと - テスト駆動開発

本書のスタイル：**テストを先に書く → 実装 → テスト通す**

```go
// 例：Lexerのテスト（先にこれを書く）
func TestNextToken(t *testing.T) {
    input := `let five = 5;`
    expected := []token.Token{
        {Type: token.LET, Literal: "let"},
        {Type: token.IDENT, Literal: "five"},
        // ...
    }
    // 期待する結果を定義してから実装に取りかかる
}
```

- テストが**仕様書の代わり**になる
- 実装の正しさをいつでも確認できる安心感

---

# 学んだこと - Go言語の知識

- **インターフェース**: 異なる型を統一的に扱う仕組み（Node, Statement, Expression）
- **ポインタ（`*`）**: 状態の共有とコピー回避のために必要
- **パッケージ構造**: 責務ごとにコードを分離する設計
- **`switch ... .(type)`**: 型で処理を振り分けるGoの型アサーション
- **インタプリタ vs コンパイラ**: ASTまでは共通、その後の処理が異なる

---

# 聞きたいこと - 技術面

- インタプリタの「Lexer → Parser → Evaluator」という段階的な設計は、
  実際のプロダクト開発でもよく出てくるパターンですか？
- Speeeの開発現場で「木構造」や「再帰」が活きた場面はありますか？
  （例：HTML/DOM処理、設定パーサー、ルーティングなど）

---

# 聞きたいこと - 開発プラクティス

- この本はテストを先に書いて実装する進め方ですが、
  VPoEとして理想的なテスト文化をチームにどう根付かせていますか？
- 「責務を分けてパッケージに切る」設計判断について、
  実務ではどのタイミング・粒度で分けるのが良いですか？

---

# 聞きたいこと - キャリア・学び方

- CS基礎（データ構造・アルゴリズム・言語処理系など）を学んだことが
  エンジニアとしてのキャリアで「効いた」と感じた瞬間はありますか？
- 新しい技術領域を学ぶとき、VPoEの立場から見て
  「写経」「読書」「アウトプット」のバランスはどう取るのがおすすめですか？

---

# ありがとうございました

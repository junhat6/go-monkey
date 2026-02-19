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
## 振り返り会 - 2026/02/19

---

# 今日のアジェンダ

1. **この取り組みの目的**
2. **書籍の概要** - 何を作っているのか
3. **第1章** - Lexer（字句解析器）
4. **第2章** - Parser & AST（構文解析）
5. **第3章** - Evaluator（評価器）
6. **第4章** - インタプリタの拡張
7. **付録** - マクロシステム
8. **学んだこと・気づき**
9. **聞きたいこと**

---

# この取り組みの目的

| ゴール | 4章までの関連 |
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
- 全4章構成（全章実装完了！）

---

# Monkey言語でできること

```javascript
let age = 1;
let name = "Monkey";
let result = 10 * (20 / 2);

let add = fn(a, b) { a + b; };
add(1, 2);

// 4章で追加された機能
let arr = [1, "hello", fn(x) { x * 2 }];
arr[0];  // => 1

let people = {"name": "Monkey", "age": 1};
people["name"];  // => Monkey

len("Hello");  // => 5
push(arr, 4);  // => [1, "hello", fn, 4]

// 付録で追加：マクロで新しい構文を定義
let unless = macro(cond, cons, alt) {
    quote(if (!(unquote(cond))) { unquote(cons) } else { unquote(alt) });
};
unless(10 > 5, puts("small"), puts("big"));
```

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
│ Macro Expand │  付録：マクロ定義の抽出 → マクロ呼び出しの展開
│（マクロ展開） │
└──────────────┘
       ↓
┌──────────────┐
│  Evaluator   │  第3章：AST → 実行結果
│   （評価）    │  第4章：文字列/配列/ハッシュ/組み込み関数を追加
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

# オブジェクトシステム

ASTノード（構文）と評価結果（値）は**別の型**で表す

```go
type Object interface {
    Type() ObjectType
    Inspect() string
}
```

| Object型 | 役割 | 例 |
|----------|------|-----|
| `Integer` | 整数値 | `42` |
| `Boolean` | 真偽値 | `true` / `false` |
| `Null` | 値なし | `null` |
| `ReturnValue` | return文の値をラップ | `return 5;` → 5を包む |
| `Error` | エラー情報 | `"unknown operator: -BOOLEAN"` |
| `Function` | 関数（引数 + 本体 + 環境） | `fn(x) { x + 1 }` |

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

# 環境（Environment）とクロージャ

**変数名と値の対応表** + **外側のスコープへの参照**

```go
type Environment struct {
    store map[string]Object
    outer *Environment  // 外側のスコープ
}
```

```javascript
let newAdder = fn(x) {
    fn(y) { x + y };   // ← 外側の x を覚えている！
};
let addTwo = newAdder(2);
addTwo(3);  // => 5
```

関数が**定義時の環境を閉じ込める** = **クロージャ**
`outer` のチェーンで外側の変数を参照できる

---

# エラー伝播の仕組み

エラーは**Errorオブジェクトとして値の世界で伝播**する

```go
func Eval(node ast.Node, env *object.Environment) object.Object {
    // ...
    case *ast.InfixExpression:
        left := Eval(node.Left, env)
        if isError(left) { return left }    // エラーなら即座に返す
        right := Eval(node.Right, env)
        if isError(right) { return right }  // エラーなら即座に返す
        return evalInfixExpression(node.Operator, left, right)
}
```

例外機構（try/catch）を使わず、**戻り値でエラーを返す** = Go的な発想

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

# 第4章：インタプリタの拡張

---

# 4章の全体像

3章までで**整数・真偽値・関数**が動く基盤ができた
4章ではその基盤の上に**新しいデータ型**と**組み込み関数**を追加

| 追加要素 | 内容 |
|---------|------|
| 文字列（String） | `"hello"` + 連結演算子 `+` |
| 配列（Array） | `[1, 2, 3]` + インデックスアクセス `arr[0]` |
| ハッシュ（Hash） | `{"key": "value"}` + キーアクセス |
| 組み込み関数 | `len`, `first`, `last`, `rest`, `push`, `puts` |

**既存のアーキテクチャを壊さず、各層に少しずつ追加していく**

---

# 文字列型の追加

**Lexer → Parser → Object → Evaluator の全層に変更が必要**

```
"hello world"
  ↓ Lexer:  readString() で " から " まで読む
  ↓ Parser: StringLiteral ノードを生成
  ↓ Eval:   object.String{Value: "hello world"} を返す
```

文字列の連結:
```javascript
"Hello" + " " + "World"  // => "Hello World"
```

evalInfixExpression に**文字列同士の `+` 演算**を追加するだけ

---

# 配列型の追加

```javascript
let arr = [1, 2 * 3, fn(x){ x + 1 }];
arr[0];   // => 1
arr[1];   // => 6
arr[2](5) // => 6（関数も要素にできる！）
```

**新しい優先順位 INDEX が必要**（最も高い優先順位）

```
LOWEST < EQUALS < LESSGREATER < SUM < PRODUCT < PREFIX < CALL < INDEX
```

`myArray[0]` は中置式: 左辺 `myArray` + 演算子 `[` + 右辺 `0`

---

# parseExpressionList のリファクタリング

3章の `parseCallArguments` と4章の配列パースは**ほぼ同じ処理**

```
fn(a, b, c)   ← カンマ区切りの式リスト、終端は ")"
[1, 2, 3]     ← カンマ区切りの式リスト、終端は "]"
```

→ **終端トークンだけが違う**ので汎用化:

```go
// 3章: parseCallArguments() → ")" 固定
// 4章: parseExpressionList(end TokenType) → 終端を引数で受け取る
```

「共通パターンを見つけて汎用化する」リファクタリングの好例

---

# ハッシュ型の追加

```javascript
let people = {"name": "Monkey", "age": 1, true: "yes"};
people["name"];  // => Monkey
people["age"];   // => 1
people[true];    // => yes
```

**キーに使える型を制限する** → `Hashable` インターフェース

```go
type Hashable interface {
    HashKey() HashKey    // ハッシュ値を返せる型だけキーにできる
}
```

Integer, Boolean, String だけが `Hashable` を実装
→ 配列や関数はキーにできない（コンパイル時にチェック）

---

# HashKey の設計

Go の map でキーとして使うために**比較可能な構造体**が必要

```go
type HashKey struct {
    Type  ObjectType
    Value uint64
}
```

| 型 | HashKey の計算方法 |
|---|---|
| Integer | 値そのまま `uint64(value)` |
| Boolean | true → 1, false → 0 |
| String | FNV-1a ハッシュ関数で計算 |

**同じ内容 → 同じ HashKey** を保証することが重要

---

# 組み込み関数（Built-in Functions）

ユーザーが定義しなくても使える関数

| 関数 | 用途 | 例 |
|------|------|-----|
| `len` | 長さを返す | `len("abc")` → 3, `len([1,2])` → 2 |
| `first` | 最初の要素 | `first([1,2,3])` → 1 |
| `last` | 最後の要素 | `last([1,2,3])` → 3 |
| `rest` | 先頭以外 | `rest([1,2,3])` → [2,3] |
| `push` | 末尾に追加 | `push([1,2], 3)` → [1,2,3] |
| `puts` | 出力 | `puts("hello")` → hello |

---

# イミュータブルな操作

`push` と `rest` は**元の配列を変更しない**

```javascript
let a = [1, 2, 3];
let b = push(a, 4);
// a は [1, 2, 3] のまま！（変わらない）
// b は [1, 2, 3, 4]（新しい配列）
```

```go
// push の実装: 新しいスライスを作ってコピー
newElements := make([]object.Object, length+1)
copy(newElements, arr.Elements)
newElements[length] = args[1]
return &object.Array{Elements: newElements}
```

**関数型プログラミングの考え方**: データを壊さず、新しいデータを作る

---

# 組み込み関数で再帰を活用

`push`/`rest`/`first` を組み合わせると**ループなしで配列処理**ができる

```javascript
let map = fn(arr, f) {
    if (len(arr) == 0) { return []; }
    let first_elem = first(arr);
    let rest_arr = rest(arr);
    push(map(rest_arr, f), f(first_elem));
};

map([1, 2, 3], fn(x) { x * 2 });  // => [2, 4, 6]
```

Monkey言語にはfor/whileループがない → **再帰で繰り返しを表現**

---

# 4章の設計思想：拡張の容易さ

新しいデータ型を追加するときの変更箇所:

```
1. token/token.go    → 新しいトークン定義を追加
2. lexer/lexer.go    → 新しい文字の読み取り処理
3. ast/ast.go        → 新しいASTノード型を定義
4. parser/parser.go  → prefix/infix 関数を登録
5. object/object.go  → 新しいObjectを定義
6. evaluator/        → Eval に case を追加
```

**各層が疎結合なので、既存コードを壊さず拡張できる**
→ これが3章までに作った基盤の強さ

---

# 付録：マクロシステム

---

# unless を実装したい

`if` の反対 — **条件が偽のときに実行**する制御構造

```javascript
unless(10 > 5, puts("not greater"), puts("greater"));
// 10 > 5 は真 → "greater" だけ出力されてほしい
```

これを Monkey 言語に追加するには？

---

# 方法1: Go側（インタプリタ本体）に組み込む

`if` と同じように、Go のコードを**4ファイル改修して再コンパイル**

```
1. token.go      UNLESS トークンを追加
2. ast.go        UnlessExpression ノードを追加
3. parser.go     parseUnlessExpression を追加
4. evaluator.go  UnlessExpression の評価ロジックを追加
```

正しく動くが、`unless` を追加したいだけで**インタプリタを書き換える**のは大げさ
→ Monkeyのユーザーが自由に新しい構文を追加できない

---

# 方法2: Monkey の関数として実装する（問題あり）

```javascript
let unless = fn(condition, consequence, alternative) {
    if (!condition) { consequence } else { alternative }
};

unless(true, puts("A"), puts("B"));
```

一見うまくいきそうだが、**関数呼び出しの評価順序**が問題になる

---

# 関数の引数は「渡す前に全部評価される」

```javascript
unless(true, puts("A"), puts("B"));
```

```
Step 1: 引数1を評価  →  true              OK
Step 2: 引数2を評価  →  puts("A") を実行  → 画面に "A" が出る
Step 3: 引数3を評価  →  puts("B") を実行  → 画面に "B" が出る
Step 4: やっと関数本体に入る
        if (!true) { ... } else { ... }
```

Step 2〜3 の時点で**両方とも実行済み** → 本体で分岐しても手遅れ
これは Monkey に限らず、**ほとんどの言語の関数呼び出しがこの順序**で動く

---

# 方法3: マクロシステム

Go側を改修せず、かつ引数の先行評価問題も起きない第3の方法

**マクロは「評価の前にコードを書き換える」フェーズを挟む**

```
Parser → [マクロ展開] → Evaluator
              ↑
         ここでASTを書き換える
         （まだ何も実行されていない）
```

マクロ展開の時点ではプログラムはまだ「コードの構造（AST）」でしかない
→ `puts("A")` も `puts("B")` もただの木のノード
→ 自由に配置し直せる。その後で評価器が動いて、初めて実行される

---

# 3つの方法の比較

| 方法 | 動作 | 代償 |
|------|------|------|
| **Go側に組み込み** | 正しく動く | Go 4ファイル改修 + 再コンパイル |
| **Monkey の関数** | 引数が先に全部評価される → **壊れる** | — |
| **マクロ** | 正しく動く | **Monkeyコードだけで完結** |

→ マクロなら**インタプリタを触らず**、Monkeyのユーザーが新しい制御構造を追加できる

---

# マクロとは何か

**「評価の前にコードを書き換える仕組み」**

|  | 関数 | マクロ |
|--|------|--------|
| 引数 | **評価してから**値を渡す | **ASTのまま**渡す |
| 戻り値 | 値（Object） | AST（Quote経由） |
| 実行タイミング | 評価時 | 評価**前**（コード書き換え） |
| できること | 値の計算 | **構文の変換** |

---

# マクロシステムの3つの柱

```
1. quote / unquote     コードをデータとして扱う
2. Modify              ASTを再帰的に走査・変換する汎用エンジン
3. DefineMacros /      マクロ定義の抽出と呼び出しの展開
   ExpandMacros
```

実装の変更箇所:

| レイヤー | 変更内容 |
|---------|---------|
| token | `MACRO` キーワード追加 |
| ast | `MacroLiteral` ノード + `Modify` 関数（新規） |
| parser | `parseMacroLiteral` 追加 |
| object | `Quote`, `Macro` オブジェクト追加 |
| evaluator | quote/unquote 処理 + `macro_expansion.go`（新規） |
| repl | マクロ展開パイプライン追加 |

---

# 柱1: quote / unquote — コードをデータとして扱う

通常 `1 + 2` は `3` に評価されるが、`quote` で囲むと**ASTのまま保持**

```javascript
quote(1 + 2)           // => QUOTE((1 + 2))  ← 評価されない！
quote(foobar + barfoo) // => QUOTE((foobar + barfoo))
```

`unquote` は quote の中で**「ここだけは評価して」**と指定する脱出口

```javascript
quote(unquote(4 + 4))                // => QUOTE(8)
quote(8 + unquote(4 + 4))            // => QUOTE((8 + 8))
let foobar = 8;
quote(unquote(foobar))               // => QUOTE(8)
```

`unquote` の部分だけ評価され、結果がASTノードに変換されて埋め込まれる

---

# quote の実装

`CallExpression` の評価で `quote` を**特別扱い**する

```go
// evaluator.go — Eval関数内
case *ast.CallExpression:
    if node.Function.TokenLiteral() == "quote" {
        return quote(node.Arguments[0], env) // 引数を評価しない！
    }
    // 通常の関数呼び出し...
```

```go
func quote(node ast.Node, env *object.Environment) object.Object {
    node = evalUnquoteCalls(node, env) // unquote()だけ先に処理
    return &object.Quote{Node: node}   // ASTノードをそのまま返す
}
```

ポイント: `quote` は組み込み関数ではなく**構文レベルの特別扱い**
（引数を評価しないので、通常の関数では実現できない）

---

# 柱2: Modify — AST変換の汎用エンジン

全ASTノード型を**再帰的に走査**して、各ノードに変換関数を適用

```go
func Modify(node Node, modifier ModifierFunc) Node {
    switch node := node.(type) {
    case *InfixExpression:
        node.Left = Modify(node.Left, modifier)   // 左を先に変換
        node.Right = Modify(node.Right, modifier)  // 右を先に変換
    case *IfExpression:
        node.Condition = Modify(node.Condition, modifier)
        // ...
    }
    return modifier(node)  // 最後に自分自身を変換（ボトムアップ）
}
```

**quote/unquote でもマクロ展開でも、この Modify が中核**
「ASTのどこかに条件を満たすノードがあったら置換する」パターンを汎用化

---

# 柱3: DefineMacros / ExpandMacros

**定義フェーズ** — ASTからマクロ定義を抽出して環境に格納

```javascript
let unless = macro(condition, consequence, alternative) {
    quote(if (!(unquote(condition))) {
        unquote(consequence);
    } else {
        unquote(alternative);
    });
};
```

→ `unless` が `Macro` オブジェクトとしてマクロ環境に登録される
→ この `let` 文は**ASTから削除**される（評価器には渡さない）

---

# マクロ展開の流れ

**展開フェーズ** — マクロ呼び出しを展開後のASTに置換

```javascript
unless(10 > 5, puts("not greater"), puts("greater"));
```

```
1. ast.Modify で CallExpression を走査
2. "unless" がマクロ環境にある → マクロ呼び出しと判定
3. 引数を評価せず Quote に包む:
     condition   = Quote(10 > 5)
     consequence = Quote(puts("not greater"))
     alternative = Quote(puts("greater"))
4. マクロ本体を評価 → unquote が引数のASTを埋め込む
5. 結果のASTで元の呼び出し式を置換
```

展開結果:
```javascript
if (!(10 > 5)) { puts("not greater") } else { puts("greater") }
```

---

# マクロ版 unless — 評価順序を追跡

```javascript
unless(true, puts("A"), puts("B"));
```

```
Step 1: マクロ展開（評価の前に起きる。何も実行しない）
        unless(true, puts("A"), puts("B"))
        ↓ ASTを組み替えるだけ
        if (!(true)) { puts("A") } else { puts("B") }

Step 2: 展開後のASTを評価器が実行
        !(true) → false
        → else 側だけ実行 → puts("B") → 画面に "B" だけ出る
```

関数版との違い: `puts("A")` は**コードの断片として移動しただけ**で、
`if` の条件が偽だから**一度も実行されない**

---

# マクロ展開のパイプライン（REPL）

```go
func Start(in io.Reader, out io.Writer) {
    env := object.NewEnvironment()
    macroEnv := object.NewEnvironment() // ← 付録で追加

    for {
        // ... 入力を読み取る ...
        program := p.ParseProgram()

        // マクロ定義を抽出し、呼び出しを展開する（付録で追加）
        evaluator.DefineMacros(program, macroEnv)
        expanded := evaluator.ExpandMacros(program, macroEnv)

        // 展開後のASTを評価する
        evaluated := evaluator.Eval(expanded, env)
    }
}
```

パーサーと評価器の間に**マクロ展開フェーズ**を挟むだけ
→ 評価器はマクロの存在を知らなくてよい（疎結合）

---

# 現在の進捗

| 章 | 内容 | 状態 |
|---|------|------|
| 第1章 | Lexer / Token | 実装完了 |
| 第2章 | Parser / AST | 実装完了 |
| 第3章 | Evaluator / Object / Environment | 実装完了 |
| 第4章 | String / Array / Hash / Builtins | 実装完了 |
| 付録 | マクロシステム（quote/unquote/Modify/展開） | 実装完了 |

**全テスト通過**: `go test ./...` OK

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
| `object` | 値の表現 | 型システム・ハッシュ |
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

# 学んだこと - 拡張しやすい設計

3章で作った基盤が4章の拡張で活きた

- **Objectインターフェース**: `Type()` と `Inspect()` さえ実装すれば新しい型を追加できる
- **Pratt Parserの関数登録**: prefix/infix に関数を登録するだけで新構文に対応
- **Evalのswitch文**: case を追加するだけで新しいノードを評価できる

→ **インターフェースによる拡張性**がGoの強み

4章を通じて「3章の設計が良かったから簡単に拡張できた」と体感

---

# 学んだこと - メタプログラミングの力

付録のマクロシステムで「**言語のユーザーが言語自体を拡張できる**」ことを体感

- `unless` のような**新しい制御構造**をMonkeyコードだけで追加できる
  （Go側のコードを触らずに！）
- 関数では不可能なこと（引数の遅延評価）がマクロなら可能
- **コードとデータの境界が曖昧になる** — Lispの哲学

マクロ展開を「パーサーと評価器の間」に挟む設計は
**既存の処理パイプラインを壊さずに新しいフェーズを追加**する好例

---

# 学んだこと - Go言語の知識

- **インターフェース**: 異なる型を統一的に扱う仕組み（Node, Statement, Expression, Object, Hashable）
- **ポインタ（`*`）**: 状態の共有とコピー回避のために必要
- **パッケージ構造**: 責務ごとにコードを分離する設計
- **`switch ... .(type)`**: 型で処理を振り分けるGoの型アサーション
- **可変長引数 `...`**: 組み込み関数の `func(args ...object.Object)` で活用
- **`copy` と `make`**: イミュータブルな操作でスライスを安全にコピー

---

# 学んだこと - 関数型プログラミングの考え方

Monkey言語には**ループ構文がない**

```javascript
// for/while の代わりに再帰 + 組み込み関数
let reduce = fn(arr, initial, f) {
    if (len(arr) == 0) { return initial; }
    reduce(rest(arr), f(initial, first(arr)), f);
};

let sum = fn(arr) {
    reduce(arr, 0, fn(acc, el) { acc + el });
};

sum([1, 2, 3, 4, 5]);  // => 15
```

- **イミュータブルなデータ操作**（push/rest は元を壊さない）
- **第一級関数**（関数を引数に渡せる、変数に入れられる）

---

# 聞きたいこと - アーキテクチャ

- 4章で「各層が疎結合だから拡張が容易」を体感しましたが、
  実務で**拡張しやすい設計**を意識するポイントはありますか？
- Goのインターフェースによる拡張（Objectに新しい型を追加するだけ）は
  実際のプロダクトでもよく使うパターンですか？

---

# 聞きたいこと - 技術面

- インタプリタの「Lexer → Parser → Evaluator」という段階的な設計は、
  実際のプロダクト開発でもよく出てくるパターンですか？
- Speeeの開発現場で「木構造」や「再帰」が活きた場面はありますか？
  （例：HTML/DOM処理、設定パーサー、ルーティングなど）
- ハッシュのキー設計（Hashableインターフェース）のように、
  「使えるものを型で制限する」設計は実務でも有効ですか？

---

# 聞きたいこと - 開発プラクティス

- この本はテストを先に書いて実装する進め方ですが、
  VPoEとして理想的なテスト文化をチームにどう根付かせていますか？
- 「責務を分けてパッケージに切る」設計判断について、
  実務ではどのタイミング・粒度で分けるのが良いですか？
- 4章の`parseExpressionList`のようなリファクタリングのタイミングは
  どう判断していますか？（先にやる vs 必要になってからやる）

---

# 聞きたいこと - キャリア・学び方

- CS基礎（データ構造・アルゴリズム・言語処理系など）を学んだことが
  エンジニアとしてのキャリアで「効いた」と感じた瞬間はありますか？
- 新しい技術領域を学ぶとき、VPoEの立場から見て
  「写経」「読書」「アウトプット」のバランスはどう取るのがおすすめですか？

---

# ありがとうございました

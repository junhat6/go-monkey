# Writing An Interpreter In Go (WAIIG) - 概要

## 書籍情報
- **原題**: Writing An Interpreter In Go
- **著者**: Thorsten Ball
- **バージョン**: 1.7 (2020年5月7日リリース)
- **公式サイト**: https://interpreterbook.com
- **続編**: Writing A Compiler In Go (https://compilerbook.com)

## Monkey言語の特徴
Monkeyはこの本で実装するプログラミング言語。

### データ型
- Integer (整数)
- Boolean (真偽値)
- String (文字列)
- Array (配列)
- Hash (ハッシュマップ)
- Function (関数)
- Null

### 言語機能
- `let`による変数束縛
- 前置/中置演算子
- `if/else`条件分岐
- `return`文
- 第一級関数 (関数も値として扱える)
- 高階関数
- クロージャ
- 再帰
- 配列/ハッシュのインデックスアクセス

### 構文例
```monkey
let age = 1;
let name = "Monkey";
let result = 10 * (20 / 2);

let myArray = [1, 2, 3, 4, 5];
let myHash = {"name": "Thorsten", "age": 28};

let add = fn(a, b) { a + b; };
add(1, 2);

let fibonacci = fn(x) {
  if (x == 0) { 0 }
  else { if (x == 1) { 1 }
  else { fibonacci(x - 1) + fibonacci(x - 2); }}
};

let map = fn(arr, f) {
  let iter = fn(arr, accumulated) {
    if (len(arr) == 0) { accumulated }
    else { iter(rest(arr), push(accumulated, f(first(arr)))); }
  };
  iter(arr, []);
};
```

## ボーナス章 (The Lost Chapter)
Elixirスタイルのマクロシステムを実装。コード生成機能を追加。

## 他言語への移植
読者によってRust, Elixir, C++, TypeScript, Python, Java, Swift, Kotlin等多数の言語に移植されている。

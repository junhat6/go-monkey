# WAIIG vs Crafting Interpreters 比較

## 書籍比較

| 項目 | Writing An Interpreter In Go | Crafting Interpreters |
|------|------------------------------|----------------------|
| 著者 | Thorsten Ball | Robert Nystrom |
| 言語 | Go | Java + C |
| 対象言語 | Monkey | Lox |
| ページ数 | ~200 | ~800 |
| アプローチ | Tree-walking interpreter | Tree-walking + Bytecode VM |
| 無料 | 有料 | Web版無料 |

## 実装アプローチの違い

### WAIIG (Monkey)
- **Tree-walking interpreter**: ASTを直接再帰評価
- シンプルで理解しやすい
- 実行速度は遅め
- Go言語の簡潔さを活かした実装

### Crafting Interpreters (Lox)
- **Part I (jlox)**: Javaで Tree-walking interpreter
- **Part II (clox)**: Cで Bytecode VM
- より本格的だが複雑

## 共通する概念

### パーサー
両方とも **Pratt Parser** (Top-Down Operator Precedence) を使用。

### Visitor Pattern vs Type Switch
```java
// Crafting Interpreters (Java) - Visitor Pattern
class Interpreter implements Expr.Visitor<Object> {
    @Override
    public Object visitBinaryExpr(Expr.Binary expr) {
        Object left = evaluate(expr.left);
        Object right = evaluate(expr.right);
        // ...
    }
}
```

```go
// WAIIG (Go) - Type Switch
func Eval(node ast.Node, env *object.Environment) object.Object {
    switch node := node.(type) {
    case *ast.InfixExpression:
        left := Eval(node.Left, env)
        right := Eval(node.Right, env)
        // ...
    }
}
```

### 環境 (Environment)
両方ともスコープを **リンクリスト的な構造** で管理:
- 外側のスコープへの参照を持つ
- クロージャで「定義時の環境」を保持

## 学習の進め方

1. **WAIIG**を先に読む (入門として最適)
2. 続編 **Writing A Compiler In Go** でVMコンパイラを学ぶ
3. **Crafting Interpreters** でより深い理解を得る

## 参考リンク
- WAIIG: https://interpreterbook.com
- Crafting Interpreters: https://craftinginterpreters.com

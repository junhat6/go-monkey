# Object System & Evaluator 詳細

## Object System (object/object.go)

### 基本インターフェース
```go
type Object interface {
    Type() ObjectType
    Inspect() string
}
```

### ObjectType一覧
```go
const (
    NULL_OBJ         = "NULL"
    ERROR_OBJ        = "ERROR"
    INTEGER_OBJ      = "INTEGER"
    BOOLEAN_OBJ      = "BOOLEAN"
    STRING_OBJ       = "STRING"
    RETURN_VALUE_OBJ = "RETURN_VALUE"
    FUNCTION_OBJ     = "FUNCTION"
    BUILTIN_OBJ      = "BUILTIN"
    ARRAY_OBJ        = "ARRAY"
    HASH_OBJ         = "HASH"
)
```

### 主要Object型
```go
type Integer struct { Value int64 }
type Boolean struct { Value bool }
type String struct { Value string }
type Null struct{}
type Error struct { Message string }
type ReturnValue struct { Value Object }
type Array struct { Elements []Object }
type Hash struct { Pairs map[HashKey]HashPair }

type Function struct {
    Parameters []*ast.Identifier
    Body       *ast.BlockStatement
    Env        *Environment  // クロージャのための環境
}

type Builtin struct {
    Fn BuiltinFunction
}
```

### Hashable インターフェース
ハッシュのキーとして使える型:
```go
type Hashable interface {
    HashKey() HashKey
}

type HashKey struct {
    Type  ObjectType
    Value uint64
}
```
Integer, Boolean, Stringが実装。

## Environment (object/environment.go)

変数のスコープを管理:
```go
type Environment struct {
    store map[string]Object
    outer *Environment  // 外側のスコープ (クロージャ用)
}

func NewEnvironment() *Environment
func NewEnclosedEnvironment(outer *Environment) *Environment
func (e *Environment) Get(name string) (Object, bool)
func (e *Environment) Set(name string, val Object) Object
```

## Evaluator (evaluator/evaluator.go)

### 主要関数
```go
func Eval(node ast.Node, env *object.Environment) object.Object
```

### シングルトン値 (最適化)
```go
var (
    NULL  = &object.Null{}
    TRUE  = &object.Boolean{Value: true}
    FALSE = &object.Boolean{Value: false}
)
```

### 評価ロジック (型スイッチ)
```go
func Eval(node ast.Node, env *object.Environment) object.Object {
    switch node := node.(type) {
    case *ast.Program:
        return evalProgram(node, env)
    case *ast.IntegerLiteral:
        return &object.Integer{Value: node.Value}
    case *ast.Boolean:
        return nativeBoolToBooleanObject(node.Value)
    case *ast.PrefixExpression:
        right := Eval(node.Right, env)
        return evalPrefixExpression(node.Operator, right)
    case *ast.InfixExpression:
        left := Eval(node.Left, env)
        right := Eval(node.Right, env)
        return evalInfixExpression(node.Operator, left, right)
    case *ast.IfExpression:
        return evalIfExpression(node, env)
    case *ast.FunctionLiteral:
        return &object.Function{Parameters: node.Parameters, Env: env, Body: node.Body}
    case *ast.CallExpression:
        function := Eval(node.Function, env)
        args := evalExpressions(node.Arguments, env)
        return applyFunction(function, args)
    // ... その他
    }
}
```

### 関数適用 (クロージャのポイント)
```go
func applyFunction(fn object.Object, args []object.Object) object.Object {
    switch fn := fn.(type) {
    case *object.Function:
        // 関数定義時の環境を拡張 (クロージャ)
        extendedEnv := extendFunctionEnv(fn, args)
        evaluated := Eval(fn.Body, extendedEnv)
        return unwrapReturnValue(evaluated)
    case *object.Builtin:
        return fn.Fn(args...)
    }
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
    env := object.NewEnclosedEnvironment(fn.Env)  // fn.Envが定義時の環境
    for paramIdx, param := range fn.Parameters {
        env.Set(param.Value, args[paramIdx])
    }
    return env
}
```

### Truthiness判定
```go
func isTruthy(obj object.Object) bool {
    switch obj {
    case NULL:
        return false
    case TRUE:
        return true
    case FALSE:
        return false
    default:
        return true  // その他は全てtruthy
    }
}
```

## 組み込み関数 (evaluator/builtins.go)

| 関数 | 説明 | 例 |
|------|------|-----|
| `len` | 配列/文字列の長さ | `len("hello")` → `5` |
| `first` | 配列の最初の要素 | `first([1, 2, 3])` → `1` |
| `last` | 配列の最後の要素 | `last([1, 2, 3])` → `3` |
| `rest` | 最初以外の要素 | `rest([1, 2, 3])` → `[2, 3]` |
| `push` | 配列に要素追加 | `push([1, 2], 3)` → `[1, 2, 3]` |
| `puts` | 出力 | `puts("hello")` |

### 組み込み関数の実装例
```go
var builtins = map[string]*object.Builtin{
    "len": &object.Builtin{
        Fn: func(args ...object.Object) object.Object {
            if len(args) != 1 {
                return newError("wrong number of arguments")
            }
            switch arg := args[0].(type) {
            case *object.Array:
                return &object.Integer{Value: int64(len(arg.Elements))}
            case *object.String:
                return &object.Integer{Value: int64(len(arg.Value))}
            }
            return newError("argument to `len` not supported")
        },
    },
}
```

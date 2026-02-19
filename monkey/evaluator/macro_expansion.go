// macro_expansion.go はマクロの定義と展開を行う。
// パーサーと評価器の間に位置し、ASTレベルでマクロを処理する。
//
// DefineMacros: プログラムからマクロ定義（let ... = macro(...)）を抽出して
//   環境に格納し、元のASTからマクロ定義文を削除する。
// ExpandMacros: ast.Modify を使ってマクロ呼び出しを見つけ、
//   マクロ本体を評価した結果のASTノードで置換する。
//
// 付録で追加。
package evaluator

import (
	"monkey/ast"
	"monkey/object"
)

// DefineMacros はプログラムからマクロ定義を抽出して環境に格納する。
// マクロ定義文はASTから削除される（通常の評価器には渡さない）。
func DefineMacros(program *ast.Program, env *object.Environment) {
	definitions := []int{}

	for i, statement := range program.Statements {
		if isMacroDefinition(statement) {
			addMacro(statement, env)
			definitions = append(definitions, i)
		}
	}

	// マクロ定義文をASTから削除（後ろから削除してインデックスがずれないようにする）
	for i := len(definitions) - 1; i >= 0; i = i - 1 {
		definitionIndex := definitions[i]
		program.Statements = append(
			program.Statements[:definitionIndex],
			program.Statements[definitionIndex+1:]...,
		)
	}
}

// isMacroDefinition は文がマクロ定義（let <name> = macro(...) { ... }）か判定する。
func isMacroDefinition(node ast.Statement) bool {
	letStatement, ok := node.(*ast.LetStatement)
	if !ok {
		return false
	}

	_, ok = letStatement.Value.(*ast.MacroLiteral)
	if !ok {
		return false
	}

	return true
}

// addMacro はマクロ定義文からMacroオブジェクトを生成して環境に格納する。
func addMacro(stmt ast.Statement, env *object.Environment) {
	letStatement, _ := stmt.(*ast.LetStatement)
	macroLiteral, _ := letStatement.Value.(*ast.MacroLiteral)

	macro := &object.Macro{
		Parameters: macroLiteral.Parameters,
		Env:        env,
		Body:       macroLiteral.Body,
	}

	env.Set(letStatement.Name.Value, macro)
}

// ExpandMacros はASTを走査してマクロ呼び出しを展開する。
// マクロ呼び出しの引数はQuoteオブジェクトとしてマクロに渡され、
// マクロ本体を評価した結果のASTノードで呼び出し式が置換される。
func ExpandMacros(program ast.Node, env *object.Environment) ast.Node {
	return ast.Modify(program, func(node ast.Node) ast.Node {
		callExpression, ok := node.(*ast.CallExpression)
		if !ok {
			return node
		}

		macro, ok := isMacroCall(callExpression, env)
		if !ok {
			return node
		}

		args := quoteArgs(callExpression)
		evalEnv := extendMacroEnv(macro, args)

		evaluated := Eval(macro.Body, evalEnv)

		quote, ok := evaluated.(*object.Quote)
		if !ok {
			panic("we only support returning AST-nodes from macros")
		}

		return quote.Node
	})
}

// isMacroCall は関数呼び出しがマクロ呼び出しかどうか判定する。
// 呼び出す関数が識別子で、その識別子が環境でMacroオブジェクトに束縛されていればマクロ呼び出し。
func isMacroCall(
	exp *ast.CallExpression,
	env *object.Environment,
) (*object.Macro, bool) {
	identifier, ok := exp.Function.(*ast.Identifier)
	if !ok {
		return nil, false
	}

	obj, ok := env.Get(identifier.Value)
	if !ok {
		return nil, false
	}

	macro, ok := obj.(*object.Macro)
	if !ok {
		return nil, false
	}

	return macro, true
}

// quoteArgs はマクロ呼び出しの引数をQuoteオブジェクトに変換する。
// マクロの引数は評価されずにASTノードとしてそのまま渡される。
func quoteArgs(exp *ast.CallExpression) []*object.Quote {
	args := []*object.Quote{}

	for _, a := range exp.Arguments {
		args = append(args, &object.Quote{Node: a})
	}

	return args
}

// extendMacroEnv はマクロ呼び出し用の環境を作成する。
// マクロのパラメータにQuoteオブジェクト（未評価の引数AST）を束縛する。
func extendMacroEnv(
	macro *object.Macro,
	args []*object.Quote,
) *object.Environment {
	extended := object.NewEnclosedEnvironment(macro.Env)

	for paramIdx, param := range macro.Parameters {
		extended.Set(param.Value, args[paramIdx])
	}

	return extended
}

// Package repl は Monkey言語のREPL（Read-Eval-Print Loop）を実装するパッケージ。
// ユーザーが入力したコードを字句解析 → 構文解析 → 評価し、結果を表示する。
package repl

import (
	"bufio"
	"fmt"
	"io"
	"monkey/evaluator"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
)

// PROMPT はREPLのプロンプト文字列。
const PROMPT = ">> "

// Start はREPLを起動する。
// 入力ストリームからコードを1行ずつ読み取り、評価結果を出力ストリームに書き出す。
// 環境（env）をループ全体で共有することで、変数束縛がセッション中持続する。
//
// 付録で追加: マクロ環境（macroEnv）を追加し、パーサーと評価器の間に
// マクロ定義・展開ステップを挟む。
// パイプライン: Parser → DefineMacros → ExpandMacros → Evaluator
func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	// 環境をループの外で作成し、変数をセッション間で保持する
	env := object.NewEnvironment()
	// マクロ環境もセッション全体で保持する（付録で追加）
	macroEnv := object.NewEnvironment()

	for {
		fmt.Fprintf(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		// パーサーエラーがあればモンキーのAAと共に表示
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		// マクロ定義を抽出し、マクロ呼び出しを展開する（付録で追加）
		evaluator.DefineMacros(program, macroEnv)
		expanded := evaluator.ExpandMacros(program, macroEnv)

		// 展開後のASTを評価器に渡して実行結果を得る
		evaluated := evaluator.Eval(expanded, env)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}
}

// MONKEY_FACE はパーサーエラー時に表示されるモンキーのアスキーアート。
const MONKEY_FACE = `            __,__
   .--.  .-"     "-.  .--.
  / .. \/  .-. .-.  \/ .. \
 | |  '|  /   Y   \  |'  | |
 | \   \  \ 0 | 0 /  /   / |
  \ '- ,\.-"""""""-./, -' /
   ''-' /_   ^ ^   _\ '-''
       |  \._   _./  |
       \   \ '~' /   /
        '._ '-=-' _.'
           '-----'
`

// printParserErrors はパーサーエラーをモンキーのAAと共に出力する。
func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, MONKEY_FACE)
	io.WriteString(out, "Woops! We ran into some monkey business here!\n")
	io.WriteString(out, " parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}

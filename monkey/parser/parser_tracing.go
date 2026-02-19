// parser_tracing.go はパーサーのデバッグ用トレーシング機能を提供する。
// パーサーの動作を追跡したい場合に、各解析関数の入口と出口でログを出力する。
// 通常のパース処理では使用しない。
package parser

import (
	"fmt"
	"strings"
)

// traceLevel はトレースのインデントレベル。
// ネストが深くなるほどインデントが増える。
var traceLevel int = 0

const traceIdentPlaceholder string = "\t"

// identLevel は現在のトレースレベルに応じたインデント文字列を返す。
func identLevel() string {
	return strings.Repeat(traceIdentPlaceholder, traceLevel-1)
}

// tracePrint はインデント付きでメッセージを出力する。
func tracePrint(fs string) {
	fmt.Printf("%s%s\n", identLevel(), fs)
}

func incIdent() { traceLevel = traceLevel + 1 }
func decIdent() { traceLevel = traceLevel - 1 }

// trace は解析関数の入口で呼ぶ。"BEGIN <msg>" を出力してインデントを増やす。
func trace(msg string) string {
	incIdent()
	tracePrint("BEGIN " + msg)
	return msg
}

// untrace は解析関数の出口で呼ぶ。"END <msg>" を出力してインデントを減らす。
func untrace(msg string) {
	tracePrint("END " + msg)
	decIdent()
}

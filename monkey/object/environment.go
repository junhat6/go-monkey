// environment.go は変数の環境（スコープ）を管理する。
// Environment は変数名から値へのマッピングを持ち、
// outer フィールドで外側のスコープへのチェーンを形成する。
// これにより、レキシカルスコープ（静的スコープ）とクロージャが実現される。
package object

// NewEnclosedEnvironment は外側の環境を持つ新しい環境を作成する。
// 関数呼び出し時に使用し、関数の定義時環境を outer として設定する。
// これにより関数内から外側の変数にアクセスできる（クロージャ）。
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// NewEnvironment は新しい空の環境を作成する。
// プログラムのトップレベル環境として使用する。
func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil}
}

// Environment は変数のスコープを表す構造体。
// store は現在のスコープの変数を保持し、
// outer は外側のスコープへの参照（なければnil）。
type Environment struct {
	store map[string]Object
	outer *Environment
}

// Get は変数名から値を検索する。
// 現在のスコープになければ外側のスコープを再帰的に探す。
// 見つかれば (値, true)、見つからなければ (nil, false) を返す。
func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

// Set は変数を現在のスコープに設定する。
func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

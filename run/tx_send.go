package main

import "github.com/davecgh/go-spew/spew"

type State map[string]int

func (s State) Copy() State {
	return s
}

func main() {
	state := State(map[string]int{"0": 0})
	state["1"] = 1

	copy := state.Copy()

	copy["0"] = 2

	spew.Dump(state)
	spew.Dump(copy)
}

package main

import (
	"os"

	"github.com/spf13/cast"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("need question id!")
	}
	id := cast.ToInt(args[1])
	q := NewQuestion(id)
	if err := q.Build(); err != nil {
		panic(err)
	}
	f := NewQuestionFile(q)
	if err := f.Create(); err != nil {
		panic(err)
	}
}

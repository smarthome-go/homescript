package main

import "github.com/MikMuellerDev/homescript/homescript"

func main() {
	err := homescript.Run(homescript.DummyExecutor{}, "print('hello world')")
	if err != nil {
		panic(err[0].Error())
	}
}

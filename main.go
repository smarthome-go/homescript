package main

import "github.com/MikMuellerDev/homescript/homescript"

func main() {
	_, err := homescript.Run(homescript.DummyExecutor{}, "print('hello world')")
	if err != nil {
		panic(err.Error())
	}
}

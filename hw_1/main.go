package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	name := os.Getenv("USER")
	if name == "" {
		fmt.Fprintln(os.Stderr, "Переменная окружения USER не задана или пуста")
		os.Exit(1)
	}

	fmt.Println(name)

	for _, arg := range os.Args[1:] {
		fmt.Println(arg)
	}

	fmt.Println("Версия Go:", runtime.Version())
}

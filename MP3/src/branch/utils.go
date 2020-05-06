package main

import "fmt"

func check(e error) {
	if e != nil {
		fmt.Println("Error Detected:")
		panic(e)
	}
}

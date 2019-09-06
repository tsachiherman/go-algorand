package main

// #cgo CFLAGS: -E pthread.c
// #include <stdio.h>
// void inCFile() {
//     return;
// }
import "C"
import "fmt"

func main() {
	fmt.Println("I am in Go code now!")
	C.inCFile()
}

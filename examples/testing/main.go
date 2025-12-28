package main

import "fmt"

// This file exists to allow the examples/testing package to be built.
// The actual examples are in main_test.go and are run with `go test`.
//
// To run the test examples:
//
//	go test -v ./examples/testing/...
func main() {
	fmt.Println("Langfuse Testing Examples")
	fmt.Println("")
	fmt.Println("This package contains test examples demonstrating how to test")
	fmt.Println("code that uses the Langfuse Go SDK.")
	fmt.Println("")
	fmt.Println("Run the examples with: go test -v ./examples/testing/...")
}

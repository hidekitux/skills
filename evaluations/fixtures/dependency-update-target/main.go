// Command greeter prints a greeting.
package main

import "fmt"

// Greet returns the greeting for name.
func Greet(name string) string {
	return "Hello, " + name
}

func main() {
	fmt.Println(Greet("world"))
}

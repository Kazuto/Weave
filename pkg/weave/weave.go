package weave

import "fmt"

// Hello returns a friendly greeting including the provided name.
func Hello(name string) string {
	if name == "" {
		name = "world"
	}
	return fmt.Sprintf("Hello, %s!", name)
}

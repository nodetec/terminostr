package main

import (
	"fmt"

	"github.com/charmbracelet/glamour"
)

func main() {
	var events = sub()
	for _, event := range events {
		out, _ := glamour.Render(event.Content, "dark")
		fmt.Print(out)
	}
}

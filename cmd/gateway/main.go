package main

import (
	"fmt"
	"os"

	"github.com/fabian4/gateway-homebrew-go/internal/version"
)

func main() {
	fmt.Printf("gateway-homebrew-go %s\n", version.Value)
	// TODO: load config → init router/proxy/l4 → start listeners
	os.Exit(0)
}

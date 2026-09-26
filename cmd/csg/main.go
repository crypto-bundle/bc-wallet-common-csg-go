// Package main — точка входа для утилиты проверки CSG.
package main

import (
	"fmt"
	"os"

	"github.com/crypto-bundle/bc-wallet-common-csg-go/pkg/csg"
)

func main() {
	if err := csg.(); err != nil {
		fmt.Fprintf(os.Stderr, "CSG validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("CSG validation passed: all required documents are present")
}

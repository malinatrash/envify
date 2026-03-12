package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/malinatrash/envify/internal/generator"
	"github.com/malinatrash/envify/internal/parser"
)

func main() {
	typeName := flag.String("type", "", "struct type name to parse (required)")
	output := flag.String("output", ".env", "output .env file path")
	dir := flag.String("dir", ".", "directory of the Go package to parse")
	flag.Parse()

	if *typeName == "" {
		fmt.Fprintln(os.Stderr, "Usage: envify -type=<StructName> [-output=.env] [-dir=.]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example go:generate directive:")
		fmt.Fprintln(os.Stderr, "  //go:generate envify -type=Conf")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// go generate sets GOFILE; use its directory if -dir not explicitly set
	if *dir == "." {
		if goFile := os.Getenv("GOFILE"); goFile != "" {
			*dir = "."
		}
	}

	entries, err := parser.Extract(*dir, *typeName)
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}

	if err := generator.Write(*output, *typeName, entries); err != nil {
		log.Fatalf("generate error: %v", err)
	}

	fmt.Printf("envify: wrote %d variables to %s\n", len(entries), *output)
}

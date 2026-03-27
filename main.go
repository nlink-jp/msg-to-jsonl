package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nlink-jp/lite-msg/internal/parser"
)

var version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	pretty := flag.Bool("pretty", false, "pretty-print JSON (default: JSONL)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lite-msg [flags] [file.msg | dir/ ...]\n\n")
		fmt.Fprintf(os.Stderr, "Parses Outlook MSG files and outputs structured JSONL to stdout.\n")
		fmt.Fprintf(os.Stderr, "Reads from stdin when no arguments are given.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	enc := newEncoder(os.Stdout, *pretty)
	hadError := false

	if flag.NArg() == 0 {
		if err := processStdin(enc); err != nil {
			fmt.Fprintf(os.Stderr, "error: stdin: %v\n", err)
			hadError = true
		}
	} else {
		for _, arg := range flag.Args() {
			if err := processArg(arg, enc); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
				hadError = true
			}
		}
	}

	if hadError {
		os.Exit(1)
	}
}

func processArg(arg string, enc *encoder) error {
	info, err := os.Stat(arg)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return processDir(arg, enc)
	}
	return processFile(arg, enc)
}

func processDir(dir string, enc *encoder) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.msg"))
	if err != nil {
		return err
	}
	hadError := false
	for _, path := range matches {
		if err := processFile(path, enc); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", path, err)
			hadError = true
		}
	}
	if hadError {
		return fmt.Errorf("one or more files in %s failed to parse", dir)
	}
	return nil
}

func processFile(path string, enc *encoder) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return processData(data, path, enc)
}

// processStdin reads all of stdin into memory (MSG requires random access).
func processStdin(enc *encoder) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("no input provided on stdin")
	}
	return processData(data, "stdin", enc)
}

func processData(data []byte, source string, enc *encoder) error {
	email, err := parser.Parse(data, source)
	if err != nil {
		return err
	}
	return enc.write(email)
}

type encoder struct {
	w      io.Writer
	pretty bool
}

func newEncoder(w io.Writer, pretty bool) *encoder {
	return &encoder{w: w, pretty: pretty}
}

func (e *encoder) write(v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if e.pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return err
	}
	_, err := e.w.Write(buf.Bytes())
	return err
}

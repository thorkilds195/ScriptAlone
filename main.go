package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"
)

type ImportPackage struct {
	name             string
	shortName        string
	packagePath      string
	isFunctionImport bool
	functions        []string
}

func parseWord(s string) string {
	// Parses from the first character in string to the next whitespace character
	a := make([]byte, 0)
	for i := range s {
		if unicode.IsSpace(rune(s[i])) {
			break
		}
		a = append(a, s[i])
	}
	return string(a)
}

func differenceListSet(l1 []string, l2 []string) []string {
	// Returns the diff between the first set and the second
	out := make([]string, 0)
	for i := range l1 {
		for j := range l2 {
			if l1[i] == l2[j] {
				break
			}
		}
		out = append(out, l1[i])
	}
	return out
}

func parseFunction(s string) string {
	// Parses from the first character in string to the next starting paranthesis
	a := make([]byte, 0)
	for i := range s {
		if s[i] == '(' {
			break
		}
		a = append(a, s[i])
	}
	return string(a)
}

func removeWhitespace(s string) string {
	out := make([]byte, 0)
	for i := range s {
		if !unicode.IsSpace(rune(s[i])) {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func parseImportFunctions(s string) []string {
	funcs := strings.Split(s, ",")
	for i := range funcs {
		funcs[i] = removeWhitespace(funcs[i])
	}
	return funcs
}

func isInList(s string, l []string) bool {
	for _, v := range l {
		if s == v {
			return true
		}
	}
	return false
}

func parsePackagesFunctions(s string, packages []*ImportPackage) {
	for p := range packages {
		i := 0
		for i < len(packages[p].shortName) {
			i = strings.Index(s[i:], packages[p].shortName)
			if i == -1 {
				break
			}
			next_function := parseFunction(s[i+len(packages[p].shortName)+1:])
			if !isInList(next_function, packages[p].functions) {
				packages[p].functions = append(packages[p].functions, next_function)
			}
			i += len(packages[p].shortName)
		}
	}
}

func isAlreadyPackage(shortName string, packages []*ImportPackage) (bool, *ImportPackage) {
	for p := range packages {
		if packages[p].shortName == shortName {
			return true, packages[p]
		}
	}
	return false, nil
}

func addImportPackage(s string, packages *[]*ImportPackage) {
	isFunctionImport := false
	var functions []string
	var shortName, name string
	if s[:4] == "from" {
		isFunctionImport = true
		name = parseWord(s[5:])
		shortName = name
		// Parse functions here
		start_idx := strings.Index(s, "import")
		functions = parseImportFunctions(s[start_idx+len("import"):])
	} else {
		// If it is not a function import type package, then we need to search the rest of the text for any occurences of the named package to know which functions to import
		start := "import"
		name = parseWord(s[len(start)+1:])
		search_from := len(start) + len(name) + 2
		if strings.Contains(s, "as") {
			shortName = parseWord(s[search_from+3:])
		} else {
			shortName = name
		}
	}
	if ok, ptr := isAlreadyPackage(shortName, *packages); ok {
		functions = differenceListSet(functions, ptr.functions)
		ptr.functions = append(ptr.functions, functions...)
	} else {
		this_package := ImportPackage{name: name, shortName: shortName,
			packagePath: "Test", isFunctionImport: isFunctionImport, functions: functions}
		*packages = append(*packages, &this_package)
	}
}

func findAllImports(path string) ([]*ImportPackage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var importPackages []*ImportPackage
	var line string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "import") {
			addImportPackage(line, &importPackages)
		} else {
			parsePackagesFunctions(line, importPackages)
		}
	}
	return importPackages, scanner.Err()
}

func main() {
	args := os.Args
	path := args[1]
	arr, err := findAllImports(path)
	if err != nil {
		log.Fatal("Something went wrong")
	}
	fmt.Println(arr)
}

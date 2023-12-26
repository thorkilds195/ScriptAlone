package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type ImportPackage struct {
	name             string   // the real name of the package
	shortName        string   // the imported name into the relevant module
	packagePath      string   // the path to the file that holds the package
	isFunctionImport bool     // is it imported as from module import function or not
	functions        []string // holds the name of the functions that is used from the package
	lines            []int    // holds the position of the lines that the import package appears in to facilitate easier search when copying later
}

func (p *ImportPackage) importDependencies(toFile string) error {
	file, err := os.Open(p.packagePath)
	if err != nil {
		return err
	}
	defer file.Close()
	outFile, err := os.OpenFile(toFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer outFile.Close()
	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(outFile)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 2 && line[:3] == "def" {
			function_name := parseFunctionName(line[4:])
			for _, v := range p.functions {
				if v == function_name {
					def := parseFunctionDefinition(v, scanner, line)
					writeFunctionDefinition(def, writer)
					break
				}
			}
		}
	}
	err = writer.Flush()
	if err != nil {
		return err
	}
	return nil
}

func importAllDependencies(packages []*ImportPackage, writePath string) {
	for i := range packages {
		packages[i].importDependencies(writePath)
	}
}

func parseFunctionDefinition(name string, scanner *bufio.Scanner, defLine string) []string {
	out := make([]string, 1)
	out[0] = defLine
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !unicode.IsSpace(rune(line[0])) {
			break
		}
		out = append(out, line)
	}
	return out
}

func writeFunctionDefinition(def []string, writer *bufio.Writer) {
	for _, v := range def {
		_, err := writer.WriteString(v + "\n")
		if err != nil {
			panic(err)
		}
	}
	_, err := writer.WriteString("\n")
	if err != nil {
		panic(err)
	}
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

func parseFunctionName(s string) string {
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

func parsePackagesFunctions(s string, packages []*ImportPackage, lineNumber int) {
	for p := range packages {
		i := 0
		for i < len(packages[p].shortName) {
			i = strings.Index(s[i:], packages[p].shortName)
			if i == -1 {
				break
			}
			next_function := parseFunctionName(s[i+len(packages[p].shortName)+1:])
			if !isInList(next_function, packages[p].functions) {
				packages[p].functions = append(packages[p].functions, next_function)
			}
			packages[p].lines = append(packages[p].lines, lineNumber)
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

func addImportPackage(s string, packages *[]*ImportPackage, parentPath string) {
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
			packagePath:      filepath.Join(filepath.Dir(parentPath), name+".py"),
			isFunctionImport: isFunctionImport, functions: functions}
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
	counter := 0
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "import") {
			addImportPackage(line, &importPackages, path)
		} else {
			parsePackagesFunctions(line, importPackages, counter)
		}
		counter++
	}
	return importPackages, scanner.Err()
}

func createOutFile(writePath string) {
	dest, err := os.Create(writePath)
	if err != nil {
		panic(err)
	}
	defer dest.Close()
}

func (p *ImportPackage) isLineInLines(lineNumber int) bool {
	for i := range p.lines {
		if lineNumber == p.lines[i] {
			return true
		}
	}
	return false
}

func replacePackageNames(lineNumber int, s string, packages []*ImportPackage) string {
	for p := range packages {
		if packages[p].isLineInLines(lineNumber) {
			s = strings.ReplaceAll(s, packages[p].shortName+".", "")
		}
	}
	return s
}

func copyOriginalFile(path string, outPath string, packages []*ImportPackage) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	outFile, err := os.OpenFile(outPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer outFile.Close()
	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(outFile)
	counter := 0
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "import") {
			line := replacePackageNames(counter, line, packages)
			_, err := writer.WriteString(line + "\n")
			if err != nil {
				return err
			}
		}
		counter++
	}
	err = writer.Flush()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	args := os.Args
	path := args[1]
	writePath := "C:\\Users\\thomast\\PycharmProjects\\ScriptAlone\\out.py"
	packages, err := findAllImports(path)
	if err != nil {
		panic("Something went wrong")
	}
	createOutFile(writePath)
	// Need to be able to flag the dependiencies that should not be overwritten in the future
	// right now it will just overwrite and look for all dependiencies that exist
	// I think we could just flag the dependencies to not overwrite at the start
	// and then import those in a function here before the necessary dependency functions are imported
	// and then file just import all the code from the passed in file after those other functions have been imported
	importAllDependencies(packages, writePath)
	err = copyOriginalFile(path, writePath, packages)
	if err != nil {
		panic("Something went wrong")
	}
}

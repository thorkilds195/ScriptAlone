package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type ImportPackage struct {
	name             string           // the real name of the package
	shortName        string           // the imported name into the relevant module
	packagePath      string           // the path to the file that holds the package
	isFunctionImport bool             // is it imported as from module import function or not
	functions        []string         // holds the name of the functions that is used from the package
	lines            []int            // holds the position of the lines that the import package appears in to facilitate easier search when copying later
	childImports     []*ImportPackage // a pointer to the next import packages in case of cascading imports
	parent           *ImportPackage   // FATHER
}

func (p *ImportPackage) importDependencies(toFile string, writtenDependencies *[]string) error {
	// For now, written dependencies holds an overview all previously written dpoendecies
	// This is to make sure we dont write the function definition for the same dependecy twice
	// This is defined as the same function name, but ideally should be defined as the same function name from the same package
	// In those cases were we have the same function name from the same package we might want to add some unique function identifier to the front of the package
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
				if v == function_name && !hasFunctionBeenWrittenBefore(function_name, writtenDependencies) {
					*writtenDependencies = append(*writtenDependencies, function_name)
					def := parseFunctionDefinition(v, scanner, line)
					writeFunctionDefinition(def, writer, p)
					break
				}
			}
		}
	}
	err = writer.Flush()
	if err != nil {
		return err
	}
	for i := range p.childImports {
		p.childImports[i].importDependencies(toFile, writtenDependencies)
	}
	return nil
}

func importAllDependencies(packages []*ImportPackage, writePath string) {
	writtenDependencies := make([]string, 0)
	for i := range packages {
		packages[i].importDependencies(writePath, &writtenDependencies)
	}
}

func hasFunctionBeenWrittenBefore(function_name string, previous_functions *[]string) bool {
	return isInList(function_name, *previous_functions)
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

func writeFunctionDefinition(def []string, writer *bufio.Writer, p *ImportPackage) {
	for _, v := range def {
		s := replacePackageNames(-1, v, p.childImports)
		_, err := writer.WriteString(s + "\n")
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

func parsePackagesFunctions(s string, packages []*ImportPackage, lineNumber int, calling_function *string) {
	for p := range packages {
		i := 0
		if len(s) > 2 && s[:3] == "def" {
			*calling_function = parseFunctionName(s[4:])
			// TODO: This calling function thing needs to be done better
			// The idea is that we need to know which function the inner function definition is called from to understand whether we should include the function in case of cascading imports from functions
			// For example it would be no point in importing a function from a package which is not used in the main script we are interested in
		}
		for i < len(packages[p].shortName) {
			i = strings.Index(s[i:], packages[p].shortName)
			if i == -1 {
				break
			}
			next_function := parseFunctionName(s[i+len(packages[p].shortName)+1:])
			if !isInList(next_function, packages[p].functions) {
				if packages[p].parent == nil || isInList(*calling_function, packages[p].parent.functions) {
					packages[p].functions = append(packages[p].functions, next_function)
				}
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

func addImportPackage(s string, packages *[]*ImportPackage, parentPath string, parent *ImportPackage) {
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
			isFunctionImport: isFunctionImport, functions: functions, parent: parent}
		*packages = append(*packages, &this_package)
	}
}

func findFileImports(path string, parent *ImportPackage) ([]*ImportPackage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var importPackages []*ImportPackage
	var line, calling_function string
	scanner := bufio.NewScanner(file)
	counter := 0
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "import") {
			addImportPackage(line, &importPackages, path, parent)
		} else {
			parsePackagesFunctions(line, importPackages, counter, &calling_function)
		}
		counter++
	}
	return importPackages, scanner.Err()
}

func (p *ImportPackage) findChildImports() error {
	// Call this recursively until we have exhausted the amount of imports
	packages, err := findFileImports(p.packagePath, p)
	if len(packages) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	// Adds the parent and child relations for these packages here right now
	// Need to refactor this later so it is in the same package creation step
	p.childImports = packages
	for i := range packages {
		packages[i].findChildImports()
	}
	return nil
}

func findAllImports(path string) ([]*ImportPackage, error) {
	// Starts by finding the main file's file imports
	packages, err := findFileImports(path, nil)
	if err != nil {
		return nil, err
	}
	// Then find any subsequent imports that is necessary for the main file
	// This alters the original packages returned from the above function
	for i := range packages {
		packages[i].findChildImports()
	}
	return packages, nil
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
		if lineNumber == -1 || packages[p].isLineInLines(lineNumber) {
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

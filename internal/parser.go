package internal

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// GetStructNames reads all .go files from modelsDir and returns the names
// of all top-level struct types found, using the DST (dave/dst) library.
func GetStructNames(modelsDir string) ([]string, error) {
	var structs []string
	fileSet := token.NewFileSet()

	files, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(modelsDir, file.Name())
		// Using decorator.ParseFile from the DST library instead of parser.ParseFile
		fileDst, err := decorator.ParseFile(fileSet, filePath, nil, parser.AllErrors)
		if err != nil {
			return nil, err
		}

		// Traverse the DST to find struct types
		for _, decl := range fileDst.Decls {
			genDecl, ok := decl.(*dst.GenDecl)
			if !ok {
				continue
			}
			// We only care about type declarations
			if genDecl.Tok != token.TYPE {
				continue
			}

			// Iterate over all type specs in the GenDecl
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*dst.TypeSpec)
				if !ok {
					continue
				}

				// Check whether the type is a StructType
				if _, ok := typeSpec.Type.(*dst.StructType); ok && typeSpec.Name != nil {
					structs = append(structs, typeSpec.Name.Name)
				}
			}
		}
	}

	return structs, nil
}

package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func GetStructNames(modelsDir string) ([]string, error) {
	var structs []string
	fset := token.NewFileSet()

	files, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(modelsDir, file.Name())
		fileAst, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors)
		if err != nil {
			return nil, err
		}

		for _, decl := range fileAst.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); ok {
					if typeSpec.Name != nil {
						structs = append(structs, typeSpec.Name.Name)
					}
				}
			}
		}
	}
	return structs, nil

}

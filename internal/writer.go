package internal

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func ImplementServiceStruct(modelName string, file *ast.File, reimplement bool) {
	serviceName := modelName + "Service"
	isServiceStructDefined := false
	var insertPos int
	var decls []ast.Decl

	for i, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		decls = append(decls, decl)
		if genDecl.Tok == token.IMPORT {
			insertPos = i + 1
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
				if typeSpec.Name != nil && typeSpec.Name.Name == serviceName {
					isServiceStructDefined = true
					if reimplement {
						decls = decls[:1]
					}
				}
			}
		}
	}

	if isServiceStructDefined && !reimplement {
		return
	}

	serviceStruct := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(serviceName),
				Type: &ast.StructType{
					Fields: &ast.FieldList{},
				},
			},
		},
	}

	file.Decls = append(decls[:insertPos], append([]ast.Decl{serviceStruct}, decls[insertPos:]...)...)
}

func ImplementModelAlias(modelName string, file *ast.File) {
	isAliasDefined := false
	aliasTypeStandard := fmt.Sprintf("models.%s", CapitalizeFirst(modelName))
	var insertPos int
	var decls []ast.Decl

	for i, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		decls = append(decls, decl)
		if genDecl.Tok == token.IMPORT {
			insertPos = i + 1
			continue
		}

		if genDecl.Tok != token.TYPE {
			continue
		}
		if len(genDecl.Specs) != 1 {
			continue
		}

		if typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec); ok {
			if typeSpec.Name != nil && typeSpec.Name.Name == modelName {
				if linkedType, ok := typeSpec.Type.(*ast.SelectorExpr); ok {
					pkg, ok := linkedType.X.(*ast.Ident)
					if !ok {
						log.Printf("Defined alias `%s` with unknown type", typeSpec.Name)
						decls = decls[:1]
						continue
					}

					if linkedType.Sel == nil {
						log.Printf("Defined alias `%s` with unknown type", typeSpec.Name)
						decls = decls[:1]
						continue
					}

					if pkg.Name != "models" {
						log.Printf(
							"Defined alias `%s` with wrong type: package `%s`, should be `models`",
							typeSpec.Name,
							pkg,
						)
						decls = decls[:1]
						continue
					}

					if linkedType.Sel.Name != modelName {
						log.Printf(
							"Defined alias `%s` with wrong type: models.%s, should be `%s`",
							typeSpec.Name,
							linkedType.Sel.Name,
							aliasTypeStandard,
						)
						decls = decls[:1]
						continue
					}
					isAliasDefined = true
				} else {
					log.Printf("Defined alias %s with unknown type", typeSpec.Name)
					decls = decls[:1]
				}
			}
		}
	}

	typeAlias := ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{
					Name: modelName,
				},
				Type: &ast.Ident{
					Name: aliasTypeStandard,
				},
			},
		},
	}

	if !isAliasDefined {
		file.Decls = append(decls[:insertPos], append([]ast.Decl{&typeAlias}, decls[insertPos:]...)...)
	}
}

func importExists(fset *token.FileSet, file *ast.File, importPath string) bool {
	for _, group := range astutil.Imports(fset, file) {
		for _, imp := range group {
			if imp.Name == nil && imp.Path.Value == `"`+importPath+`"` {
				return true
			}
		}
	}
	return false
}

func MaintainImports(fileSet *token.FileSet, file *ast.File) error {
	for _, importPath := range ServiceImports {
		if !importExists(fileSet, file, importPath) {
			if !astutil.AddImport(fileSet, file, importPath) {
				err := fmt.Sprintf("%s: Failed to add import: %s", fileSet.Position(file.Pos()), importPath)
				return errors.New(err)
			}
		}
	}
	return nil
}

func ImplementMethods(structName string, methodName string, template string, node ast.Node) {

}

func CreateServiceFileIfNotExists(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		// file wasn't created
		f, err := os.Create(filePath)
		if err != nil {
			log.Fatalf("Failed to create file: %s", filePath)
			return err
		}

		_, err = f.Write([]byte("package services\n"))
		if err != nil {
			log.Fatalf("Failed to write file: %s", filePath)
			return err
		}

		defer f.Close()
	}
	return nil
}

func ImplementService(mainPkgPath string, modelName string, reimplement bool) error {
	serviceRelativePath := fmt.Sprintf("services/%s.go", strings.ToLower(modelName))
	filePath := filepath.Join(mainPkgPath, serviceRelativePath)

	err := CreateServiceFileIfNotExists(filePath)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	serviceFile, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Parsing error: %s: %v", mainPkgPath, err)
		return err
	}

	err = MaintainImports(fset, serviceFile)
	if err != nil {
		return err
	}
	ImplementModelAlias(modelName, serviceFile)
	ImplementServiceStruct(modelName, serviceFile, reimplement)

	file, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Error occured to open `%s` service file: %v", modelName, err)
		return err
	}

	err = printer.Fprint(file, fset, serviceFile)
	if err != nil {
		log.Fatalf("Error occurred to writing changes in `%s` service file: %v", modelName, err)
		return err
	}

	defer file.Close()

	return nil
}

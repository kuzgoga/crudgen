package internal

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"html/template"
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
				Assign: 1, // quick and dirty
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

func GenerateCrudMethodCode(methodName string, context CrudTemplatesContext) string {
	templateCode, templateExists := RawTemplates[methodName]
	if !templateExists {
		panic(fmt.Sprintf("Template doesn't exist: %s \n", methodName))
	}
	var buffer bytes.Buffer
	templateCode = "package services\n\n" + templateCode
	tmpl, err := template.New("CrudMethodTemplate").Parse(templateCode)
	if err != nil {
		panic(err)
	}
	if err := tmpl.Execute(&buffer, context); err != nil {
		panic(err)
	}
	return buffer.String()
}

func MethodCodeToDeclaration(methodCode string) (ast.FuncDecl, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "src.go", methodCode, parser.SkipObjectResolution)
	if err != nil {
		return ast.FuncDecl{}, err
	}

	methodDecl := ast.FuncDecl{}
	ast.Inspect(file, func(node ast.Node) bool {
		funcDecl, ok := node.(*ast.FuncDecl)
		if !ok {
			return true
		}
		methodDecl = *funcDecl
		return false
	})

	return methodDecl, nil
}

func ImplementCrudMethods(modelName string, serviceName string, file *ast.File, reimplement bool) error {
	templateContext := CrudTemplatesContext{
		ServiceName:  serviceName,
		EntityType:   modelName,
		EntityPlural: ToPlural(modelName),
	}

	for _, methodName := range []string{CreateMethod, GetAllMethod, GetByIdMethod, UpdateMethod, CountMethod} {
		methodCode := GenerateCrudMethodCode(methodName, templateContext)
		methodDecl, err := MethodCodeToDeclaration(methodCode)
		if err != nil {
			fmt.Println(methodDecl)
			panic(err)
		}
		err = ImplementMethod(file, &methodDecl, reimplement)
		if err != nil {
			return err
		}
	}

	return nil
}

func ImplementMethod(file *ast.File, methodDecl *ast.FuncDecl, reimplement bool) error {
	var decls []ast.Decl
	methodImplemented := false

	methodStructure := methodDecl.Recv.List[0].Names[0].Name
	methodName := methodDecl.Name.Name
	log.Printf("Standard method structure: %s\n", methodStructure)
	log.Printf("Standard method name: %s\n", methodName)

	for _, decl := range file.Decls {
		decls = append(decls, decl)
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if len(funcDecl.Recv.List) > 0 && len(funcDecl.Recv.List[0].Names) > 0 {
			fmt.Printf("Method structure: %s\n", funcDecl.Recv.List[0].Names[0].Name)
			fmt.Printf("Method name: %s\n", funcDecl.Name.Name)
			if funcDecl.Recv.List[0].Names[0].Name == methodStructure {
				if funcDecl.Name != nil && funcDecl.Name.Name == methodName {
					if methodImplemented {
						err := fmt.Sprintf("`%s` method redeclarated for struct `%s`", methodName, methodStructure)
						log.Println(err)
						return errors.New(err)
					} else {
						methodImplemented = true
					}
					if reimplement {
						decls = decls[:1]
					}
				}
			}
		}
	}

	if reimplement || !methodImplemented {
		file.Decls = append(decls, methodDecl)
	}

	// Reload AST: fix issues with syntax 'hallucinations'
	// Write AST
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		return err
	}

	// Load AST
	formattedFile, err := parser.ParseFile(token.NewFileSet(), "", buf.String(), parser.ParseComments)
	if err != nil {
		return err
	}

	*file = *formattedFile

	return nil
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
	serviceName := modelName + "Service"

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
	err = ImplementCrudMethods(modelName, serviceName, serviceFile, reimplement)

	if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error occured to open `%s` service file: %v", modelName, err))
	}

	err = printer.Fprint(file, fset, serviceFile)
	if err != nil {
		return errors.New(
			fmt.Sprintf("Error occurred to writing changes in `%s` service file: %v", modelName, err),
		)
	}

	defer file.Close()

	return nil
}

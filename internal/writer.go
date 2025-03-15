package internal

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func ImplementServiceStruct(modelName string, file *dst.File, reimplement bool) (wasModified bool) {
	serviceName := modelName + "Service"
	var (
		isServiceStructDefined bool
		insertPos              int
		decls                  []dst.Decl
	)

	for i, decl := range file.Decls {
		genDecl, ok := decl.(*dst.GenDecl)
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
			typeSpec, ok := spec.(*dst.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := typeSpec.Type.(*dst.StructType); ok {
				if typeSpec.Name != nil && typeSpec.Name.Name == serviceName {
					isServiceStructDefined = true
					if reimplement {
						decls = decls[:len(decls)-1]
					}
				}
			}
		}
	}

	if isServiceStructDefined && !reimplement {
		return false
	}

	serviceStruct := &dst.GenDecl{
		Tok: token.TYPE,
		Specs: []dst.Spec{
			&dst.TypeSpec{
				Name: dst.NewIdent(serviceName),
				Type: &dst.StructType{
					Fields: &dst.FieldList{},
				},
			},
		},
	}

	file.Decls = append(decls[:insertPos], append([]dst.Decl{serviceStruct}, decls[insertPos:]...)...)
	return true
}

func ImplementModelAlias(modelName string, file *dst.File) (wasModified bool) {
	aliasTypeStandard := fmt.Sprintf("models.%s", CapitalizeFirst(modelName))
	var (
		insertPos      int
		decls          []dst.Decl
		isAliasDefined bool
	)

	for i, decl := range file.Decls {
		genDecl, ok := decl.(*dst.GenDecl)
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

		if typeSpec, ok := genDecl.Specs[0].(*dst.TypeSpec); ok {
			if typeSpec.Name != nil && typeSpec.Name.Name == modelName {
				if linkedType, ok := typeSpec.Type.(*dst.SelectorExpr); ok {
					pkg, ok := linkedType.X.(*dst.Ident)
					if !ok {
						log.Printf("Defined alias `%s` with unknown type", typeSpec.Name)
						decls = decls[:len(decls)-1]
						continue
					}

					if linkedType.Sel == nil {
						log.Printf("Defined alias `%s` with unknown type", typeSpec.Name)
						decls = decls[:len(decls)-1]
						continue
					}

					if pkg.Name != "models" {
						log.Printf(
							"Defined alias `%s` with wrong type: package `%s`, should be `models`",
							typeSpec.Name,
							pkg,
						)
						decls = decls[:len(decls)-1]
						continue
					}

					if linkedType.Sel.Name != modelName {
						log.Printf(
							"Defined alias `%s` with wrong type: models.%s, should be `%s`",
							typeSpec.Name,
							linkedType.Sel.Name,
							aliasTypeStandard,
						)
						decls = decls[:len(decls)-1]
						continue
					}
					isAliasDefined = true
				} else {
					log.Printf("Defined alias %s with unknown type", typeSpec.Name)
					decls = decls[:len(decls)-1]
				}
			}
		}
	}

	typeAlias := dst.GenDecl{
		Tok: token.TYPE,
		Specs: []dst.Spec{
			&dst.TypeSpec{
				Name: &dst.Ident{
					Name: modelName,
				},
				Assign: true,
				Type: &dst.Ident{
					Name: aliasTypeStandard,
				},
			},
		},
	}

	if !isAliasDefined {
		file.Decls = append(decls[:insertPos], append([]dst.Decl{&typeAlias}, decls[insertPos:]...)...)
		return true
	} else {
		return false
	}
}

func importExists(file *dst.File, importPath string) bool {
	for _, imp := range file.Imports {
		if imp.Path.Value == `"`+importPath+`"` {
			return true
		}
	}
	return false
}

func MaintainImports(file *dst.File) (wasModified bool, e error) {
	var (
		modified   bool
		importDecl *dst.GenDecl
	)

	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importDecl = genDecl
			break
		}
	}

	if importDecl == nil {
		modified = true
		importDecl = &dst.GenDecl{
			Tok:   token.IMPORT,
			Specs: []dst.Spec{},
		}
		file.Decls = append([]dst.Decl{importDecl}, file.Decls...)
	}

	for _, importPath := range ServiceImports {
		if !importExists(file, importPath) {
			modified = true
			importDecl.Specs = append(importDecl.Specs, &dst.ImportSpec{
				Path: &dst.BasicLit{
					Kind:  token.STRING,
					Value: `"` + importPath + `"`,
				},
			})
		}
	}

	return modified, nil
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

func MethodCodeToDeclaration(methodCode string) (*dst.FuncDecl, error) {
	fset := token.NewFileSet()
	file, err := decorator.ParseFile(fset, "src.go", methodCode, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	var methodDecl *dst.FuncDecl
	dst.Inspect(file, func(node dst.Node) bool {
		funcDecl, ok := node.(*dst.FuncDecl)
		if !ok {
			return true
		}
		methodDecl = funcDecl
		return false
	})

	return methodDecl, nil
}

func ImplementCrudMethods(modelName string, serviceName string, file *dst.File, reimplement bool) (wasModified bool, e error) {
	templateContext := CrudTemplatesContext{
		ServiceName:  serviceName,
		EntityType:   modelName,
		EntityPlural: ToPlural(modelName),
	}
	modified := reimplement

	for _, methodName := range implementedMethods {
		methodCode := GenerateCrudMethodCode(methodName, templateContext)
		methodDecl, err := MethodCodeToDeclaration(methodCode)
		fmt.Printf("%s\n", methodCode)
		if err != nil {
			fmt.Println(methodDecl)
			panic(err)
		}

		methodModified, err := ImplementMethod(file, methodDecl, reimplement)
		if err != nil {
			return false, err
		}
		if methodModified {
			modified = true
		}
	}

	return modified, nil
}

func ImplementMethod(file *dst.File, methodDecl *dst.FuncDecl, reimplement bool) (wasModified bool, e error) {
	var (
		decls             []dst.Decl
		methodImplemented bool
	)
	modified := reimplement
	methodStructure := methodDecl.Recv.List[0].Names[0].Name
	methodName := methodDecl.Name.Name

	for _, decl := range file.Decls {
		decls = append(decls, decl)
		funcDecl, ok := decl.(*dst.FuncDecl)
		if !ok {
			continue
		}
		if len(funcDecl.Recv.List) > 0 && len(funcDecl.Recv.List[0].Names) > 0 {
			log.Printf("Method structure: %s\n", funcDecl.Recv.List[0].Names[0].Name)
			log.Printf("Method name: %s\n", funcDecl.Name.Name)
			if funcDecl.Recv.List[0].Names[0].Name == methodStructure {
				if funcDecl.Name != nil && funcDecl.Name.Name == methodName {
					if methodImplemented {
						err := fmt.Sprintf("`%s` method redeclarated for struct `%s`", methodName, methodStructure)
						log.Println(err)
						return false, errors.New(err)
					} else {
						methodImplemented = true
					}
					if reimplement {
						decls = decls[:len(decls)-1]
					}
				}
			}
		}
	}

	if reimplement || !methodImplemented {
		modified = true
		file.Decls = append(
			decls,
			&dst.FuncDecl{
				Recv: methodDecl.Recv,
				Name: methodDecl.Name,
				Type: methodDecl.Type,
				Body: &dst.BlockStmt{
					List: methodDecl.Body.List,
				},
			},
		)
	}

	return modified, nil
}

func CreateServiceFileIfNotExists(filePath string) (wasModified bool, e error) {
	var modified bool
	if _, err := os.Stat(filePath); err != nil {
		// file wasn't created
		f, err := os.Create(filePath)
		if err != nil {
			log.Fatalf("Failed to create file: %s", filePath)
			return false, err
		}

		_, err = f.Write([]byte("package services\n"))
		if err != nil {
			log.Fatalf("Failed to write file: %s", filePath)
			return false, err
		}
		modified = true

		defer f.Close()
	}
	return modified, nil
}

func ImplementService(mainPkgPath string, modelName string, reimplement bool) (wasModified bool, e error) {
	serviceRelativePath := fmt.Sprintf("services/%s.go", strings.ToLower(modelName))
	filePath := filepath.Join(mainPkgPath, serviceRelativePath)
	serviceName := modelName + "Service"
	modified := reimplement

	fileModified, err := CreateServiceFileIfNotExists(filePath)
	if err != nil {
		return false, err
	}
	if fileModified {
		modified = true
	}

	fset := token.NewFileSet()
	serviceFile, err := decorator.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Parsing error: %s: %v", mainPkgPath, err)
		return false, err
	}

	importsModified, err := MaintainImports(serviceFile)
	if err != nil {
		return false, err
	}
	if importsModified {
		modified = true
	}

	aliasAdded := ImplementModelAlias(modelName, serviceFile)
	if aliasAdded {
		modified = true
	}

	serviceStructModified := ImplementServiceStruct(modelName, serviceFile, reimplement)
	if serviceStructModified {
		modified = true
	}

	methodsModified, err := ImplementCrudMethods(modelName, serviceName, serviceFile, reimplement)
	if err != nil {
		return false, err
	}
	if methodsModified {
		modified = true
	}

	file, err := os.Create(filePath)
	if err != nil {
		errMessage := errors.New(fmt.Sprintf("Error occured to open `%s` service file: %v", modelName, err))
		return false, errMessage
	}

	err = decorator.Fprint(file, serviceFile)
	if err != nil {
		errMessage := errors.New(
			fmt.Sprintf("Error occurred to writing changes in `%s` service file: %v", modelName, err),
		)
		return false, errMessage
	}

	defer file.Close()

	return modified, nil
}

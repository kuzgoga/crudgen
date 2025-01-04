package main

import (
	"crudgen/internal"
	"flag"
	"fmt"
	"log"
	"path/filepath"
)

func ImplementServices(mainPkgDir string, reimplement bool) {
	modelsNames, err := internal.GetStructNames(filepath.Join(mainPkgDir, "models"))
	if err != nil {
		log.Printf("Error: %s\n", err)
		return
	}

	fmt.Printf("Found models: %#v\n", modelsNames)
	
	for _, modelName := range modelsNames {
		log.Print(modelName)
		err := internal.ImplementService(mainPkgDir, modelName, reimplement)
		if err != nil {
			log.Printf("Error implement service for model %s: %s\n", modelName, err)
		}
	}
}

func main() {
	projectPath := flag.String("p", ".", "project path")
	reimplement := flag.Bool("f", false, "pass -f to allow tool to overwrite exist functions and service structure")
	flag.Parse()
	ImplementServices(*projectPath, *reimplement)
}

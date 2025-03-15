package main

import (
	"flag"
	"git.gogacoder.ru/NTO/crudgen/internal"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

func ImplementServices(mainPkgDir string, reimplement bool) (modified bool) {
	modelsNames, err := internal.GetStructNames(filepath.Join(mainPkgDir, "models"))
	if err != nil {
		log.Printf("Error: %s\n", err)
		return
	}
	var wasModified bool

	log.Printf("Found models: %v\n", modelsNames)

	for _, modelName := range modelsNames {
		codeModified, err := internal.ImplementService(mainPkgDir, modelName, reimplement)
		if codeModified {
			wasModified = true
		}
		if err != nil {
			log.Printf("Error implement service for model %s: %s\n", modelName, err)
		}
	}
	return wasModified
}

func runPostHook(postHook *string, wasModified bool) {
	if wasModified && postHook != nil && *postHook != "" {
		log.Printf("Running post hook %s\n", *postHook)
		args := strings.Fields(*postHook)

		var cmd *exec.Cmd

		if len(args) == 0 {
			log.Printf("Empty post hook %s\n", *postHook)
			return
		}
		if len(args) == 1 {
			cmd = exec.Command(args[0])
		} else {
			cmd = exec.Command(args[0], args[1:]...)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("Error running post hook for %s: %s\n", *postHook, err)
		} else {
			log.Printf("Post hook output: %s\n", string(output))
		}
	}
}

func main() {
	log.SetFlags(0)
	projectPath := flag.String("p", ".", "project path")
	reimplement := flag.Bool("f", false, "pass -f to allow tool to overwrite exist functions and service structure")
	postHook := flag.String("h", "", "post hook to run command after code modifications")

	flag.Parse()
	wasModified := ImplementServices(*projectPath, *reimplement)
	runPostHook(postHook, wasModified)
}

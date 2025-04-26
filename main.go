package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/treaster/sprout/processor"
)

type TemplateMgr interface {
	ParseOne(tmplName string, tmplBody []byte) error
	Execute(tmplName string, tmplData any, output io.Writer) error
}

func main() {
	var inputRoot string
	flag.StringVar(&inputRoot, "input", "", "The root directory of the template to sprout.")

	var outputRoot string
	flag.StringVar(&outputRoot, "output", "", "The root directory where sprouted output should be placed. This directory will be created if it does not exist.")

	var paramsFileName string
	flag.StringVar(&paramsFileName, "params", "params.hjson", "The name of the params file to look for in the output.")

	flag.Parse()

	templateMgrFactories := map[string]func() processor.TemplateMgr{
		"go/template": processor.GoTemplateMgr,
		"jet":         processor.JetTemplateMgr,
	}

	if inputRoot == "" {
		fmt.Println("--input is required and not defined")
		os.Exit(1)
	}

	if outputRoot == "" {
		fmt.Println("--output is required and not defined")
		os.Exit(1)
	}

	// Create output directory
	err := os.MkdirAll(outputRoot, 0755)
	if err != nil {
		Printfln("error creating --output directory: %s", err.Error())
		os.Exit(1)
	}

	// Load the params template file. This is an example or placeholder file
	// which will be used to parameterize the output, but when we start it
	// probably doesn't exist. So we copy an example params file from the
	// input directory, then invite the user to customize it for their
	// purposes.
	//
	// The next time we run sprout, the final params file will already exist,
	// and we'll skip this step and just run the actual template execution.
	//
	// TODO(treaster): Ensure we ignore the example params file from the
	// template execution.
	// TODO(treaster): Look for the customized params file somewhere besides
	// the output directory, so we can delete the output directory and restart,
	// without losing the params.
	//
	// TODO(treaster): Auto-scrub the output directory if it appears to already
	// exist. We don't want old template remnants cluttering the space if we
	// rerun sprout to regenerate the directory.
	paramsLoader := processor.MakeFileLoader(outputRoot, ".", os.ReadFile)
	paramsPath := filepath.Join(outputRoot, paramsFileName)
	var params processor.Params
	err = paramsLoader.LoadFile(paramsPath, &params)
	if err == os.ErrNotExist {
		paramsTemplatePath := filepath.Join(inputRoot, paramsFileName)
		paramsTemplateBytes, err := os.ReadFile(paramsTemplatePath)
		if err != nil {
			Printfln("error reading params template to copy: %s", err.Error())
			os.Exit(1)
		}

		err = os.WriteFile(paramsPath, paramsTemplateBytes, 644)
		if err != nil {
			Printfln("error writing params template: %s", err.Error())
			os.Exit(1)
		}

		os.Exit(0)
	}

	// We make two passes of executing the templates:
	// 1. A dry-run pass where we make sure everything evaluates cleanly
	// 2. A final pass where we actually write the output.
	// TODO(treaster): We could do this in one execution, by creating a map
	// of filename->contents, then writing only after all processing is done.
	writeFns := []func(name string, data []byte, perms os.FileMode) error{
		func(name string, data []byte, perms os.FileMode) error {
			Printfln("Dry run success: %s", name)
			return nil
		},
		os.WriteFile,
	}

	for _, writeFn := range writeFns {
		errs := processor.LoadTemplates(
			templateMgrFactories,
			inputRoot,
			outputRoot,
			params,
			os.ReadFile,
			writeFn,
		)
		for _, err := range errs {
			fmt.Println(err.Error())
		}

		if len(errs) > 0 {
			os.Exit(1)
		}
	}
}

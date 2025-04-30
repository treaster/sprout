package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/treaster/sprout/processor"
)

func main() {
	var sourceConfigPath string
	flag.StringVar(&sourceConfigPath, "source-config", "", "The definition config of the template to sprout.")

	var outputRoot string
	flag.StringVar(&outputRoot, "output", "", "The root directory where sprouted output should be placed. This directory will be created if it does not exist.")

	var paramsPath string
	flag.StringVar(&paramsPath, "params", "params.hjson", "The name of the params file to look for in the output.")

	var deleteExistingOutput bool
	flag.BoolVar(&deleteExistingOutput, "delete-existing-output", false, "delete an existing output directory, if it exists")

	flag.Parse()

	templateMgrFactories := map[string]func() processor.TemplateMgr{
		".gotmpl": processor.GoTemplateMgr,
		".jet":    processor.JetTemplateMgr,
	}

	hasErrors := false
	if sourceConfigPath == "" {
		fmt.Println("--source-config is required and not defined")
		hasErrors = true
	}

	if outputRoot == "" {
		fmt.Println("--output is required and not defined")
		hasErrors = true
	}

	inputRoot := filepath.Dir(sourceConfigPath)
	absParamsPath := processor.ScrubPath(paramsPath)
	absOutputPath := processor.ScrubPath(outputRoot) + "/"
	if strings.HasPrefix(absParamsPath, absOutputPath) {
		processor.Printfln("--params path is inside --output path. This is not allowed.")
		hasErrors = true
	}

	var config processor.Config
	configLoader := processor.MakeFileLoader(".", ".", os.ReadFile)
	err := configLoader.LoadFile(sourceConfigPath, &config)
	if err != nil {
		processor.Printfln("error loading source config: %s", err.Error())
		hasErrors = true
	}

	if hasErrors {
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
	// TODO(treaster): Auto-scrub the output directory if it appears to already
	// exist. We don't want old template remnants cluttering the space if we
	// rerun sprout to regenerate the directory.
	paramsLoader := processor.MakeFileLoader(".", ".", os.ReadFile)

	var params processor.Params
	err = paramsLoader.LoadFile(paramsPath, &params)
	if os.IsNotExist(err) {
		templateParamsPath := filepath.Join(inputRoot, config.TemplateParamsFile)
		err := processor.Copy(templateParamsPath, paramsPath)
		if err != nil {
			processor.Printfln("error copying params template from %s to %s: %s", config.TemplateParamsFile, paramsPath, err.Error())
			hasErrors = true
		} else {
			processor.Printfln("created placeholder params at %s. customize the file, then rerun your previous command.", paramsPath)
			os.Exit(0)
		}
	}

	fileInfo, err := os.Stat(outputRoot)
	if err != nil && !os.IsNotExist(err) {
		processor.Printfln("error checking status of --output directory: %s", err.Error())
		hasErrors = true
	}

	if err == nil {
		processor.Printfln("found existing directory")
		if !fileInfo.IsDir() {
			processor.Printfln("error in --output target. Expected directory, but found a normal file")
			hasErrors = true
		}

		if !deleteExistingOutput {
			processor.Printfln("--output directory already exists. Remove it, or use --delete-existing-output")
			hasErrors = true
		}

	}

	templateMgrFactory, hasExt := templateMgrFactories[config.TemplateTypeExt]
	if !hasExt {
		processor.Printfln("unrecognized template type %q in config", config.TemplateTypeExt)
		hasErrors = true
	}

	templateMgr := templateMgrFactory()
	configBytes, err := configLoader.LoadFileAsBytes(sourceConfigPath)
	if err != nil {
		processor.Printfln("error reloading config for template rewrite?!?: %s", err.Error())
		hasErrors = true
	}
	err = templateMgr.ParseOne("__config__", configBytes)
	if err != nil {
		processor.Printfln("error processing config as template: %s", err.Error())
		hasErrors = true
	}

	var processedConfigBuf bytes.Buffer
	err = templateMgr.Execute("__config__", params, &processedConfigBuf)
	if err != nil {
		processor.Printfln("error executing config template: %s", err.Error())
		hasErrors = true
	}

	var processedConfig processor.Config
	err = configLoader.DeserializeBytes(sourceConfigPath, processedConfigBuf.Bytes(), &processedConfig)
	if err != nil {
		processor.Printfln("error deserializing processed config bytes: %s", err.Error())
		hasErrors = true
	}

	if hasErrors {
		os.Exit(1)
	}

	_ = os.RemoveAll(outputRoot)
	errs := processor.Process(
		templateMgrFactory(),
		inputRoot,
		outputRoot,
		processedConfig,
		params,
		os.ReadFile,
		os.WriteFile,
	)
	for _, err := range errs {
		fmt.Println(err.Error())
	}

	if len(errs) > 0 {
		os.Exit(1)
	}
}

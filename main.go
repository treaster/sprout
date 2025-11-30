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
	const defaultDigestFile = "digest.txt"

	var sourceConfigPath string
	flag.StringVar(&sourceConfigPath, "source-config", "", "The definition config of the template to sprout.")

	var outputRoot string
	flag.StringVar(&outputRoot, "output", "", "The root directory where sprouted output should be placed. This directory will be created if it does not exist.")

	var paramsPath string
	flag.StringVar(&paramsPath, "params", "params.hjson", "The name of the params file to look for in the output.")

	var unused bool
	flag.BoolVar(&unused, "delete-existing-output", false, "This flag is unused.")

	var digestPath string
	flag.StringVar(&digestPath, "digest", defaultDigestFile, "record the filepaths of each generated file, so they can be cleaned up if necessary.")

	flag.Parse()

	templateMgrFactories := map[string]func() processor.TemplateMgr{
		".gotmpl": processor.GoTemplateMgr,
		".jet":    processor.JetTemplateMgr,
		".pongo":  processor.PongoTemplateMgr,
	}

	hasErrors := false
	if sourceConfigPath == "" {
		fmt.Println("--source-config is required and not defined")
		hasErrors = true
	}
	inputRoot := filepath.Dir(sourceConfigPath)

	if outputRoot == "" {
		fmt.Println("--output is required and not defined")
		hasErrors = true
	}
	outputRoot = filepath.Clean(outputRoot) + "/"

	// Load the digest file, if it exists.
	absDigestPath := filepath.Join(outputRoot, digestPath)
	if digestPath == defaultDigestFile {
		processor.Printfln("using default digest path %s", absDigestPath)
	}
	digestBytes, err := os.ReadFile(absDigestPath)
	if err != nil && !os.IsNotExist(err) {
		processor.Printfln("error reading digest file: %s", err.Error())
		hasErrors = true
	}

	var digestPaths []string
	if err == nil {
		digestPaths = strings.Split(string(digestBytes), "\n")
		digestPaths = append(digestPaths, digestPath)
	}

	// Load the template config.
	var config processor.Config
	configLoader := processor.MakeFileLoader(".", ".", os.ReadFile)
	err = configLoader.LoadFile(sourceConfigPath, &config)
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

	// Select a template engine, based on the TemplateTypeExt specified in the config.
	templateMgrFactory, hasExt := templateMgrFactories[config.TemplateTypeExt]
	if !hasExt {
		processor.Printfln("unrecognized template type %q in config", config.TemplateTypeExt)
		hasErrors = true
	}
	templateMgr := templateMgrFactory()

	// Reload the config again, and this type parse it as a template using
	// the params as an input. This will fully resolve any templated variables
	// anywhere in the config specification.
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

	// If we've encountered any errors so far, exit before we start doing any
	// mutations to the filesystem.
	if hasErrors {
		os.Exit(1)
	}

	// Delete all entries from the digest, which represents files written by a
	// previous run of sprout. If removing a file would leave its directory
	// empty, remove the directory also. Recursively repeat this until a
	// nonempty directory is found.
	for _, digestEntry := range digestPaths {
		for digestPart := digestEntry; digestPart != ""; digestPart = filepath.Dir(digestPart) {
			pathInOutput := filepath.Join(outputRoot, digestPart)
			err = os.Remove(pathInOutput)
			if err != nil && digestPart == digestEntry {
				processor.Printfln("error removing digest entry %s: %s", pathInOutput, err.Error())
				hasErrors = true
			}
			if err != nil {
				break
			}
			processor.Printfln("delete digest entry %s", pathInOutput)
		}
	}

	// Execute the template logic.
	errs := processor.Process(
		templateMgrFactory(),
		inputRoot,
		outputRoot,
		absDigestPath,
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

package processor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Params map[string]any

func Process(
	templateMgrFactories map[string]func() TemplateMgr,
	inputRoot string,
	outputRoot string,
	params Params,
	readFileFn func(string) ([]byte, error),
	writeFileFn func(string, []byte, os.FileMode) error,
) []error {
	templatesLoader := MakeFileLoader(
		inputRoot,
		".",
		readFileFn,
	)

	var errs []error
	addError := func(s string, args ...any) {
		errs = append(errs, fmt.Errorf(s, args...))
	}

	// Find all files in the input directory.
	templateNames := templatesLoader.FindFiles()
	if len(templateNames) == 0 {
		addError("no input files found in %q", inputRoot)
		return errs
	}

	// Process each template found, generating a corresponding output file in
	// the output directory.
	for _, templateName := range templateNames {
		templateExt := filepath.Ext(templateName)
		templateContents, err := templatesLoader.LoadFileAsBytes(templateName)
		if err != nil {
			addError("error reading template %q: %s", templateName, err.Error())
			continue
		}

		// If the file extension isn't recognized as a template file type,
		// assume it's a non-templated file and just copy it over directly.
		templateMgrFactory, hasExt := templateMgrFactories[templateExt]
		if !hasExt {
			Printfln("Copying non-template file: %s", templateName)
			err = writeFileFn(templateName, templateContents, 0644)
			continue
		}

		templateMgr := templateMgrFactory()
		err = templateMgr.ParseOne(templateName, templateContents)
		if err != nil {
			addError("error parsing template %q: %s", templateName, err.Error())
			continue
		}

		var output bytes.Buffer
		err = templateMgr.Execute(templateName, params, &output)
		if err != nil {
			addError("error executing template: %s", err.Error())
			continue
		}

		// Write the output to a corresponding file in the output directory.
		outputName := strings.TrimSuffix(templateName, templateExt)
		outputPath := filepath.Join(outputRoot, outputName)
		outputFileDir := filepath.Dir(outputPath)
		err = os.MkdirAll(outputFileDir, 0755)
		if err != nil {
			addError("error creating output directory: %s", err.Error())
			continue
		}

		Printfln("    Writing file %s", outputPath)
		err = os.WriteFile(outputPath, output.Bytes(), 0644)
		if err != nil {
			addError("error writing output file: %s", err.Error())
			continue
		}
	}

	return errs
}

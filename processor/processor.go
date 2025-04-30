package processor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	TemplateTypeExt    string
	TemplateParamsFile string
	DirsMapping        map[string]string
}

type TemplateMgr interface {
	ParseOne(tmplName string, tmplBody []byte) error
	Execute(tmplName string, tmplData any, output io.Writer) error
}

type Params map[string]any

func Process(
	templateMgr TemplateMgr,
	inputRoot string,
	outputRoot string,
	config Config,
	params Params,
	readFileFn func(string) ([]byte, error),
	writeFileFn func(string, []byte, os.FileMode) error,
) []error {
	var errs []error
	addError := func(s string, args ...any) {
		errs = append(errs, fmt.Errorf(s, args...))
	}

	outputContents := map[string][]byte{}
	for inputSubdir, targetSubdir := range config.DirsMapping {
		templatesLoader := MakeFileLoader(
			filepath.Join(inputRoot, inputSubdir),
			".",
			readFileFn,
		)

		// Find all files in the input directory.
		templateNames := templatesLoader.FindFiles()
		if len(templateNames) == 0 {
			addError("no input files found in %q", inputRoot)
			return errs
		}

		// Process each template found, generating a corresponding output file in
		// the output directory.
		for _, templateName := range templateNames {
			fileExt := filepath.Ext(templateName)
			templateContents, err := templatesLoader.LoadFileAsBytes(templateName)
			if err != nil {
				addError("error reading template %q: %s", templateName, err.Error())
				continue
			}

			var output bytes.Buffer
			if fileExt != config.TemplateTypeExt {
				// If the file extension isn't recognized as a template file type,
				// assume it's a non-templated file and just copy it over directly.
				_, err = output.Write(templateContents)
				if err != nil {
					addError("error copying file contents of %s into buffer: %s", templateName, err.Error())
					continue
				}
			} else {
				err = templateMgr.ParseOne(templateName, templateContents)
				if err != nil {
					addError("error parsing template %q: %s", templateName, err.Error())
					continue
				}

				err = templateMgr.Execute(templateName, params, &output)
				if err != nil {
					addError("error executing template: %s", err.Error())
					continue
				}
				templateName = strings.TrimSuffix(templateName, config.TemplateTypeExt)
			}

			outputSubdirRelativePath := filepath.Join(outputRoot, targetSubdir)

			// Write the output to a corresponding file in the output directory.
			outputPath := filepath.Join(outputSubdirRelativePath, templateName)
			outputFileDir := filepath.Dir(outputPath)
			err = os.MkdirAll(outputFileDir, 0755)
			if err != nil {
				addError("error creating output directory %s: %s", outputFileDir, err.Error())
				continue
			}

			_, hasPath := outputContents[outputPath]
			if hasPath {
				addError("at least two template files map to the same output location: %s", outputPath)
				continue
			}
			outputContents[outputPath] = output.Bytes()
		}
	}

	for path, content := range outputContents {
		Printfln("    Writing file %s", path)
		err := os.WriteFile(path, content, 0644)
		if err != nil {
			addError("error writing output file: %s", err.Error())
			continue
		}
	}

	return errs
}

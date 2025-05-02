package processor

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Config is a static definition defining the template behavior. It itself
// can be templated, except for the TemplateTypeExt field. All other fields
// will be used only after any template expressions are resolved.
type Config struct {
	TemplateTypeExt    string
	TemplateParamsFile string
	DirsMapping        map[string]string
}

// Params is the user-specified input to the template. These params are combined
// with the actual template files to produce the final output. All variables that
// are referenced in the templates must be defined in the Params. A clean
// template specification should include an example Params, to demonstrate what
// fields are expected.
type Params map[string]any

// TemplateMgr represents a templating engine. A new template system
// (e.g. Mustache) can be dropped in easily if it satisfies, or can be wrapped
// in, this interface.
type TemplateMgr interface {
	ParseOne(tmplName string, tmplBody []byte) error
	Execute(tmplName string, tmplData any, output io.Writer) error
}

func Process(
	templateMgr TemplateMgr,
	inputRoot string,
	outputRoot string,
	absDigestPath string,
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

			// Prepare output content, but don't write it yet, until we're
			// confident there are no processing errors in any templates.
			outputSubdirPath := filepath.Join(outputRoot, targetSubdir)
			outputPath := filepath.Join(outputSubdirPath, templateName)
			_, hasPath := outputContents[outputPath]
			if hasPath {
				addError("at least two template files map to the same output location: %s", outputPath)
				continue
			}

			outputBytes := bytes.TrimSpace(output.Bytes())
			if len(outputBytes) == 0 {
				Printfln("skipping output file with no output: %s", outputPath)
				continue
			}
			outputContents[outputPath] = output.Bytes()
		}
	}

	// Short-circuit before doing any writes, if errors occurred.
	if len(errs) > 0 {
		return errs
	}

	// Write the output to a corresponding file in the output directory.
	filesWritten := make([]string, 0, len(outputContents))
	allPaths := slices.Sorted(maps.Keys(outputContents))
	for _, path := range allPaths {
		content := outputContents[path]
		outputFileDir := filepath.Dir(path)
		err := os.MkdirAll(outputFileDir, 0755)
		if err != nil {
			addError("error creating output directory %s: %s", outputFileDir, err.Error())
			continue
		}

		Printfln("    Writing file %s", path)
		err = os.WriteFile(path, content, 0644)
		if err != nil {
			addError("error writing output file: %s", err.Error())
			continue
		}

		path = SafeCutPrefix(path, outputRoot)
		filesWritten = append(filesWritten, path)
	}

	// Write the digest file.
	digestContents := strings.Join(filesWritten, "\n")
	err := os.WriteFile(absDigestPath, []byte(digestContents), 0644)
	if err != nil {
		addError("error writing digest file: %s", err.Error())
	}

	return errs
}

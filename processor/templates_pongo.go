package processor

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/flosch/pongo2/v6"
)

type pongoCustomLoader map[string][]byte

func (cl pongoCustomLoader) Abs(base string, name string) string {
	// base seems to be the name of the calling template? We'll just
	// drop it, and assume all template-internal template references
	// have a fully(enough)-qualified path. So return (only) the name
	// that's actually in the calling template.
	return name
}

func (cl pongoCustomLoader) Get(name string) (io.Reader, error) {
	name = strings.TrimPrefix(name, "/")
	contents, hasName := cl[name]
	if !hasName {
		return nil, fmt.Errorf("unrecognized template name %q", name)
	}
	return io.NopCloser(bytes.NewBuffer(contents)), nil
}

func (cl pongoCustomLoader) add(name string, contents []byte) {
	name = strings.TrimPrefix(name, "/")
	cl[name] = contents
}

func PongoTemplateMgr() TemplateMgr {
	pongo2.SetAutoescape(false)

	loader := pongoCustomLoader{}
	set := pongo2.NewSet("sprout", loader)

	templateFuncs := TemplateFuncs()
	for name, fn := range templateFuncs {
		set.Globals[name] = fn
	}

	set.Options.TrimBlocks = false   // default is false
	set.Options.LStripBlocks = false // default is false

	return &pongoTemplateMgr{
		loader,
		set,
	}
}

type pongoTemplateMgr struct {
	loader pongoCustomLoader
	set    *pongo2.TemplateSet
}

func (tm *pongoTemplateMgr) ParseOne(tmplName string, tmplBody []byte) error {
	tm.loader.add(tmplName, tmplBody)
	return nil
}

func (tm *pongoTemplateMgr) Execute(tmplName string, tmplData any, output io.Writer) error {
	outputStr, err := tm.set.RenderTemplateFile(tmplName, map[string]any{"PARAMS": tmplData})
	if err != nil {
		return err
	}
	_, err = output.Write([]byte(outputStr))
	return err
}

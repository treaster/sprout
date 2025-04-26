package processor

import (
	"errors"
	"fmt"
	"io"
	"text/template"
)

func GoTemplateMgr() TemplateMgr {
	tmpl := template.
		New("sprout").
		Funcs(template.FuncMap{
			"NamedArgs": NamedArgs,
		}).
		Option("missingkey=error")

	return &goTemplateMgr{tmpl}
}

type goTemplateMgr struct {
	tmpl *template.Template
}

func (tm *goTemplateMgr) ParseOne(tmplName string, tmplBody []byte) error {
	_, err := tm.tmpl.New(tmplName).Parse(string(tmplBody))
	if err != nil {
		return fmt.Errorf("error parsing template %q: %s", tmplName, err.Error())
	}
	return nil
}

func (tm *goTemplateMgr) Execute(tmplName string, tmplData any, output io.Writer) error {
	tmpl := tm.tmpl.Lookup(tmplName)
	if tmpl == nil {
		panic(fmt.Sprintf("error: template %q not found", tmplName))
	}

	return tmpl.Execute(output, tmplData)
}

func NamedArgs(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

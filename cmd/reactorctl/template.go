package main

import (
	"bytes"
	"strings"
	"text/template"
)

func render(tmpl string, data interface{}) (string, error) {
	var buf bytes.Buffer

	t := template.Must(template.New("").Parse(strings.Trim(tmpl, "\n")))
	err := t.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

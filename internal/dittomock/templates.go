package dittomock

import (
	"bytes"
	"text/template"
	"time"
)

var funcMap = template.FuncMap{
	"now":             time.Now,
	"now_rfc3339":     nowRfc3339,
	"now_utc_rfc3339": nowUtcRfc3339,
}

func nowRfc3339() string {
	return time.Now().Format(time.RFC3339)
}

func nowUtcRfc3339() string {
	return time.Now().Format(time.RFC3339)
}

func parseTemplate(tpl []byte) ([]byte, error) {
	tmpl, err := template.New("tpl").Funcs(funcMap).Parse(string(tpl))
	if err != nil {
		return nil, err
	}

	sb := &bytes.Buffer{}
	err = tmpl.Execute(sb, nil)
	if err != nil {
		return nil, err
	}

	return sb.Bytes(), nil
}

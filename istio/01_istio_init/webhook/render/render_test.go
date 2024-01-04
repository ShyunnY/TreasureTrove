package render

import (
	"bytes"
	"fishnet-inject/sugar"
	sprig "github.com/go-task/slim-sprig"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
	"text/template"
)

func init() {
	sugar.InitLogger()
}

func TestRenderHello(t *testing.T) {

	var tmpl = `
{{ define "a" }}
{{ . | repeat 5 }}
{{ end }}

{{ define "b" }}
 {{ . | repeat 4 }}
{{ end }}
`
	tmp, err := template.New("base").Funcs(sprig.FuncMap()).Parse(tmpl)
	assert.NoError(t, err)

	renderData := bytes.NewBuffer(nil)
	err = tmp.Execute(renderData, "z3")
	assert.NoError(t, err)
	assert.Equal(t, "z3z3z3z3z3", renderData.String())

}

func TestRender_AddTextTemplate(t *testing.T) {

	render := NewRender()

	var t1 = `{{ . | repeat 3 }} 
`
	var t2 = `{{ . | upper }} 
`

	err := render.AddTextTemplate("repeat_template", NewRenderData(func() (io.Writer, any) {
		return os.Stdout, "z3"
	}, nil), t1)
	assert.NoError(t, err)

	err = render.AddTextTemplate("lower_template", NewRenderData(func() (io.Writer, any) {
		return os.Stdout, "z3"
	}, nil), t2)
	assert.NoError(t, err)

	err = render.RunRenderTemplate()
	assert.NoError(t, err)
}

func TestRender_AddFileTemplate(t *testing.T) {

	render := NewRender()

	err := render.AddFileTemplate(NewRenderData(func() (io.Writer, any) {
		return os.Stdout, "jane"
	}, nil), "test.tpl")
	assert.NoError(t, err)

	runErr := render.RunRenderTemplate()
	assert.NoError(t, runErr)

}

package render

import (
	"bytes"
	sprig "github.com/go-task/slim-sprig"
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestRenderHello(t *testing.T) {

	var tmpl = `{{ . | repeat 5 }}`
	tmp, err := template.New("base").Funcs(sprig.FuncMap()).Parse(tmpl)
	assert.NoError(t, err)

	renderData := bytes.NewBuffer(nil)
	err = tmp.Execute(renderData, "z3")
	assert.NoError(t, err)
	assert.Equal(t, "z3z3z3z3z3", renderData.String())

}

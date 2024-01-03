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
	}, func(writer io.Writer) {
		writer.Write([]byte("\n hello,postHook \n"))
	}), "test.tpl")
	assert.NoError(t, err)

	runErr := render.RunRenderTemplate()
	assert.NoError(t, runErr)

}

//func TestRender_SidecarTemplate(t *testing.T) {
//
//	render := NewRender()
//	tc := webhooks.TemplateData{
//		SidecarName:  "envoyproxy",
//		SidecarImage: "envoyproxy:2.0",
//		SidecarEnvs: []corev1.EnvVar{
//			{
//				Name:  "usr",
//				Value: "Peter",
//			},
//		},
//		SidecarArgs: []string{
//			"-a",
//			"-b",
//		},
//	}
//
//	assert.NoError(t, render.AddFileTemplate(func() (io.Writer, any) {
//		return os.Stdout, tc
//	}, "tpls/sidecar.tpl"))
//
//	assert.NoError(t, render.RunRenderTemplate())
//
//}

//func TestRender_InitContainerTemplate(t *testing.T) {
//
//	render := NewRender()
//	tc := webhooks.TemplateData{
//		InitContainerImage: "fishnet.io/init:2.0",
//		InitContainerName:  "fishnet-init",
//		InitContainerArgs: []string{
//			"istio-iptables",
//			"-p",
//			"15001",
//			"-z",
//			"15006",
//			"-u",
//			"1337",
//			"-m",
//			"REDIRECT",
//			"-i",
//			`'*'`,
//			`-x`,
//			`""`,
//			"-b",
//			`'*'`,
//			"-d",
//			"15090,15021,15020",
//			"--log_output_level=default:info",
//		},
//	}
//
//	assert.NoError(t, render.AddFileTemplate(func() (io.Writer, any) {
//		return os.Stdout, tc
//	}, "tpls/initContainer.tpl"))
//
//	assert.NoError(t, render.RunRenderTemplate())
//
//}

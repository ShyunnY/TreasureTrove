package render

import (
	"errors"
	"fishnet-inject/sugar"
	sprig "github.com/go-task/slim-sprig"
	"io"
	"sync"
	"text/template"
)

// Render
// 主要用于渲染自动注入的模板数据
// 我们可以将其设计成一个通用的渲染器
//
//	1.可以渲染"一组"模板
//	2.外部可以传递自定义数据进行渲染
//	3.可以同时存在多个不同功能的模板, 不同模板的数据, 将数据与模板对应起来
//
// TODO: update 测试
// TODO: 增强鲁棒性
// TODO: 增加后环绕hook机制
type render struct {
	mut sync.Mutex

	// 外部模板函数
	templates map[string]*template.Template

	// 模板对应的执行函数
	execTemplates map[string]RenderData

	// 将可以在模板中使用的funcMaps
	funcMaps []template.FuncMap

	hook ExecHookFunc
}

type RenderData struct {
	writer   io.Writer
	data     any
	postHook ExecHookFunc
}
type ExecHookFunc func(io.Writer) error

func NewRenderData(build func() (io.Writer, any), postHook ExecHookFunc) RenderData {

	writer, data := build()
	rd := RenderData{
		writer:   writer,
		data:     data,
		postHook: postHook,
	}

	return rd
}

func NewRender() *render {

	render := &render{
		mut:       sync.Mutex{},
		templates: map[string]*template.Template{},
		// 默认使用功能丰富的sprig.FuncMap
		funcMaps:      []template.FuncMap{sprig.FuncMap()},
		execTemplates: map[string]RenderData{},

		hook: nil,
	}

	return render
}

func (r *render) RunRenderTemplate() error {

	for name, tmpl := range r.templates {
		tmpl := tmpl

		rd, ok := r.execTemplates[name]
		if !ok {
			return errors.New("execParamFunc function for this template does not exist")
		}

		if err := tmpl.Execute(rd.writer, rd.data); err != nil {
			return err
		} else if rd.postHook != nil {
			if hookErr := rd.postHook(rd.writer); hookErr != nil {
				return hookErr
			}
		}

		sugar.Debugf("render %s template", name)
	}

	return nil
}

func (r *render) AddTextTemplate(name string, rd RenderData, text string) error {
	if name == "" {
		return errors.New("template name must not empty")
	}

	if text == "" {
		return errors.New("template text must not empty")
	}

	tmpl := template.New(name)

	// apply funcMap
	for _, fm := range r.funcMaps {
		tmpl.Funcs(fm)
	}

	tmpl, err := tmpl.Parse(text)
	if err != nil {
		return err
	}

	r.addTemplate(tmpl, rd)

	return nil
}

func (r *render) AddFileTemplate(rd RenderData, path ...string) error {

	if len(path) <= 0 {
		return errors.New("add template file path must not empty")
	}

	tmpl, err := template.ParseFiles(path...)
	if err != nil {
		return err
	}

	// apply funcMap
	for _, fm := range r.funcMaps {
		tmpl.Funcs(fm)
	}

	r.addTemplate(tmpl, rd)

	return nil
}

func (r *render) SetFuncMaps(fm ...template.FuncMap) {
	r.mut.Lock()
	defer r.mut.Unlock()

	r.funcMaps = append(r.funcMaps, fm...)
}

func (r *render) SetHook(hook ExecHookFunc) {
	r.mut.Lock()
	defer r.mut.Unlock()

	r.hook = hook
}

func (r *render) addTemplate(tmpl *template.Template, rd RenderData) {
	r.mut.Lock()
	defer r.mut.Unlock()

	r.templates[tmpl.Name()] = tmpl
	r.execTemplates[tmpl.Name()] = rd

}

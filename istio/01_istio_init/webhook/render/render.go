package render

import (
	sprig "github.com/go-task/slim-sprig"
	"sync"
	"text/template"
)

// TODO: 使用Go.template库进行渲染模板

// Render
// 主要用于渲染自动注入的模板数据
// 我们可以将其设计成一个通用的渲染器
// TODO: 1.可以渲染"一组"模板
// TODO: 2.外部可以传递自定义数据进行渲染
// TODO: 3.可以同时存在多个不同功能的模板, 不同模板的数据, 将数据与模板对应起来
type Render struct {
	mut sync.Mutex

	// 外部模板函数
	templates []*template.Template

	// 将可以在模板中使用的funcMaps
	funcMaps []template.FuncMap
}

func NewRender() *Render {

	render := &Render{
		mut:       sync.Mutex{},
		templates: []*template.Template{},
		// 默认使用功能丰富的sprig.FuncMap
		funcMaps: []template.FuncMap{sprig.FuncMap()},
	}

	return render
}

func (r *Render) AppendTemplate(tmpl *template.Template) {
	r.mut.Lock()
	defer r.mut.Unlock()

	r.templates = append(r.templates, tmpl)
}

func (r *Render) SetFuncMaps(fm ...template.FuncMap) {
	r.mut.Lock()
	defer r.mut.Unlock()

	r.funcMaps = append(r.funcMaps, fm...)
}

package tpls

import _ "embed"

// static template
var (
	//go:embed sidecar.tpl
	SidecarTemplate string

	//go:embed initContainer.tpl
	InitContainerTemplate string
)

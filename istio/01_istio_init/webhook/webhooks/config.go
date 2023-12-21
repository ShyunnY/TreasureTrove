package webhooks

import "text/template"

type Config struct {
	InitTemplate    template.Template
	SidecarTemplate template.Template
}

func NewConfig() *Config {
	return &Config{}
}

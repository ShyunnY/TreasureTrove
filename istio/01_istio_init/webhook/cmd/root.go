package cmd

import (
	_ "embed"
	"fishnet-inject/cmd/injector"
	"fishnet-inject/utils"
	"fishnet-inject/vars"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"os"
	"text/template"
)

var (

	//go:embed usage.tpl
	usageTpl string
	rootCmd  = &cobra.Command{
		Use:     "fishnet",
		Short:   "A lightweight ServiceMesh",
		Long:    "A lightweight ServiceMesh, has service registration, observability, auto-injection and more",
		Version: version(),
	}
)

// Exec
// rootCmd执行的入口
func Exec() {

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(aurora.Red(err.Error()))
		os.Exit(1)
	}

}

func version() string {
	return vars.Version
}

func init() {

	cobra.AddTemplateFuncs(template.FuncMap{
		"blue":    utils.Blue,
		"green":   utils.Green,
		"rpadx":   utils.Rpadx,
		"rainbow": utils.Rainbow,
	})

	// set usage template
	rootCmd.SetUsageTemplate(usageTpl)

	// add command
	rootCmd.AddCommand(injector.Cmd)
}

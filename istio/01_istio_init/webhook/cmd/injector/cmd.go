package injector

import (
	"context"
	"fishnet-inject/sugar"
	"fishnet-inject/webhooks"
	"github.com/spf13/cobra"
	"os"
)

// use example
var examples = `
  running on the specified port:  fishnet injector -p 9527
  running on the specified log level:  fishnet injector -v debug`

// flags
var (
	varLogLevel string
	varPort     int
)

// Cmd injector command
var (
	Cmd = &cobra.Command{
		Use:     "injector",
		Short:   "control sidecar auto-injection",
		Aliases: []string{"ij"},
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {

			sugar.InitLogger(varLogLevel)

			// init injector webhook
			sre := webhooks.NewWebhookServer(webhooks.WebhookServerConfig{})

			sugar.Infof("fishnet injector webhook running on 9527")
			if err := sre.Start(context.Background()); err != nil {
				sugar.Error("fishnet injector webhook run error: ", err)
				os.Exit(1)
			}
			return nil
		},
	}
)

func init() {

	// injector flags
	Cmd.Flags().StringVarP(&varLogLevel, "log-level", "v", "", "set the logger level. (Default logger level is INFO)")
	Cmd.Flags().IntVarP(&varPort, "port", "p", 0, "set injector webhook run port. (Default port 9527)")
}

package injector

import (
	"context"
	"fishnet-inject/sugar"
	"fishnet-inject/utils"
	"fishnet-inject/webhooks"
	"github.com/spf13/cobra"
	"os"
)

// use example
var examples = `
  # running on the specified port:
  $ fishnet injector -p 9527

  # running on the specified log level:
  $ fishnet injector -v debug
	
  # running on specified tls config:
  $ fishnet injector -v info -d /tls --cert ca.pem --key key.pem
`

// flags
var (
	varLogLevel string
	varPort     int
	varCertDir  string
	varCertName string
	varKeyName  string
)

// Cmd injector command
var (
	Cmd = &cobra.Command{
		Use:     "injector",
		Short:   "control sidecar auto-injection",
		Aliases: []string{"ij"},
		Example: utils.Rainbow(examples),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: 后续我们可以做成Factory模式

			sugar.InitLogger(varLogLevel)
			wsc := webhooks.WebhookServerConfig{
				Port:     varPort,
				CertDir:  varCertDir,
				CertName: varCertName,
				KeyName:  varKeyName,
			}
			wsc.Default()

			// init injector webhook
			sre := webhooks.NewWebhookServer(wsc)

			sugar.Infof("fishnet injector webhook running on %d", wsc.Port)
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
	Cmd.Flags().StringVarP(&varCertDir, "certdir", "d", "", "set tls cert and private key storage directory. (Default current directory)")
	Cmd.Flags().StringVarP(&varCertName, "cert", "", "", "set tls cert file. (Default tls.pem)")
	Cmd.Flags().StringVarP(&varKeyName, "key", "", "", "set tls private key file. (Default tls-key.pem)")
}

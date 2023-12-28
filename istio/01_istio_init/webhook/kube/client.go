package kube

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type Client struct {
	url        string
	kubeconfig string
	restConfig *rest.Config
}

type ClientConfig struct {
	Host       string
	Port       string
	KubeConfig string
}

func NewClient(cc ClientConfig) (*Client, error) {

	var url string
	var configPath string
	if cc.Host != "" && cc.Port != "" {
		url = fmt.Sprintf("%s:%s", cc.Host, cc.Port)
	}

	if cc.KubeConfig == "" {
		configPath = "D:\\treasure\\istio\\01_istio_init\\webhook\\kubeconfig"
	} else {
		configPath = cc.KubeConfig
	}

	cli := &Client{
		url:        url,
		kubeconfig: configPath,
	}
	if err := cli.conn(); err != nil {
		return nil, err
	}

	return cli, nil
}

func (c *Client) conn() error {

	cliRestConfig, err := clientcmd.BuildConfigFromFlags(c.url, c.kubeconfig)
	if err != nil {
		log.Println("remote connection error: ", err)

		c.restConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Println("in cluster connection error: ", err)
			return err
		}
	}
	c.restConfig = cliRestConfig

	return nil
}

func (c *Client) ClientSet() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(c.restConfig)
}

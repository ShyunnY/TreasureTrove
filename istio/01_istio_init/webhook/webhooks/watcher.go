package webhooks

// Watcher
// TODO: 1. 调用configMap informer进行获取cm
// TODO: 2. 将配置设置进config中
type Watcher interface {
	SetCallback()

	Run(stop <-chan interface{})
}

var _ Watcher = &ConfigMapWatcher{}

type ConfigMapWatcher struct {
	callbacks interface{}
}

func (c *ConfigMapWatcher) SetCallback() {

}

func (c *ConfigMapWatcher) Run(stop <-chan interface{}) {

}

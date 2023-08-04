// Package plugin is the plugin package.
package plugin

import (
	"trpc.group/trpc-go/trpc-go/log"

	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName = "custom"
	pluginType = "custom"
)

func init() {
	plugin.Register(pluginName, &customPlugin{})
}

// customPlugin struct implements plugin.Factory interface.
type customPlugin struct {
	config customConfig
}

var c customPlugin

// customConfig plugin config
type customConfig struct {
	Test    string `yaml:"test"`
	TestObj struct {
		Key1 string `yaml:"key1"`
		Key2 bool   `yaml:"key2"`
		Key3 int32  `yaml:"key3"`
	} `yaml:"test_obj"`
}

// Type return plugin type
func (custom *customPlugin) Type() string {
	return pluginType
}

// Setup init plugin
// trpc will call Setup function to init plugin.
func (custom *customPlugin) Setup(name string, decoder plugin.Decoder) error {

	if err := decoder.Decode(&c.config); err != nil {
		return err
	}

	log.Infof("[plugin] init customPlugin success, config: %v", c.config)

	return nil
}

// Record is a custom plugin function
// you can call this function in your code print plugin config.
func Record() {
	log.Infof("[plugin] call key1 : %s, key2 : %t, key3 : %d",
		c.config.TestObj.Key1, c.config.TestObj.Key2, c.config.TestObj.Key3)
}

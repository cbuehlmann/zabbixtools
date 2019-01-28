package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"os"
	"testing"
)

const CONFIGURATION_EXAMPLE = "conf/example.yaml"

func TestReadConfigurationFromFile(t *testing.T) {
	configuration := ReadConfigurationFromFile(CONFIGURATION_EXAMPLE)
	validateConfiguration(t, configuration)
}

func TestExampleConfiguration(t *testing.T) {

	file, err := os.Open(CONFIGURATION_EXAMPLE)
	if err != nil {
		t.Error(err)
	}

	info, err := os.Stat(file.Name())

	data := make([]byte, info.Size())
	_, err = file.Read(data)
	if err != nil {
		t.Error(err)
	}

	configuration := Configuration{}
	fmt.Printf("parsing yaml from %s %s\n", file.Name(), string(data))

	err = yaml.Unmarshal([]byte(data), &configuration)
	if err != nil {
		t.Error(err)
	}

	validateConfiguration(t, configuration)
}

func validateConfiguration(t *testing.T, configuration Configuration) {
	assert.NotNil(t, configuration.Zabbix)
	assert.NotNil(t, configuration.Zabbix.Api)

	assert.Equal(t, "http://127.0.0.1/zabbix/api_jsonrpc.php", configuration.Zabbix.Api.URL)
	assert.Equal(t, "api-user", configuration.Zabbix.Api.Username)
	assert.Equal(t, "zabbix", configuration.Zabbix.Api.Password)

	assert.Equal(t, "127.0.0.1", configuration.Zabbix.Trapper.Host)
	assert.Equal(t, 10051, configuration.Zabbix.Trapper.Port)
}

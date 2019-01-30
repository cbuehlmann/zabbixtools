package zabbix

import (
	"encoding/json"
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"os"
	"testing"
)

const CONFIGURATION_EXAMPLE = "../conf/example.yaml"

func TestReadConfigurationFromFile(t *testing.T) {
	configuration, err := ReadConfigurationFromFile(CONFIGURATION_EXAMPLE)
	assert.Nil(t, err)
	validateConfiguration(t, configuration)
}

func TestFilter(t *testing.T) {
	Log.SetHandler(log15.StdoutHandler)
	configuration, err := ReadConfigurationFromFile(CONFIGURATION_EXAMPLE)
	assert.Nil(t, err)
	assert.NotNil(t, configuration.Templates)
	assert.Equal(t, 2, len(configuration.Templates))
	fmt.Fprintf(os.Stdout, "template 0 %v\n", configuration.Templates[0])
	fmt.Fprintf(os.Stdout, "template 1 %v\n", configuration.Templates[1])
}

func TestFilterToJson(t *testing.T) {
	Log.SetHandler(log15.StdoutHandler)
	configuration, err := ReadConfigurationFromFile(CONFIGURATION_EXAMPLE)
	assert.Nil(t, err)
	assert.NotNil(t, configuration.Templates)
	assert.Equal(t, 2, len(configuration.Templates))
	fmt.Fprintf(os.Stdout, "template 0: %v\n", configuration.Templates[0])
	fmt.Fprintf(os.Stdout, "template 1: %v\n", configuration.Templates[1])

	jsonString, err := json.Marshal(configuration.Templates[0])
	assert.Nil(t, err)
	fmt.Fprintf(os.Stdout, "template 0 to json: %s\n", jsonString)

	jsonString, err = json.Marshal(configuration.Templates[1])
	assert.Nil(t, err)
	fmt.Fprintf(os.Stdout, "template 1 to json: %s\n", jsonString)

	assert.Equal(t, 2, len(configuration.Items))
	fmt.Fprintf(os.Stdout, "template 1 item filter: %v\n", configuration.Items[0])
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

	fmt.Printf("parsed Configuration structure %v\n", configuration)

	validateConfiguration(t, configuration)
}

func validateConfiguration(t *testing.T, configuration Configuration) {
	assert.NotNil(t, configuration.Zabbix)
	assert.NotNil(t, configuration.Zabbix.Api)

	assert.Equal(t, "http://127.0.0.1/zabbix/api_jsonrpc.php", configuration.Zabbix.Api.URL)
	assert.Equal(t, "zabbixapi-user", configuration.Zabbix.Api.Username)
	assert.Equal(t, "zabbixapi-pw", configuration.Zabbix.Api.Password)

	assert.Equal(t, "127.0.0.1", configuration.Zabbix.Trapper.Host)
	assert.Equal(t, 10051, configuration.Zabbix.Trapper.Port)
}

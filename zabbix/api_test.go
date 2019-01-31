package zabbix

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var session Session

/**
 * setup / before each test
 */
func TestMain(m *testing.M) {
	configuration, err := ReadConfigurationFromFile("integration.yaml")
	if err != nil {
		fmt.Fprintln(os.Stdout, "missing configuration file 'integration.yaml' to run integration tests against a ZABBIX server")
		os.Exit(0)
	}
	session.URL = configuration.Zabbix.Api.URL
	err = Login(&session, configuration.Zabbix.Api.Username, configuration.Zabbix.Api.Password)
	if err != nil {
		panic(err)
	}
	Log.SetHandler(log15.StdoutHandler)
	os.Exit(m.Run())
}

func TestHostQuery(t *testing.T) {
	assert.True(t, session.Token != "")
	Log.SetHandler(log15.StdoutHandler)
	filter := HostFilterConfiguration{Search: make(map[string][]string)}
	filter.Search["host"] = []string{"b*"}
	query := session.NewHostQuery([]string{"10001"}, filter.Filter, filter.Search)
	result := query.Query()
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	for index, host := range result {
		fmt.Fprintf(os.Stdout, "[%d] %s host: %s\n", index, host.Name, host.Host)
	}
}

func TestItemQuery(t *testing.T) {
	assert.True(t, session.Token != "")
	Log.SetHandler(log15.StdoutHandler)
	filter := HostFilterConfiguration{Search: make(map[string][]string)}
	filter.Search["key_"] = []string{"net.if*"}
	query := session.NewItemQuery([]string{"10109"}, filter.Filter, filter.Search)
	result := query.Query()
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	for index, item := range result {
		fmt.Fprintf(os.Stdout, "[%d] %s key: %s\n", index, item.Name, item.Key)
	}
}

type blankHostQuery struct {
}

func TestHostQueryAll(t *testing.T) {
	assert.True(t, session.Token != "")

	req := blankHostQuery{}
	response := hostQueryResponse{}
	request := Request{session: session, method: "host.get", request: req, response: &response}
	request.query()

	result := response.Elements
	//	query := session.NewHostQuery([]string{}, map[string][]string {}, map[string][]string {})
	//	result := query.Query()
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	for index, host := range result {
		fmt.Fprintf(os.Stdout, "[%d] %s %s host: %s\n", index, host.HostID, host.Name, host.Host)
	}
}

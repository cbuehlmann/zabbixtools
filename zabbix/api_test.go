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
		panic(err)
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

	filter := HostFilterConfiguration{}
	query := session.NewHostQuery([]string{"10001"}, filter.Filter, filter.Search)
	result := query.Query()
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	for index, host := range result {
		fmt.Fprintf(os.Stdout, "[%d] %s host: %s\n", index, host.Name, host.Host)
	}
}

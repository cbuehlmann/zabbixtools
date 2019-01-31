package zabbix

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/stretchr/testify/assert"
	"os"
	"regexp"
	"testing"
	"time"
)

var session Session

// manual or by running all tests
const HOST = "10109"
const ITEM = "23973"

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

func TestHostLogin(t *testing.T) {
	Log.SetHandler(log15.StdoutHandler)
	configuration, err := ReadConfigurationFromFile("integration.yaml")
	Login(&session, configuration.Zabbix.Api.Username, configuration.Zabbix.Api.Password)
	match, err := regexp.Match("", []byte(session.ServerVersion))
	assert.Nil(t, err)
	assert.True(t, match)
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
	query := session.NewItemQuery([]string{HOST}, filter.Filter, filter.Search)
	result := query.Query()
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	for index, item := range result {
		fmt.Fprintf(os.Stdout, "[%d] %s key: %s\n", index, item.Name, item.Key)
	}
}

func TestTrendQuery(t *testing.T) {
	assert.True(t, session.Token != "")
	Log.SetHandler(log15.StdoutHandler)
	now := time.Now()
	oneDay, err := time.ParseDuration("24h")
	assert.Nil(t, err)

	for i := 0; i < 10; i++ {
		now = now.Add(-oneDay)
	}

	query := session.NewTrendQuery([]string{ITEM}, now.Add(-oneDay), now) // one hour back
	result := query.Query()
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	for index, value := range result {
		fmt.Fprintf(os.Stdout, "[%d] @ %s %v\n", index, time.Unix(value.Clock, 0).Format("Mon 01-02 15:04:05"), value)
	}
}

func TestHistoryQuery(t *testing.T) {
	assert.True(t, session.Token != "")
	Log.SetHandler(log15.StdoutHandler)

	to := time.Now()

	oneDay, err := time.ParseDuration("24h")
	assert.Nil(t, err)

	// 10 days back
	for i := 0; i < 10; i++ {
		to = to.Add(-oneDay)
	}

	query := session.NewHistoryQuery()
	query.ValueType = 3
	query.Items = []string{"23973"}
	query.From = to.Add(-oneDay).Unix() // one day back
	query.To = to.Unix()

	fmt.Fprintf(os.Stdout, "searching in range %s to %s\n",
		time.Unix(query.From, 0).Format("Mon 01-02 15:04:05"),
		time.Unix(query.To, 0).Format("Mon 01-02 15:04:05"))

	result := query.Query()
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	for index, value := range result {
		fmt.Fprintf(os.Stdout, "[%d] @ %s %v\n", index, time.Unix(value.Clock, 0).Format("Mon 01-02 15:04:05"), value)
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

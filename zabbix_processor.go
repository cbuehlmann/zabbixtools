package main

import (
	"flag"
	"fmt"
	"github.com/cbuehlmann/zabbixtools/zabbix"
	log "github.com/inconshreveable/log15"
	"math"
	"os"
	"strconv"
	"time"
)

/**
 * Concept:
 *  1 Find Host ID's
 *  1a filter by Template
 *  1b filter by Host-Filter
 *  2 Find Items (filter by Host ID's)
 *  3 Process items -> store processing value
 * [4] push data back to server
 */

var Log = log.New()
var destination = os.Stdout

// Template ID to Template Name
var templates map[string]string = make(map[string]string, 0)

// Host ID to Host Name
var hosts map[string]string = make(map[string]string, 0)

func main() {
	var err error

	// configuration
	configfile := flag.String("config", "~/.zabbix_processor.yml", "configuration file")

	apiUrl := flag.String("url", "", "ZABBIX frontend/API URL")
	username := flag.String("username", "", "ZABBIX username")
	password := flag.String("password", "", "ZABBIX password")

	// operation
	allhosts := flag.Bool("all", false, "if no hosts are found, work on all hosts. this might block your server for a long time")
	verbose := flag.Bool("verbose", false, "be verbose (log level debug)")
	quiet := flag.Bool("quiet", false, "just print result. overrides -verbose")
	output := flag.String("output", "-", "destination for processed values")

	flag.Parse()

	var configuration zabbix.Configuration

	if *configfile != "" {
		var err error
		configuration, err = zabbix.ReadConfigurationFromFile(*configfile)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "unable to parse configuration file", *configfile, err)
			os.Exit(2)
		}
	} else {
		configuration = zabbix.Configuration{}
	}

	handler := log.StdoutHandler
	if *verbose == false {
		handler = log.LvlFilterHandler(log.LvlInfo, log.StdoutHandler)
	}
	if *quiet {
		handler = log.DiscardHandler()
	}

	// without filter
	Log.SetHandler(handler)
	zabbix.Log.SetHandler(handler)

	if *username != "" {
		if configuration.Zabbix.Api.Username != "" {
			Log.Debug("username from command line overrides configuration value")
		}
		configuration.Zabbix.Api.Username = *username
	}

	if *password != "" {
		if configuration.Zabbix.Api.Password != "" {
			Log.Debug("password from command line overrides configuration value")
		}
		configuration.Zabbix.Api.Username = *password
	}

	if *apiUrl != "" {
		if configuration.Zabbix.Api.URL != "" {
			Log.Debug("api uri from command line overrides configuration value")
		}
		configuration.Zabbix.Api.URL = *apiUrl
	}

	if *output != "-" {
		destination, err = os.OpenFile(*output, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "cannot write file ", *output, err.Error())
		}
	}

	session := zabbix.Session{URL: configuration.Zabbix.Api.URL}
	Log.Info("authenticating", "server", session.URL)
	err = zabbix.Login(&session, configuration.Zabbix.Api.Username, configuration.Zabbix.Api.Password)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "login failed", err)
		os.Exit(3)
	}
	Log.Info("login successful", "token", session.Token)

	collectHostsByTemplate(session, configuration)
	collectHosts(session, configuration)

	if len(hosts) == 0 && *allhosts == false {
		Log.Warn("no hosts found by filter. to process all hosts, use the --all command line option")
		return
	}

	findItems(session, configuration)

	destination.Close()
}

func fetch(session zabbix.Session, item zabbix.ItemResponseElement, date time.Time, window time.Duration) []zabbix.HistoryValue {
	query := session.NewHistoryQuery()
	query.ValueType = item.ValueType
	query.Items = []string{item.ItemID}

	query.From = date.Add(-window).Unix()
	query.To = date.Add(+window).Unix()

	Log.Debug("loading history for item", "item", item,
		"from", time.Unix(query.From, 0).Format("Mon 01-02 15:04:05"),
		"to", time.Unix(query.To, 0).Format("Mon 01-02 15:04:05"))

	values := query.Query()
	return values
}

func getClosestValue(timepoint time.Time, values []zabbix.HistoryValue) zabbix.HistoryValue {
	closest := 3600.0 * 24 * 356 // 1Y
	index := -1
	for i, value := range values {
		if closest > math.Abs(float64(timepoint.Unix()-value.Clock)) {
			index = i
		}
	}
	if index >= 0 {
		return values[index]
	} else {
		return zabbix.HistoryValue{}
	}

}

/**
 * Fetch n weeks back.
 */
func compareWeeks(session zabbix.Session, item zabbix.ItemResponseElement, weeks int, window time.Duration) (float64, time.Time) {

	now := time.Now()
	// now fetch latest value
	values := fetch(session, item, now.Add(-window), window)
	if len(values) == 0 {
		Log.Info("no current value found in window",
			"from", now.Add(-window).Format("01-02 15:04:05"),
			"to", now.Add(window).Format("01-02 15:04:05"))
		return math.NaN(), now
	}
	current, _ := strconv.ParseFloat(values[0].Value, 64)
	// Sample timepoint
	timestamp := time.Unix(values[0].Clock, values[0].Nano)
	Log.Info("current value", "value", current, "exact timepoint", timestamp.Format("Mon 01-02 15:04:05"))

	historicValues := make([]float64, 0)
	// search with the exact timestamp of most recent sample
	tp := timestamp
	oneWeek := time.Hour * 24 * 7
	for i := 0; i < weeks; i++ {
		tp = tp.Add(-oneWeek) // step one week back
		closest := getClosestValue(tp, fetch(session, item, tp, window))
		if closest.Clock != 0 {
			value, _ := strconv.ParseFloat(closest.Value, 64)
			historicValues = append(historicValues, value)
			when := time.Unix(closest.Clock, closest.Nano)
			Log.Info("historic value", "value", value, "date", when.Format("Mon 01-02 15:04:05"))
		} else {
			Log.Warn("missing historic value", "around", tp.Format("Mon 01-02 15:04:05"))
		}
	}

	historic := average(historicValues)
	Log.Info("calculation done", log.Ctx{"average": historic, "current": current, "difference": current - historic})

	return current - historic, timestamp

}

func average(values []float64) float64 {
	sum := float64(0)
	for _, value := range values {
		sum = sum + value
	}
	return sum / float64(len(values))
}

/**
 * Find matching Hosts by template filter
 */
func collectHostsByTemplate(session zabbix.Session, configuration zabbix.Configuration) {

	for index, templateConfiguration := range configuration.Templates {
		Log.Debug("filtering templateHits with", "filter", templateConfiguration, "index", index)

		req := session.NewTemplateQuery(templateConfiguration.Filter, templateConfiguration.Search)
		templateHits := req.Query()

		Log.Debug("processing matching templateHits", "templateHits", templateHits)
		for _, template := range templateHits {
			Log.Debug("adding template", "id", template.TemplateId, "name", template.Name)
			templates[template.TemplateId] = template.Name
		}
		Log.Info("collected templates id's", "templateids", keysFromMap(templates))
	}
}

/**
 * Collect host details
 */
func collectHosts(session zabbix.Session, configuration zabbix.Configuration) {

	// collect hosts linked with templates
	if len(templates) > 0 {
		keys := keysFromMap(templates)
		hostQuery := session.NewHostQuery(keys, nil, nil)
		hostElements := hostQuery.Query()
		for _, hostElement := range hostElements {
			hosts[hostElement.HostID] = hostElement.Name
		}

		Log.Debug("collected hosts via template lookup", "hosts", hosts)
	}

	// collect hosts by filters
	if len(configuration.Hosts) > 0 {

		for index, hostConfiguration := range configuration.Hosts {
			hostQuery := session.NewHostQuery([]string{}, hostConfiguration.Filter, hostConfiguration.Search)
			hostElements := hostQuery.Query()
			for _, hostElement := range hostElements {
				hosts[hostElement.HostID] = hostElement.Name
			}
			Log.Debug("collected hosts via host filter", "hosts", hosts, "index", index)
		}

	}

	Log.Info("working with the following hosts", "hosts", hosts)
}

func keysFromMap(input map[string]string) []string {
	keys := make([]string, 0)
	for key := range input {
		keys = append(keys, key)
	}
	return keys
}

func findItems(session zabbix.Session, configuration zabbix.Configuration) {
	for index, itemFilter := range configuration.Items {
		Log.Debug("processing items of filter", "index", index)
		query := session.NewItemQuery(keysFromMap(hosts), itemFilter.Filter, itemFilter.Search)
		query.SearchWildcardsEnabled = true
		items := query.Query()

		if len(items) > 0 {
			// find all active hosts
			processItems(session, items, itemFilter)
		} else {
			Log.Warn("no items found", "hosts", keysFromMap(hosts))
		}
	}
}

func processItems(session zabbix.Session, items []zabbix.ItemResponseElement, itemConfiguration zabbix.ItemConfiguration) {
	for index, item := range items {
		Log.Info(fmt.Sprintf("processing item %d/%d", index, len(items)), "itemid", item.ItemID, "key", item.Key, "data", item)
		if itemConfiguration.PastWeeks.Weeks > 0 {

			halfWindow := time.Duration(itemConfiguration.PastWeeks.Window / 2)
			value, timestamp := compareWeeks(session, item, itemConfiguration.PastWeeks.Weeks, halfWindow*time.Second)
			if math.IsNaN(value) == false {
				addSenderLine(hosts[item.HostID], item.Key, itemConfiguration.Postfix, timestamp, value)
			} else {
				Log.Warn("skipping item due to missing data", "item", item)
			}
		}
	}
}

func addSenderLine(hostname string, key string, postfix string, timestamp time.Time, value float64) {
	line := fmt.Sprintf("\"%s\" %s%s %d %f\n", hostname, key, postfix, timestamp.Unix(), value)
	Log.Info("appending zabbix_sender line", "line", line)
	destination.WriteString(line)
}

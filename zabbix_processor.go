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

// Host ID to Host Name
var hosts map[string]string

// Template ID to Template Name
var templates map[string]string

func fetch(session zabbix.Session, item string, date time.Time, window time.Duration) []zabbix.Value {
	query := session.NewHistoryQuery()
	query.History = 3
	query.Items = item

	query.From = date.Add(-window).Unix()
	query.To = date.Add(+window).Unix()

	values := query.Query()
	return values
}

func getClosestValue(timepoint time.Time, values []zabbix.Value) zabbix.Value {
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
		return zabbix.Value{}
	}

}

/**
 * Fetch n weeks back.
 */
func compareWeeks(session zabbix.Session, item string, weeks int, window time.Duration) (float64, time.Time) {

	now := time.Now()
	// now fetch latest value
	values := fetch(session, item, now.Add(-window), window)
	if len(values) == 0 {
		Log.Error("no current value found in window",
			"from", now.Add(-window).Format("01-02 15:04:05"),
			"to", now.Add(window).Format("01-02 15:04:05"))
		os.Exit(10)
	}
	current, _ := strconv.ParseFloat(values[0].Value, 64)
	// Sample timepoint
	timestamp := time.Unix(values[0].Clock, values[0].Nano)
	Log.Info("current value", "value", current, "exact timepoint", timestamp.Format("Mon 01-02 15:04:05"))

	historicValues := make([]float64, weeks)
	tp := timestamp
	oneWeek := time.Hour * 24 * 7
	for i := 0; i < weeks; i++ {
		tp = tp.Add(-oneWeek) // step one week back
		closest := getClosestValue(tp, fetch(session, item, tp, window))
		if closest.Clock != 0 {
			value, _ := strconv.ParseFloat(closest.Value, 64)
			historicValues[i] = value
			when := time.Unix(closest.Clock, closest.Nano)
			Log.Info("historic value", log.Ctx{"value": value, "date": when.Format("Mon 01-02 15:04:05")})
		} else {
			historicValues[i] = math.NaN()
		}
	}

	historic := average(historicValues)
	Log.Info("calculation done", log.Ctx{"average": historic, "current": current, "difference": current - historic})

	return current - historic, timestamp

}

func average(values []float64) float64 {
	sum := float64(0)
	count := 0.0
	for _, value := range values {
		if value != math.NaN() {
			sum = sum + value
			count += 1.0
		}
	}
	return sum / count
}

func main() {
	var err error

	// configuration
	configfile := flag.String("config", "~/.zabbix_processor.yml", "configuration file")

	apiUrl := flag.String("url", "", "ZABBIX frontend/API URL")
	username := flag.String("username", "", "ZABBIX username")
	password := flag.String("password", "", "ZABBIX password")

	// operation
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

	s := zabbix.Session{URL: configuration.Zabbix.Api.URL}
	Log.Info("authenticating", "server", s.URL)
	err = zabbix.Login(&s, configuration.Zabbix.Api.Username, configuration.Zabbix.Api.Password)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "login failed", err)
		os.Exit(3)
	}
	Log.Info("login successful", "token", s.Token)

}

/**
 * Find matching Hosts by template filter
 */
func processTemplate(session zabbix.Session, configuration zabbix.Configuration) {

	for index, templateConfiguration := range configuration.Templates {
		Log.Debug("filtering templateHits with", "filter", templateConfiguration, "index", index)

		req := session.NewTemplateQuery(templateConfiguration.Filter, templateConfiguration.Search)
		templateHits := req.Query()

		Log.Info("processing matching templateHits", "templateHits", templateHits)

		if templateHits != nil {

			for _, template := range templateHits {
				Log.Info("adding template", log.Ctx{"id": template.TemplateId, "name": template.Name})
				templates[template.TemplateId] = template.Name
			}

		} else {
			_, _ = fmt.Fprintln(os.Stderr, "failed to read templateHits")
			os.Exit(3)
		}
	}
}

func findHosts(session zabbix.Session, configuration zabbix.Configuration) {

	if len(templates) > 0 {
		keys := make([]string, len(templates))
		for key := range templates {
			keys = append(keys, key)
		}
		hostQuery := session.NewHostQuery(keys, []string{}, templateConfiguration.Hosts.Search)
	}
	hostQuery := session.NewHostQuery([]string{template.TemplateId}, templateConfiguration.Hosts.Filter, templateConfiguration.Hosts.Search)

	for index, hostConfiguration := range configuration.Hosts {

		hostQuery := session.NewHostQuery([]string{template.TemplateId}, templateConfiguration.Hosts.Filter, templateConfiguration.Hosts.Search)
		hosts := hostQuery.Query()

	}
}

func filterItems() {
	if len(templateConfiguration.Items) > 0 {
		for _, itemFilter := range templateConfiguration.Items {
			query := session.NewItemQuery([]string{template.TemplateId}, itemFilter.Filter, itemFilter.Search)
			items := query.Query()

			if len(items) > 0 {
				// find all active hosts
				hostQuery := session.NewHostQuery([]string{template.TemplateId}, templateConfiguration.Hosts.Filter, templateConfiguration.Hosts.Search)
				hosts := hostQuery.Query()

				processItems(session, items, hosts, template, itemFilter)
			} else {
				Log.Warn("no items found on template", "template", template.Name)
			}
		}
	} else {
		// no item filters, process all items!
		Log.Warn("no item filter criteria for template", "template", template.Name)
		//query := session.NewItemQuery([]string{template.TemplateId}, nil, nil)
		//processItems(session, query, template, itemFilter)
	}

}

func processItems(session zabbix.Session, items []zabbix.ItemResponseElement, hosts []zabbix.HostResponseElement, template zabbix.TemplateResponseItem, itemConfiguration zabbix.ItemConfiguration) {
	for _, item := range items {
		Log.Info("found item", "itemid", item.ItemID, "key", item.Key, "data", item)
		if itemConfiguration.PastWeeks.Weeks > 0 {

			halfWindow := time.Duration(itemConfiguration.PastWeeks.Window / 2)
			value, timestamp := compareWeeks(session, item.ItemID, itemConfiguration.PastWeeks.Weeks, halfWindow*time.Second)

			addSenderLine(item.HostID, item.Key+itemConfiguration.Postfix, timestamp, value)

		}
	}
}

func addSenderLine(hostname string, itemkey string, timestamp time.Time, value float64) {
	line := fmt.Sprintf("\"%s\" %s %d %f", hostname, itemkey, timestamp.Unix(), value)
	Log.Info("appending zabbix_sender line", "line", line)
	destination.WriteString(line)
}

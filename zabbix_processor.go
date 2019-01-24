package main

import (
	"./zabbix"
	"flag"
	"fmt"
	log "github.com/inconshreveable/log15"
	"math"
	"os"
	"strconv"
	"time"
)

var Log = log.New()

func fetch(session zabbix.Session, item int, date time.Time, window time.Duration) []zabbix.Value {
	query := zabbix.NewHistoryQuery()
	query.History = 0
	query.Items = strconv.Itoa(item)

	query.From = date.Add(-window).Unix()
	query.To = date.Add(+window).Unix()

	values, err := zabbix.History(session, query)
	if err != nil {
		return nil
	}
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
func compareWeeks(session zabbix.Session, item int, weeks int, window time.Duration) float64 {

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
	Log.Info("current value", "value", current, "exact timepoint", time.Unix(values[0].Clock, values[0].Nano).Format("Mon 01-02 15:04:05"))

	historicValues := make([]float64, weeks)
	tp := now
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

	return current - historic

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
	Log.SetHandler(log.DiscardHandler())

	// data access
	apiUrl := flag.String("url", "http://127.0.0.1/zabbix", "ZABBIX webfrontend base url. /api_jsonrpc.php will be appended")
	username := flag.String("username", "", "ZABBIX username")
	password := flag.String("password", "", "ZABBIX password")
	itemId := flag.Int("itemid", 37364, "ZABBIX item id. obtain it from the frontend: url: http:/127.0.0.1/zabbix/history.php?action=showgraph&itemid=37364&sid=23bue3f84af4c3")

	// algorithm parameters
	weeks := flag.Int("weeks", 3, "numbers of weeks back")
	window := flag.Int64("window", 3600, "historic value search window size in seconds. 3600 for 1 hour")

	// configure output
	command := flag.Bool("command", false, "print ZABBIX trapper command template")
	verbose := flag.Bool("verbose", false, "just print result")

	flag.Parse()

	if *username == "" {
		_, _ = fmt.Fprintln(os.Stderr, "missing username. provide option -username=")
		os.Exit(2)
	}

	if *verbose {
		Log.SetHandler(log.StdoutHandler)
		zabbix.Log.SetHandler(log.StdoutHandler)
	}

	s := zabbix.Session{URL: *apiUrl + "/api_jsonrpc.php"}
	Log.Info("authenticating", "server", s.URL)
	err := zabbix.Login(&s, *username, *password)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "login failed", err)
		os.Exit(3)
	}

	Log.Info("login successful", "token", s.Token)
	halfWindow := time.Duration(*window / 2)
	result := compareWeeks(s, *itemId, *weeks, halfWindow*time.Second)

	if *command {
		Log.Debug("print zabbix_sender command template to stdout", "value", result)
		// ./zabbix_sender -z zabbix -s "Linux DB3" -k db.connections -o 43
		fmt.Printf("zabbix_sender -z ${SERVER} -s \"${SENDERNAME}\" -k ${ITEMKEY} -o %f", result)
	} else {
		Log.Debug("writing difference to stdout", "value", result)
		// just print the bare value
		fmt.Printf("%f", result)
	}

}

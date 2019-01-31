package zabbix

/**
 * According to https://www.zabbix.com/documentation/current/manual/api
 */

import (
	"bytes"
	"encoding/json"
	"fmt"
	logging "github.com/inconshreveable/log15"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

var Log = logging.New()
var requestEnumerator int64 = rand.New(rand.NewSource(time.Now().Unix())).Int63()

const contentType string = "application/json-rpc"

type Session struct {
	// ZABBIX web frontend base url
	URL string
	// reuse connection HTTP 1.1
	Connection http.Client
	// Authentication Token
	Token string

	ServerVersion string
}

type Request struct {
	session  Session
	method   string
	request  interface{}
	response interface{}
}

// Note: json.Marshal does only process fields with upper case name
type request struct {
	Encoding string      `json:"jsonrpc"` // "2.0"
	Method   string      `json:"method"`  // example: "user.login"
	Params   interface{} `json:"params"`
	Id       int64       `json:"id"` // request id
	Auth     string      `json:"auth,omitempty"`
}

type auth struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type apiError struct {
	Code    int    `json:"error"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type loginResponse struct {
	Encoding string `json:"jsonrpc"` // "2.0"
	Result   string `json:"result"`
	//Id       string `json:"id"` // referencing request id
	// Error reporting
	Error apiError `json:"error"`
}

/**
* Refer to https://www.zabbix.com/documentation/4.0/manual/api/reference/history/get
 */
type HistoryQuery struct {
	ValueType int      `json:"history"`             // 0 - numeric float; 1 - character; 2 - log; 3 - numeric unsigned; 4 - text.
	Output    string   `json:"output"`              // extend | count
	Hosts     string   `json:"hostids,omitempty"`   // host ids, numeric
	Items     []string `json:"itemids"`             // item ids, numeric
	From      int64    `json:"time_from,omitempty"` // timerange start. seconds since epoch
	To        int64    `json:"time_till,omitempty"` // timerange end. seconds since epoch
	Limit     int      `json:"limit,omitempty"`     // limit number of records
	SortField string   `json:"sortfield"`           // clock|value|ns
	SortOrder string   `json:"sortorder,omitempty"` // DESC|ASC

	session Session
}

type historyQueryResponse struct {
	Encoding string         `json:"jsonrpc"` // "2.0"
	Items    []HistoryValue `json:"result"`

	//	Id       string  `json:"id"` // referencing request id
}

type HistoryValue struct {
	Value string `json:"value"`
	Item  string `json:"itemid"`
	Clock int64  `json:"clock,string"` // seconds since epoch
	Nano  int64  `json:"ns,string"`    // nanoseconds
}

/**
* Refer to https://www.zabbix.com/documentation/4.0/manual/api/reference/trend/get
 */
type TrendQuery struct {
	Items     []string `json:"itemids"` // item ids, numeric
	Trend     int      // 0 - numeric float; 1 - character; 2 - log; 3 - numeric unsigned; 4 - text.
	From      int64    `json:"time_from,omitempty"` // timerange start. seconds since epoch
	To        int64    `json:"time_till,omitempty"` // timerange end. seconds since epoch
	Limit     int      `json:"limit,omitempty"`     // limit number of records
	Output    []string `json:"output"`              // field selection "itemid", "clock", "num", "value_min", "value_avg", "value_max"
	SortField string   `json:"sortfield"`           // clock|value|ns
	SortOrder string   `json:"sortorder,omitempty"` // DESC|ASC

	session Session
}

type trendQueryResponse struct {
	Encoding string       `json:"jsonrpc"` // "2.0"
	Items    []TrendValue `json:"result"`
	//	Id       string  `json:"id"` // referencing request id
}

type TrendValue struct {
	Item     string `json:"itemid"`
	Clock    int64  `json:"clock,string"` // seconds since epoch
	Num      int64  `json:"num,string"`   // number of values per hour
	MinValue string `json:"value_min"`
	AvgValue string `json:"value_avg"`
	MaxValue string `json:"value_max"`
}

/**
* Refer to https://www.zabbix.com/documentation/4.0/manual/api/reference/template/get
 */
type TemplateQuery struct {
	Output                 string              `json:"output"` // extend | count
	Filter                 map[string][]string `json:"filter,omitempty"`
	Search                 map[string][]string `json:"search,omitempty"`
	SearchWildcardsEnabled bool                `json:"searchWildcardsEnabled"`

	session Session
}

type templateQueryResponse struct {
	Encoding string                 `json:"jsonrpc"` // "2.0"
	Elements []TemplateResponseItem `json:"result"`  // Elements
	//	Id       string  `json:"id"` // referencing request id
}

type TemplateResponseItem struct {
	Host       string
	Name       string
	TemplateId string
}

/**
* Refer to https://www.zabbix.com/documentation/4.0/manual/api/reference/item/get
 */
type ItemQuery struct {
	TemplateIDs            []string            `json:"templateids,omitempty"` // search for specific template id's
	HostIDs                []string            `json:"hostids,omitempty"`
	Output                 string              `json:"output"`           // extend | count
	Filter                 map[string][]string `json:"filter,omitempty"` // possible filter
	Search                 map[string][]string `json:"search,omitempty"` // possible search criteria
	SearchWildcardsEnabled bool                `json:"searchWildcardsEnabled"`

	SortField []string

	session Session
}

type ItemResponseElement struct {
	ItemID      string `json:"itemid"`
	HostID      string `json:"hostid"`
	ValueType   int    `json:"value_type,string"` // 0 - numeric float; 1 - character; 2 - log; 3 - numeric unsigned; 4 - text.
	Key         string `json:"key_"`              // Item key
	Delay       string // sample interval in seconds
	Name        string
	TemplateID  string
	Description string
}

type itemQueryResponse struct {
	Encoding string                `json:"jsonrpc"` // "2.0"
	Elements []ItemResponseElement `json:"result"`  // items
}

/**
* Refer to https://www.zabbix.com/documentation/4.0/manual/api/reference/host/get
 */
type HostQuery struct {
	TemplateIDs            []string            `json:"templateids,omitempty"` // search for specific template id's
	Output                 string              `json:"output"`                // extend | count
	Filter                 map[string][]string `json:"filter,omitempty"`
	Search                 map[string][]string `json:"search,omitempty"`
	SearchWildcardsEnabled bool                `json:"searchWildcardsEnabled"`
	IncludeTemplates       bool                `json:"templated_hosts"` // Return both hosts and templates.
	IncludeMonitored       bool                `json:"monitored_hosts"` // Return only monitored hosts.

	SortField []string

	session Session
}

type hostQueryResponse struct {
	Encoding string                `json:"jsonrpc"` // "2.0"
	Elements []HostResponseElement `json:"result"`  // elements
	//	Id       string  `json:"id"` // referencing request id
}

type HostResponseElement struct {
	HostID     string `json:"hostid"`
	TemplateID string
	Host       string
	Name       string
	Status     string
	Available  string
}

func init() {
	Log.SetHandler(logging.DiscardHandler())
}

func Version() string {
	return "0.0.3"
}

/**
 * Initialize history query
 */
func (s *Session) NewHistoryQuery() HistoryQuery {
	q := HistoryQuery{ValueType: 3, SortField: "clock", Output: "extend", SortOrder: "DESC", session: *s}
	return q
}

func (q *HistoryQuery) Query() []HistoryValue {
	response := historyQueryResponse{}
	req := Request{session: q.session, request: q, response: &response, method: "history.get"}
	err := req.query()
	if err != nil {
		Log.Error("failed to read history", "error", err)
		return nil
	}
	Log.Debug("loaded", logging.Ctx{"count": len(response.Items)})
	return response.Items
}

/**
 * Initialize trend query
 */
func (s *Session) NewTrendQuery(items []string, from time.Time, to time.Time) TrendQuery {
	q := TrendQuery{Items: items, Output: []string{"itemid", "clock", "num", "value_min", "value_avg", "value_max"}, session: *s}
	q.From = from.Unix()
	q.To = to.Unix()
	return q
}

func (q *TrendQuery) Query() []TrendValue {
	response := trendQueryResponse{}
	req := Request{session: q.session, request: q, response: &response, method: "trend.get"}
	err := req.query()
	if err != nil {
		Log.Error("failed to read trend", "error", err)
		return nil
	}
	Log.Debug("loaded", logging.Ctx{"count": len(response.Items)})
	return response.Items
}

func (s *Session) NewTemplateQuery(filter map[string][]string, search map[string][]string) TemplateQuery {
	q := TemplateQuery{Output: "extend", session: *s}
	q.Filter = filter
	q.Search = search
	if search != nil {
		q.SearchWildcardsEnabled = true
	}
	return q
}

func (q *TemplateQuery) Query() []TemplateResponseItem {
	response := templateQueryResponse{}
	req := Request{session: q.session, request: q, response: &response, method: "template.get"}
	err := req.query()
	if err != nil {
		Log.Error("failed to read templates", "error", err)
		return nil
	}
	Log.Debug("loaded", logging.Ctx{"count": len(response.Elements)})

	return response.Elements
}

func (s *Session) NewItemQuery(hostids []string, filter map[string][]string, search map[string][]string) ItemQuery {
	q := ItemQuery{Output: "extend", session: *s}
	//q.TemplateIDs = templateids
	q.HostIDs = hostids
	q.Filter = filter
	q.Search = search
	if search != nil {
		q.SearchWildcardsEnabled = true
	}
	q.SortField = []string{"hostid"}
	return q
}

func (q *ItemQuery) Query() []ItemResponseElement {
	response := itemQueryResponse{}
	req := Request{session: q.session, request: q, response: &response, method: "item.get"}
	err := req.query()
	if err != nil {
		Log.Error("failed to read templates", "error", err)
		return nil
	}
	Log.Debug("loaded", logging.Ctx{"count": len(response.Elements)})

	return response.Elements
}

func (s *Session) NewHostQuery(templateids []string, filter map[string][]string, search map[string][]string) HostQuery {
	q := HostQuery{Output: "extend", session: *s}
	q.TemplateIDs = templateids
	q.Filter = filter
	q.Search = search
	if search != nil {
		q.SearchWildcardsEnabled = true
	}
	// ignore templates by default
	q.IncludeTemplates = false
	q.IncludeMonitored = false
	q.SortField = []string{"hostid"}
	return q
}

func (q *HostQuery) Query() []HostResponseElement {
	response := hostQueryResponse{}
	req := Request{session: q.session, request: q, response: &response, method: "host.get"}
	err := req.query()
	if err != nil {
		Log.Error("failed to read templates", "error", err)
		return nil
	}
	Log.Debug("loaded", logging.Ctx{"count": len(response.Elements)})

	return response.Elements
}

func (query *Request) query() error {
	uri := query.session.URL
	request := request{Encoding: "2.0", Method: query.method, Params: query.request, Id: requestEnumerator}
	requestEnumerator++
	request.Auth = query.session.Token
	message, err := json.Marshal(request)
	if err != nil {
		return err
	}
	Log.Debug("zabbix api call", "url", uri, "json", string(message))
	start := time.Now()
	response, err := query.session.Connection.Post(uri, contentType, bytes.NewReader(message))
	end := time.Now()
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()

	if err != nil {
		return err
	}

	duration := end.Sub(start)
	Log.Debug("result from server", "ms", 1.0*float64(duration.Nanoseconds())/(1000*1000), "response", string(body[0:min(700, len(body)-1)]))

	err = json.Unmarshal(body, query.response)
	if err != nil {
		Log.Error("failed to parse json", "error", err)
		return err
	}

	Log.Debug("received", "result", query.response)

	return nil
}

/**
 *
	POST http://company.com/zabbix/api_jsonrpc.php HTTP/1.1
	Content-Type: application/json-rpc

{
    "jsonrpc": "2.0",
    "method": "user.login",
    "params": {
        "user": "Admin",
        "password": "zabbix"
    },
    "id": 1
 }
*/
func Login(settings *Session, user string, password string) error {
	//
	uri := settings.URL

	Log.Debug("reading server version", "uri", uri)
	response, err := settings.Connection.Post(uri, contentType, bytes.NewReader([]byte("{\"jsonrpc\":\"2.0\",\"method\":\"apiinfo.version\",\"id\":-1,\"auth\":null,\"params\":{}}")))
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	settings.ServerVersion = string(body)

	Log.Debug("successfully conneted", "response body", settings.ServerVersion, "HTTP response", response)

	auth := request{Encoding: "2.0", Method: "user.login", Params: auth{User: user, Password: password}, Id: requestEnumerator}
	requestEnumerator++
	message, err := json.Marshal(auth)
	if err != nil {
		return err
	}
	Log.Debug("authenticating with server", "username", user)
	response, err = settings.Connection.Post(uri, contentType, bytes.NewReader(message))
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("failed to authenticate. status code %d", response.StatusCode)
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	Log.Debug("received response from server", "response", string(body[0:min(700, len(body)-1)]))

	result := loginResponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	Log.Debug("received token", "token", result.Result)
	if len(result.Result) < 5 || result.Error.Code != 0 {
		return fmt.Errorf("failed to authenticate: %#v", result.Error)
	}

	settings.Token = result.Result

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

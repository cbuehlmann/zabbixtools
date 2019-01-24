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
	"net/http"
	"time"
)

var Log = logging.New()

const contentType string = "application/json-rpc"

type Session struct {
	URL string
	// reuse connection HTTP 1.1
	Connection http.Client
	// Authentication Token
	Token string
}

// Note: json.Marshal does only process fields with upper case name
type request struct {
	Encoding string      `json:"jsonrpc"` // "2.0"
	Method   string      `json:"method"`  // example: "user.login"
	Params   interface{} `json:"params"`
	Id       int         `json:"id"` // request id
	Auth     string      `json:"auth"`
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
	Id       string `json:"id"` // referencing request id
	// Error reporting
	Error apiError `json:"error"`
}

type Value struct {
	Value string `json:"value"`
	Item  string `json:"itemid"`
	Clock int64  `json:"clock,string"` // seconds since epoch
	Nano  int64  `json:"ns,string"`    // nanoseconds
}

type queryResponse struct {
	Encoding string  `json:"jsonrpc"` // "2.0"
	Items    []Value `json:"result"`
	Id       string  `json:"id"` // referencing request id
}

/**
* Refer to https://www.zabbix.com/documentation/4.0/manual/api/reference/history/get
 */
type HistoryQuery struct {
	History   int    `json:"history"`             // 0 - numeric float; 1 - character; 2 - log; 3 - numeric unsigned; 4 - text.
	Output    string `json:"output"`              // extend | count
	Hosts     string `json:"hostids,omitempty"`   // host ids, numeric
	Items     string `json:"itemids"`             // item ids, numeric
	From      int64  `json:"time_from,omitempty"` // timerange start. seconds since epoch
	To        int64  `json:"time_till,omitempty"` // timerange end. seconds since epoch
	SortField string `json:"sortfield"`           // clock|value|ns
	Limit     int    `json:"limit,omitempty"`     // limit number of records
	SortOrder string `json:"sortorder,omitempty"` // DESC|ASC
}

func init() {
	Log.SetHandler(logging.DiscardHandler())
}

func newRequest(method string, payload interface{}) request {
	return request{Encoding: "2.0", Method: method, Params: payload, Id: 1}
}

func Version() string {
	return "0.0.1"
}

/**
 * Initialize history query
 */
func NewHistoryQuery() HistoryQuery {
	return HistoryQuery{History: 3, SortField: "clock", Output: "extend", SortOrder: "DESC"}
}

func History(settings Session, query HistoryQuery) ([]Value, error) {
	return Query(settings, query, "history.get")
}

func Query(settings Session, query interface{}, api string) ([]Value, error) {
	uri := settings.URL
	request := newRequest(api, query)
	request.Auth = settings.Token
	message, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	Log.Debug("zabbix api call", "url", uri, "json", string(message))
	start := time.Now()
	response, err := settings.Connection.Post(settings.URL, contentType, bytes.NewReader(message))
	end := time.Now()
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()

	if err != nil {
		return nil, err
	}

	Log.Debug("result from server", "duration", (end.Nanosecond()-start.Nanosecond())/10000, "response", string(body[0:min(100, len(body)-1)]))

	result := queryResponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	Log.Debug("hits from server", "number", len(result.Items))

	return result.Items, nil
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
	auth := newRequest("user.login", auth{User: user, Password: password})
	message, err := json.Marshal(auth)
	if err != nil {
		return err
	}
	Log.Debug("connecting to", "url", uri)
	response, err := settings.Connection.Post(uri, contentType, bytes.NewReader(message))
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("failed to authenticate. status code %d", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	result := loginResponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	Log.Debug("received token", "json", string(body), "token", result.Result)
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

package gridas

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"labix.org/v2/mgo/bson"
)

var methods = []string{"GET", "POST", "PUT", "DELETE", "HEAD"}
var data = [][]byte{{1, 2, 3, 4, 0, 5, 6, 7}}
var urls = []string{"http://0.0.0.0:0"}
var targetHosts = []string{"0.0.0.0:0"}

func do(request *http.Request, listener *Listener) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	listener.ServeHTTP(response, request)
	return response
}

func doRequest(request *http.Request, t *testing.T) (*httptest.ResponseRecorder, *Listener) {
	channel := make(chan *Petition, 1000)
	listener := &Listener{SendTo: channel, Cfg: cfgTest, SessionSeed: sessionTest}
	response := do(request, listener)
	return response, listener
}
func doAny(listener *Listener, t *testing.T) *httptest.ResponseRecorder {
	var request, err = http.NewRequest("GET", urls[0], bytes.NewReader(data[0]))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set(RelayerHost, targetHosts[0])
	var response = do(request, listener)
	return response
}

func TestListenerMissingHeader(t *testing.T) {
	setUp(t)
	defer tearDown(t)
	for _, method := range methods {
		request, err := http.NewRequest(method, urls[0], bytes.NewReader(data[0]))
		if err != nil {
			t.Fatal(err)
		}
		response, listener := doRequest(request, t)
		if response.Code != 400 {
			t.Errorf("missing x-relayer-host should return 400 %d", response.Code)
		}
		if len(listener.SendTo) > 0 {
			t.Errorf("invalid request should not be enqueued, method %q len(listener.SendTo) %d",
				method, len(listener.SendTo))
		}
	}
}
func TestListenerBadProtocol(t *testing.T) {
	setUp(t)
	defer tearDown(t)
	for _, method := range methods {
		request, err := http.NewRequest(method, urls[0], bytes.NewReader(data[0]))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set(RelayerHost, "unknownveryfaraway")
		request.Header.Set(RelayerProtocol, "ftp")
		response, listener := doRequest(request, t)
		if response.Code != 400 {
			t.Errorf("unsupported x-relayer-protocol ftp should return 400 %d", response.Code)
		}
		if len(listener.SendTo) > 0 {
			t.Errorf("invalid request should not be enqueued, method %q len(listener.SendTo) %d",
				method, len(listener.SendTo))
		}
	}
}
func TestListener(t *testing.T) {
	setUp(t)
	defer tearDown(t)
	for _, method := range methods {
		var request, err = http.NewRequest(method, urls[0], bytes.NewReader(data[0]))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set(RelayerHost, targetHosts[0])
		var response, l = doRequest(request, t)
		if response.Code != 200 {
			t.Errorf("expected status code 200: %d", response.Code)
		}
		if len(l.SendTo) != 1 {
			t.Errorf("valid request should be enqueued, method %q len(l.SendTo) %d", method, len(l.SendTo))
		}
		respData := make(map[string]string)
		if err := json.Unmarshal(response.Body.Bytes(), &respData); err != nil {
			t.Fatal(err)
		}
		var id = respData["id"]
		var pet = Petition{}

		//Hoping no problem for non-strong mode
		db := sessionTest.DB(l.Cfg.Database)
		petColl := db.C(cfgTest.Instance + cfgTest.PetitionsColl)
		err = petColl.Find(bson.M{"id": id}).One(&pet)
		if err != nil {
			t.Fatalf("petition should be stored with returned ID, method %q id %q err %v", method, id, err)
		}

		if pet.Method != method {
			t.Errorf("petition's method should be equal to request's one, id %q method %q petition's method %q",
				id, method, pet.Method)
		}
		if !reflect.DeepEqual(pet.Body, data[0]) {
			t.Errorf("petition's body should be equal to request's one, id %q method %q", id, method)
		}
		if pet.TargetHost != targetHosts[0] {
			t.Logf("petition's target host should be equal to target host, id %q method %q petition's host %q target host %q",
				id, method, pet.TargetHost, targetHosts[0])
			t.Error(pet)
		}

	}
}
func TestListenerStop(t *testing.T) {
	setUp(t)
	defer tearDown(t)
	var listener = &Listener{SendTo: make(chan *Petition, 1000), Cfg: cfgTest, SessionSeed: sessionTest}
	listener.Stop()
	var response = doAny(listener, t)
	if response.Code != 503 {
		t.Errorf("Listener stopping should return 503 code, not %d", response.Code)
	}
}
func TestListenerFullQueue(t *testing.T) {
	setUp(t)
	defer tearDown(t)
	var listener = &Listener{
		SendTo: make(chan *Petition), //not buffered, would block with first petition, simulate full queue
		Cfg:    cfgTest, SessionSeed: sessionTest}
	var response = doAny(listener, t)
	if response.Code != 500 {
		t.Errorf("Listener with full queue should return 500 code, not %d", response.Code)
	}
}

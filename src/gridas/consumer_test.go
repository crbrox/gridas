// consumer_test.go
package gridas

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"labix.org/v2/mgo/bson"
)

const idPetition = "1"

var testBody = []byte("sent content to target host")
var testBodyResponse = []byte("Hello, client (from target host)\n")

func TestConsumer(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	reqChan := make(chan *Petition, cfg.QueueSize)
	resultCh := make(chan *http.Request, 1)
	errCh := make(chan error, 1)
	consumer := newTestConsumer(reqChan)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("received ", r)
		fmt.Fprint(w, string(testBodyResponse))
		rcvdBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errCh <- err
		}
		if !reflect.DeepEqual(rcvdBody, testBody) {
			errCh <- fmt.Errorf("received body in target host is not equal to sent body")
		}
		resultCh <- r
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/x/y/z"
	u.RawQuery = "q=a&r=b"
	petition := &Petition{
		ID:           idPetition,
		TargetHost:   u.Host,
		TargetScheme: u.Scheme,
		Method:       "GET",
		URL:          u,
		Proto:        "HTTP/1.1",
		Body:         testBody,
		RemoteAddr:   "127.0.0.1",
		Host:         u.Host,
		Created:      time.Now(),
	}
	endCh := consumer.Start(2)
	reqChan <- petition
	var rcvRequest *http.Request
	select {
	case rcvRequest = <-resultCh:
	case e := <-errCh:
		t.Fatal(e)
	case <-time.After(5 * time.Second):
		t.Fatal("target server waiting too long")
	}
	if rcvRequest.URL.Path != petition.URL.Path {
		t.Errorf("received url path is not equal to sent url path %q %q", rcvRequest.URL, petition.URL)
	}
	if rcvRequest.URL.RawQuery != petition.URL.RawQuery {
		t.Errorf("received query is not equal to sent query %q %q", rcvRequest.URL, petition.URL)
	}
	if rcvRequest.Method != petition.Method {
		t.Errorf("received method is not equal to sent method %q %q", rcvRequest.URL, petition.URL)
	}

	consumer.Stop()
	select {
	case <-endCh:
	case <-time.After(time.Second):
		t.Fatal("time out stopping")
	}

	reply := Reply{}
	err = responseStoreTest.Find(bson.M{"id": idPetition}).One(&reply)
	if err != nil {
		t.Fatalf("reply should be stored %v", err)
	}
	if !reflect.DeepEqual(reply.Body, testBodyResponse) {
		t.Errorf("target response body does not match %v %v", reply.Body, testBodyResponse)
	}
	err = petitionStoreTest.Find(bson.M{"id": idPetition}).One(&Petition{})
	if err == nil {
		t.Errorf("petition should be deleted %q", idPetition)
	}
}
func TestConsumerErrorResponse(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	reqChan := make(chan *Petition, cfg.QueueSize)
	resultCh := make(chan *http.Request, 1)
	consumer := newTestConsumer(reqChan)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("received ", r)
		w.WriteHeader(http.StatusServiceUnavailable)
		resultCh <- r
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/x/y/z"
	u.RawQuery = "q=a&r=b"
	petition := &Petition{
		ID:           idPetition,
		TargetHost:   u.Host,
		TargetScheme: u.Scheme,
		Method:       "GET",
		URL:          u,
		Proto:        "HTTP/1.1",
		RemoteAddr:   "127.0.0.1",
		Host:         u.Host,
		Created:      time.Now(),
	}
	endCh := consumer.Start(2)
	reqChan <- petition

	select {
	case <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("target server waiting too long")
	}

	consumer.Stop()
	select {
	case <-endCh:
	case <-time.After(time.Second):
		t.Fatal("time out stopping")
	}

	reply := Reply{}
	err = responseStoreTest.Find(bson.M{"id": idPetition}).One(&reply)
	if err != nil {
		t.Fatalf("reply should be stored in petition store %v", err)
	}
	if reply.StatusCode != http.StatusServiceUnavailable {
		t.Error("reply status code distinct to response status code %d != %d", reply.StatusCode, http.StatusServiceUnavailable)
	}
	err = errorStoreTest.Find(bson.M{"id": idPetition}).One(&reply)
	if err != nil {
		t.Fatalf("reply should be stored in errors collections %v", err)
	}
	if reply.StatusCode != http.StatusServiceUnavailable {
		t.Error("reply status code distinct to response status code %d != %d", reply.StatusCode, http.StatusServiceUnavailable)
	}
	err = petitionStoreTest.Find(bson.M{"id": idPetition}).One(&Petition{})
	if err == nil {
		t.Errorf("petition should be deleted %q", idPetition)
	}
}

func newTestConsumer(petCh chan *Petition) *Consumer {
	return &Consumer{GetFrom: petCh,
		PetitionStore: petitionStoreTest,
		ReplyStore:    responseStoreTest,
		ErrorStore:    errorStoreTest,
	}
}

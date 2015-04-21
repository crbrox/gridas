package gridas

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"labix.org/v2/mgo/bson"
)

func TestReplyer(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	var obj = Reply{
		ID:         "1234_abcxxxxxxxxxxxx",
		Error:      "Errare humanum est",
		StatusCode: 102,
		Header:     make(http.Header), // So it is retrieved from mongoDB. nil does not work
		Trailer:    make(http.Header), // So it is retrieved from mongoDB. nil does not work
		Proto:      "protocol",
		Body:       []byte{1, 2, 3, 0, 9, 8, 7},
		Done:       bson.Now(),
		Created:    bson.Now(),
	}

	replyer := &Replyer{Cfg: cfgTest, SessionSeed: sessionTest}
	db := sessionTest.DB(cfgTest.Database)
	respColl := db.C(cfgTest.ResponsesColl)
	respColl.Insert(obj)
	if err := sessionTest.Fsync(false); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest("GET", "/one/two/three/"+obj.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	replyer.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Errorf("response should be 200 for existent reply. reponse.Code %d", response.Code)
	}

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("content type of response should be \"application/json\". It is %q", contentType)
	}
	var objR = Reply{}
	err = json.Unmarshal(response.Body.Bytes(), &objR)
	if err != nil {
		t.Fatalf("%v : %q", err, response.Body.Bytes())
	}
	if !reflect.DeepEqual(obj, objR) {
		t.Error("response body should be equal to body reply %#v %#v", obj, objR)
	}

}
func TestReplyerNotFound(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	const id = "1234_abc"
	replyer := &Replyer{Cfg: cfgTest, SessionSeed: sessionTest}
	request, err := http.NewRequest("GET", "/one/two/three/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	replyer.ServeHTTP(response, request)
	if response.Code != 404 {
		t.Error("response should be 404 for inexistent reply")
	}
}

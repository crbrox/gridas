// recoverer_test.go
package gridas

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

func TestRecoverer(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	session := sessionTest.New()
	session.SetMode(mgo.Strong, true)
	defer session.Close()

	url1, _ := url.Parse("http://golang.org/pkg/net/http/#NewRequest")
	url2, _ := url.Parse("https://www.google.es")
	p1 := &Petition{
		ID:           "id1",
		Body:         []byte("body1"),
		Method:       "GET",
		URL:          url1,
		Proto:        "HTTP/1.1",
		RemoteAddr:   "RemoteAddr1",
		RequestURI:   "RequestURI1",
		TargetHost:   "targetHost1",
		TargetScheme: "http",
		Header:       make(http.Header), // So it is retrieved from mongoDB. nil does not work
		Trailer:      make(http.Header), // So it is retrieved from mongoDB. nil does not work
		Created:      bson.Now()}
	p2 := &Petition{
		ID:           "id2",
		Body:         []byte("body2"),
		Method:       "POST",
		URL:          url2,
		Proto:        "HTTP/1.1",
		RemoteAddr:   "RemoteAddr2",
		RequestURI:   "RequestURI2",
		TargetHost:   "targetHost2",
		TargetScheme: "https",
		Header:       make(http.Header), // So it is retrieved from mongoDB. nil does not work
		Trailer:      make(http.Header), // So it is retrieved from mongoDB. nil does not work
		Created:      bson.Now()}
	var petitions = []*Petition{p1, p2}
	petCh := make(chan *Petition, 1000)
	for _, p := range petitions {
		db := session.DB(cfgTest.Database)
		petColl := db.C(cfgTest.Instance + cfgTest.PetitionsColl)
		e := petColl.Insert(p)
		if e != nil {
			t.Fatal(e)
		}
	}
	recoverer := &Recoverer{SendTo: petCh, SessionSeed: session, Cfg: cfgTest}
	err := recoverer.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(petCh) != len(petitions) {
		t.Fatalf("number of stored petitions should be equal to enqueued ones len(petCh) %d len(petitions) %d ",
			len(petCh), len(petitions))
	}
	for i := 0; i < len(petitions); i++ {
		select {
		case ep := <-petCh:
			var sp *Petition
			switch ep.ID {
			case "id1":
				sp = p1
			case "id2":
				sp = p2
			default:
				t.Fatalf("unknown enqueued petition id %q", ep.ID)
			}
			if !reflect.DeepEqual(ep, sp) {
				t.Fatalf("enqued petition is not equal to stored petition %#v %#v", ep, sp)
			}
		default:
			t.Fatal("should be more enqueued petitions")
		}
	}
}

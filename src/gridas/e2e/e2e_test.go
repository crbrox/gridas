package e2e

import (
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"gridas"
)

func init() {
	flag.StringVar(&benchURL, "url", "http://localhost:8080/potoclon", "URL to test against")
	flag.Parse()
}

var benchURL = "http://localhost:8080/pirri"

var OK = []byte{'O', 'K'}
var client = http.Client{}

func get(url string, t *testing.T) {
	var request, err = http.NewRequest("GET", benchURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set(gridas.RelayerHost, url)
	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

}
func TestE2EGetUnknownHost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(OK)
	}))
	url := "askdjasjdlasjdlasjdsa"
	defer ts.Close()
	get(url, t)
}
func TestE2EGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(OK)
	}))
	url := ts.URL[len("http://"):]
	defer ts.Close()
	get(url, t)
}
func TestE2ERoundGet(t *testing.T) {
	gotCh := make(chan bool, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCh <- true
		w.Write(OK)
	}))
	url := ts.URL[len("http://"):]
	defer ts.Close()
	get(url, t)
	<-gotCh
}

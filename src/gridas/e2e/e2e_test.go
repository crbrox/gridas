package e2e

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"gridas"
)

const benchURL = "http://localhost:8080/pirri"

var OK = []byte{'O', 'K'}
var client = http.Client{}

func get(url string, b *testing.B) {
	var request, err = http.NewRequest("GET", benchURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	request.Header.Set(gridas.RelayerHost, url)
	b.StartTimer()
	resp, err := client.Do(request)
	if err != nil {
		b.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		b.Fatal(err)
	}

}

func BenchmarkE2EGetUnknownHost(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(OK)
	}))
	url := "askdjasjdlasjdlasjdsa"
	defer ts.Close()
	b.ResetTimer()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		get(url, b)
		b.StopTimer()
	}
}
func BenchmarkE2EGet(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(OK)
	}))
	url := ts.URL[len("http://"):]
	defer ts.Close()
	b.ResetTimer()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		get(url, b)
		b.StopTimer()
	}
}
func BenchmarkE2ERoundGet(b *testing.B) {
	gotCh := make(chan bool, b.N)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCh <- true
		w.Write(OK)
	}))
	url := ts.URL[len("http://"):]
	defer ts.Close()
	b.ResetTimer()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		get(url, b)
		<-gotCh
		b.StopTimer()
	}
}

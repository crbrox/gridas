package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
)

const benchURL = "http://localhost:8080/pirri"

var OK = []byte{'O', 'K'}
var client = http.Client{}

func main() {
	BenchmarkE2ERoundGet()
}
func BenchmarkE2ERoundGet() {
	log.Println("begin")
	gotCh := make(chan bool, 1000)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCh <- true
		w.Write(OK)
		log.Println("peticion recibida")
	}))
	defer ts.Close()
	log.Println("Preparando request")
	var request, err = http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		panic(err)
	}
	request.Header.Set("x-relayer-host", "churimiri")
	log.Println("A punto de realizar request", request)
	resp, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		panic("response status code != 200")
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	<-gotCh
	//b.StopTimer()

}

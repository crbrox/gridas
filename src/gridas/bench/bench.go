package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

const benchURL = "http://localhost:8080/pirri"

var OK = []byte{'O', 'K'}
var client = http.Client{}

const N = 100
const M = 200

var wg sync.WaitGroup

func main() {
	wg.Add(N)
	for i := 0; i < N; i++ {
		go BenchmarkE2ERoundGet()
	}
	wg.Wait()
}
func BenchmarkE2ERoundGet() {
	defer wg.Done()
	defer func() {
		fmt.Println("*")
	}()
	gotCh := make(chan bool, 1000)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCh <- true
		w.Write(OK)
	}))
	defer ts.Close()
	for j := 0; j < M; j++ {
		start := time.Now()
		var request, err = http.NewRequest("GET", benchURL, nil)
		if err != nil {
			fmt.Println(err)
		}
		request.Header.Set("x-relayer-host", ts.URL[len("http://"):])
		resp, err := client.Do(request)
		if err != nil {
			fmt.Println(err)
			return
		}
		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			fmt.Println("response status code != 200")
			return
		}
		defer resp.Body.Close()
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		<-gotCh
		fmt.Println(time.Since(start).Nanoseconds())
	}

}

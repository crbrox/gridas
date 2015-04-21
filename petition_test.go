// petition_test.go
package gridas

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

// TODO: Refactor!! DRY && use gocheck && ...

var body = []byte{1, 2, 3, 0, 255, 254}

const targetHost = "veryfarawayhost:1234"
const targetScheme = "https"
const rushURL = "http://rushhost:9876/path/subpath?q=a&r=b&r=c&españa=olé"

//We want to try lowercase. Don`t use package constants!!
const (
	relayerHostField    = "x-relayer-host"
	relayerSchemeField  = "x-relayer-protocol"
	relayerTraceidField = "x-relayer-traceid"
	relayerTopicField   = "x-relayer-topic"
	relayerRetryField   = "x-relayer-retry"
	relayerProxyField   = "x-relayer-proxy"
)

func TestPetition(t *testing.T) {
	for _, traceHdrToTry := range []string{relayerTraceidField, relayerTopicField} {
		for _, traceid := range []string{"", "un identificador algo largo pero interesante", "*", "españa y olé"} {

			original, err := http.NewRequest("GET", rushURL, bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			original.Header.Set(relayerHostField, targetHost)
			original.Header.Set(relayerSchemeField, targetScheme)
			original.Header.Set(traceHdrToTry, traceid)
			original.Header.Set("x-another-thing", "verywell")
			original.Header.Add("x-another-thing", "fandango")
			petition, err := newPetition(original)
			t.Logf("%#v\n", petition)
			if err != nil {
				t.Fatal(err)
			}
			if petition.TraceID != traceid {
				t.Errorf("traceID %q should be equal to original %q", petition.TraceID, traceid)
			}
			retrieved, err := petition.Request()
			if err != nil {
				t.Fatal(err)
			}
			retrievedBody, err := ioutil.ReadAll(retrieved.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, retrievedBody) {
				t.Error("retrieved body should be equal to sent body")
			}
			for _, hf := range []string{relayerHostField, relayerSchemeField, traceHdrToTry} {
				if value, ok := retrieved.Header[http.CanonicalHeaderKey(hf)]; ok {
					t.Errorf("retrieved header %q should have been erased, value %q", hf, value)
				}
			}

			if !reflect.DeepEqual(retrieved.Header, original.Header) {
				t.Error("retrieved header should be equal to sent header")
			}
			if retrieved.URL.Host != targetHost {
				t.Errorf("retrieved host %q should be equal to target host %q", retrieved.URL.Host, targetHost)
			}
			if retrieved.URL.Scheme != targetScheme {
				t.Errorf("retrieved scheme %q should be equal to target scheme %q", retrieved.URL.Scheme, targetScheme)
			}

			if retrieved.URL.RequestURI() != original.URL.RequestURI() {
				t.Errorf("retrieved requestURI %q should be equal to original requestURI %q",
					retrieved.URL.RequestURI(), original.URL.RequestURI())
			}

		}
	}
}
func TestPetitionOlderHost(t *testing.T) {
	original, err := http.NewRequest("GET", rushURL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []url.URL{{Scheme: "http", Host: "localhost:9812"}, {Scheme: "https", Host: "another"}} {
		str := u.String()
		t.Log("trying older x-relayer-host", str)
		original.Header.Set(relayerHostField, str)

		original.Header.Set("x-another-thing", "blondie")
		original.Header.Add("x-another-thing", "onewayoranother")
		petition, err := newPetition(original)
		t.Logf("%#v\n", petition)
		if err != nil {
			t.Fatal(err)
		}
		retrieved, err := petition.Request()
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(retrieved.Header, original.Header) {
			t.Error("retrieved header should be equal to sent header")
		}
		if retrieved.URL.Host != u.Host {
			t.Errorf("retrieved host %q should be equal to target host %q", retrieved.URL.Host, u.Host)
		}
		if retrieved.URL.Scheme != u.Scheme {
			t.Errorf("retrieved scheme %q should be equal to target scheme %q", retrieved.URL.Scheme, u.Scheme)
		}

		if retrieved.URL.RequestURI() != original.URL.RequestURI() {
			t.Errorf("retrieved requestURI %q should be equal to original requestURI %q",
				retrieved.URL.RequestURI(), original.URL.RequestURI())
		}
	}
}
func TestPetitionRemoveOlder(t *testing.T) {
	for _, hf := range []string{RelayerProxy, RelayerTopic, RelayerRetry} {
		original, err := http.NewRequest("GET", rushURL, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}

		original.Header.Set(relayerHostField, targetHost)
		original.Header.Set(hf, "some value")
		original.Header.Set("x-another-thing", "blondie")
		original.Header.Set("x-another-thing", "blondie")
		original.Header.Add("x-another-thing", "onewayoranother")
		petition, err := newPetition(original)
		t.Logf("petition %#v\n", petition)
		if err != nil {
			t.Fatal(err)
		}
		retrieved, err := petition.Request()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("retrieved %#v\n", retrieved)
		t.Logf("retrieved  URL %#v\n", retrieved.URL)

		if !reflect.DeepEqual(retrieved.Header, original.Header) {
			t.Error("retrieved header should be equal to sent header")
		}
		if retrieved.URL.Host != targetHost {
			t.Errorf("retrieved host %q should be equal to target host %q", retrieved.URL.Host, targetHost)
		}
		if retrieved.URL.Scheme != "http" {
			t.Errorf("retrieved scheme %q should be equal to target scheme %q", retrieved.URL.Scheme, "http")
		}

		if retrieved.URL.RequestURI() != original.URL.RequestURI() {
			t.Errorf("retrieved requestURI %q should be equal to original requestURI %q",
				retrieved.URL.RequestURI(), original.URL.RequestURI())
		}
		if value, ok := retrieved.Header[http.CanonicalHeaderKey(hf)]; ok {
			t.Errorf("retrieved header %q should have been erased, value %q", hf, value)
		}
	}
}

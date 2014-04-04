// petition_test.go
package gridas

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
)

var body = []byte{1, 2, 3, 0, 255, 254}

const targetHost = "veryfarawayhost:1234"
const targetScheme = "https"
const rushURL = "http://rushhost:9876/path/subpath?q=a&r=b&r=c&españa=olé"

//We want to try lowercase. Don`t use package constants!!
const (
	relayerHostField    = "x-relayer-host"
	relayerSchemeField  = "x-relayer-protocol"
	relayerTraceidField = "x-relayer-traceid"
)

func TestPetition(t *testing.T) {
	for _, traceid := range []string{"", "un identificador algo largo pero interesante", "*", "españa y olé"} {

		original, err := http.NewRequest("GET", rushURL, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		original.Header.Set(relayerHostField, targetHost)
		original.Header.Set(relayerSchemeField, targetScheme)
		original.Header.Set(relayerTraceidField, traceid)
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
		//This is not necessary probably. Just in case implementation changes
		original.Header.Del(relayerHostField)
		original.Header.Del(relayerSchemeField)
		original.Header.Del(relayerTraceidField)

		if !reflect.DeepEqual(retrieved.Header, original.Header) {
			t.Error("retrieved header should be equal to sent header")
		}
		if retrieved.URL.Host != targetHost {
			t.Errorf("retrieved host %q should be equal to target host %q", retrieved.URL.Host, targetHost)
		}
		if retrieved.URL.Scheme != targetScheme {
			t.Errorf("retrieved scheme %q should be equal to target scheme %q", retrieved.URL.Host, targetHost)
		}

		if retrieved.URL.RequestURI() != original.URL.RequestURI() {
			t.Errorf("retrieved requestURI %q should be equal to original requestURI %q",
				retrieved.URL.RequestURI(), original.URL.RequestURI())
		}

	}

}

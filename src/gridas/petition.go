package gridas

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"labix.org/v2/mgo/bson"

	"gridas/mylog"
)

//Petition is a representation from the request received. Header fields are cooked to represent
//the final request meant to be sent to the target host. The relayer's own fields are removed
type Petition struct {
	ID           string      `json:"id"`
	TraceID      string      `json:"traceid"`
	TargetHost   string      `json:"targethost"`
	TargetScheme string      `json:"targetscheme"`
	Method       string      `json:"method"` // GET, POST, PUT, etc.
	URL          *url.URL    `json:"-"`
	URLString    string      `json:"urlstring"`
	Proto        string      `json:"proto"` // "HTTP/1.0"
	Header       http.Header `json:"header"`
	Trailer      http.Header `json:"trailer"`
	Body         []byte      `json:"body"`
	RemoteAddr   string      `json:"remoteaddr"`
	RequestURI   string      `json:"requesturi"`
	Host         string      `json:"host"`
	Created      time.Time   `json:"created"`
}

//newPetition creates a petition from an http.Request. It checks header fields and make necessary transformations.
//The body is read and saved as a slice of byte.
func newPetition(original *http.Request) (*Petition, error) {
	targetHost := original.Header.Get(RelayerHost)
	if targetHost == "" {
		return nil, fmt.Errorf("gridas: Missing mandatory header %s", RelayerHost)
	}
	original.Header.Del(RelayerHost)
	scheme := strings.ToLower(original.Header.Get(RelayerProtocol))
	switch scheme {
	case "http", "https":
	case "":
		scheme = "http"
	default:
		mylog.Debug("unsupported protocol", scheme)
		return nil, fmt.Errorf("gridas: unsupported protocol %s", scheme)

	}
	original.Header.Del(RelayerProtocol)
	traceID := original.Header.Get(RelayerTraceID)
	original.Header.Del(RelayerTraceID)
	//save body content
	body, err := ioutil.ReadAll(original.Body)
	if err != nil {
		mylog.Debugf("error reading body request %v %+v", err, original)
		return nil, err
	}
	id := uuid.New()
	relayedRequest := &Petition{
		ID:           id,
		Body:         body,
		Method:       original.Method,
		URL:          original.URL,
		Proto:        original.Proto, // "HTTP/1.0"
		Header:       original.Header,
		Trailer:      original.Trailer,
		RemoteAddr:   original.RemoteAddr,
		RequestURI:   original.RequestURI,
		TargetHost:   targetHost,
		TargetScheme: scheme,
		Created:      bson.Now(),
		TraceID:      traceID}
	return relayedRequest, nil
}

//Request returns the original http.Request with the body restored as a CloserReader
//so it can be used to do a request to the original target host
func (p *Petition) Request() (*http.Request, error) {
	p.URL.Host = p.TargetHost
	p.URL.Scheme = p.TargetScheme
	p.URLString = p.URL.String()
	req, err := http.NewRequest(
		p.Method,
		p.URLString,
		ioutil.NopCloser(bytes.NewReader(p.Body))) //Restore body as ReadCloser
	if err != nil {
		mylog.Debugf("error restoring request %v %+v", err, p)
		return nil, err
	}
	req.Header = p.Header
	req.Trailer = p.Trailer

	return req, nil
}

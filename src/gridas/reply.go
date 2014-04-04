package gridas

import (
	"io/ioutil"
	"net/http"
	"time"

	"labix.org/v2/mgo/bson"
)

//Reply represents the response from the target host
type Reply struct {
	//Reply id. Currently the same as the petition id
	ID      string `json:"id"`
	TraceID string `json:"traceid"`
	//Possible error in making the request. Could be ""
	Error      string      `json:"error"`
	StatusCode int         `json:"statuscode"` // e.g. 200
	Proto      string      `json:"proto"`      // e.g. "HTTP/1.0"
	Header     http.Header `json:"header"`
	Trailer    http.Header `json:"trailer"`
	Body       []byte      `json:"body"`
	//Petition that
	Petition *Petition `json:"petition"`
	//Beginning of the request
	Created time.Time `json:"created"`
	//Time when response was received
	Done time.Time `json:"done"`
}

//newReply returns the Reply for the Petition made, the http.Response gotten and the possible error
func newReply(resp *http.Response, p *Petition, e error) *Reply {
	var reply = &Reply{ID: p.ID, Petition: p, TraceID: p.TraceID}
	if e != nil {
		reply.Error = e.Error()
		return reply
	}
	reply.StatusCode = resp.StatusCode
	reply.Proto = resp.Proto
	reply.Header = resp.Header
	reply.Trailer = resp.Trailer
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		reply.Error = e.Error()
	} else {
		reply.Body = body
	}
	reply.Done = bson.Now()
	return reply
}

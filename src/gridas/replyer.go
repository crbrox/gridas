package gridas

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

//Replyer provides the replies from the destination hosts through HTTP as a JSON document.
//The response body is encoded in base64
type Replyer struct {
	ReplyStore *mgo.Collection
}

func (r *Replyer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	base := filepath.Base(req.URL.Path)
	rpl := &Reply{}
	e := r.ReplyStore.Find(bson.M{"id": base}).One(&rpl)
	if e != nil {
		http.Error(w, e.Error(), http.StatusNotFound)
		return
	}
	text, e := json.MarshalIndent(rpl, "", " ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(text)
}

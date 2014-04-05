package gridas

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"gridas/mylog"
)

//Replyer provides the replies from the destination hosts through HTTP as a JSON document.
//The response body is encoded in base64
type Replyer struct {
	ReplyStore *mgo.Collection
}

func (r *Replyer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	base := filepath.Base(req.URL.Path)
	mylog.Debug("base ", base)
	rpl := &Reply{}
	e := r.ReplyStore.Find(bson.M{"id": base}).One(&rpl)
	if e != nil {
		mylog.Debug("reply not found ", base)
		http.Error(w, e.Error(), http.StatusNotFound)
		return
	}
	text, e := json.MarshalIndent(rpl, "", " ")
	if e != nil {
		mylog.Debug("error marshaling JSON ", base, e)
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	mylog.Debugf("returning JSON %q for %v", text, base)
	w.Header().Set("Content-Type", "application/json")
	w.Write(text)
}

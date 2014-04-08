package gridas

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"gridas/config"
	"gridas/mylog"
)

//Replyer provides the replies from the destination hosts through HTTP as a JSON document.
//The response body is encoded in base64
type Replyer struct {
	Cfg         *config.Config
	SessionSeed *mgo.Session
}

func (r *Replyer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	base := filepath.Base(req.URL.Path)
	mylog.Debug("request for response ", base)
	session := r.SessionSeed.New()
	defer session.Close()
	session.SetMode(mgo.Eventual, true)
	db := session.DB(r.Cfg.Database)
	respColl := db.C(r.Cfg.ResponsesColl)
	rpl := &Reply{}
	mylog.Debug("searching response", base)
	e := respColl.Find(bson.M{"id": base}).One(&rpl)
	if e != nil {
		mylog.Debug("reply not found ", base)
		http.Error(w, e.Error(), http.StatusNotFound)
		return
	}
	mylog.Debug("json marshaling response", base)
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

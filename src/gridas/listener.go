package gridas

import (
	"fmt"
	"net/http"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"gridas/config"
	"gridas/mylog"
)

//Listener is responsible for receiving requests and storing them in PetitionStore.
//It then passes a reference to the object Petition which wraps the original HTTP request through the channel Sendto,
//where the Consumer should collected it for further processing
type Listener struct {
	//Channel for sending petitions
	SendTo chan<- *Petition
	//Configuration object
	Cfg *config.Config
	//Flag signaling listener should finish
	stopping bool
	//Session seed for mongo
	SessionSeed *mgo.Session
}

//ServeHTTP implements HTTP handler interface
func (l *Listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mylog.Debugf("received request %+v", r)
	if l.stopping {
		mylog.Debug("warning client server is stopping")
		http.Error(w, "Server is shutting down", 503)
		return
	}
	relayedRequest, e := newPetition(r)
	if e != nil {
		mylog.Debug("petition with error", e)
		http.Error(w, e.Error(), 400)
		return
	}
	db := l.SessionSeed.DB(l.Cfg.Database)
	petColl := db.C(l.Cfg.Instance + l.Cfg.PetitionsColl)
	mylog.Debugf("petition created %+v", relayedRequest)
	e = petColl.Insert(relayedRequest)
	if e != nil {
		http.Error(w, relayedRequest.ID, 500)
		mylog.Alert("ERROR inserting", relayedRequest.ID, e)
		l.SessionSeed.Refresh()
		return
	}
	select {
	case l.SendTo <- relayedRequest:
		mylog.Debug("enqueued petition", relayedRequest)
		fmt.Fprintln(w, relayedRequest.ID)
	default:
		mylog.Alert("server is busy")
		http.Error(w, "Server is busy", 500)
		mylog.Debugf("before remove petition", relayedRequest.ID)
		err := petColl.Remove(bson.M{"id": relayedRequest.ID})
		mylog.Debugf("after remove petition", relayedRequest.ID)
		if err != nil {
			mylog.Alert("ERROR removing petition", relayedRequest.ID, e)
			l.SessionSeed.Refresh()
			return
		}
		return
	}
}

//Stop asks listener to stop receiving petitions
func (l *Listener) Stop() {
	mylog.Debug("listener received stop")
	l.stopping = true //Risky??
}

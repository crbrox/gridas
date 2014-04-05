package gridas

import (
	"fmt"
	"net/http"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"gridas/mylog"
)

//Listener is responsible for receiving requests and storing them in PetitionStore.
//It then passes a reference to the object Petition which wraps the original HTTP request through the channel Sendto,
//where the Consumer should collected it for further processing
type Listener struct {
	//Channel for sending petitions
	SendTo chan<- *Petition
	//Store for saving petitions in case of crash
	PetitionStore *mgo.Collection
	//Flag signaling listener should finish
	stopping bool
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
	mylog.Debugf("petition created %+v", relayedRequest)
	e = l.PetitionStore.Insert(relayedRequest)
	if e != nil {
		http.Error(w, relayedRequest.ID, 500)
		mylog.Alert("ERROR inserting", relayedRequest.ID, e)
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
		err := l.PetitionStore.Remove(bson.M{"id": relayedRequest.ID})
		mylog.Debugf("after remove petition", relayedRequest.ID)
		if err != nil {
			mylog.Alert("ERROR removing petition", relayedRequest.ID, e)
		}
		return
	}
}

//Stop asks listener to stop receiving petitions
func (l *Listener) Stop() {
	mylog.Debug("listener received stop")
	l.stopping = true //Risky??
}

package gridas

import (
	"fmt"
	"log"
	"net/http"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
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
	log.Println("received request", r)
	if l.stopping {
		http.Error(w, "Server is shutting down", 503)
		return
	}
	relayedRequest, e := newPetition(r)
	if e != nil {
		http.Error(w, e.Error(), 400)
		return
	}
	e = l.PetitionStore.Insert(relayedRequest)
	if e != nil {
		http.Error(w, relayedRequest.ID, 500)
		log.Println(relayedRequest.ID, e.Error())
		return
	}
	select {
	case l.SendTo <- relayedRequest:
		log.Println("enqueued petition", relayedRequest)
		fmt.Fprintln(w, relayedRequest.ID)
	default:
		log.Println("server is busy", relayedRequest)
		http.Error(w, "Server is busy", 500)
		l.PetitionStore.Remove(bson.M{"id": relayedRequest.ID})
		return
	}
}

//Stop asks listener to stop receiving petitions
func (l *Listener) Stop() {
	l.stopping = true //Risky??
}

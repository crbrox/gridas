package gridas

import (
	"labix.org/v2/mgo"
)

//Recoverer takes the petitions stored in PetitionStore and enqueues them again into SendTo.
type Recoverer struct {
	SendTo        chan<- *Petition
	PetitionStore *mgo.Collection
}

//Recover gets all the petitions stored and sends them to a channel for processing by a consumer.
//It returns when all of them are re-enqueued or when an error happens. It should be run before starting
//a listener (with the same PetitionStore) or new petitions could be enqueued twice. Listeners with a different PetitionStore
//should not be a problem. A Consumer can be started before with the same PetitionStore to avoid overflowing the queue.
func (r *Recoverer) Recover() error {
	p := Petition{}
	iter := r.PetitionStore.Find(nil).Iter()
	for iter.Next(&p) {
		paux := Petition{}
		paux = p
		r.SendTo <- &paux
	}
	//iter.Err()
	if err := iter.Close(); err != nil {
		return err
	}
	return nil
}

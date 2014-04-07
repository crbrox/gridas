package gridas

import (
	"labix.org/v2/mgo"

	"gridas/config"
	"gridas/mylog"
)

//Recoverer takes the petitions stored in PetitionStore and enqueues them again into SendTo.
type Recoverer struct {
	SendTo chan<- *Petition
	//Configuration object
	Cfg *config.Config
	//Session seed for mongo
	SessionSeed *mgo.Session
}

//Recover gets all the petitions stored and sends them to a channel for processing by a consumer.
//It returns when all of them are re-enqueued or when an error happens. It should be run before starting
//a listener (with the same PetitionStore) or new petitions could be enqueued twice. Listeners with a different PetitionStore
//should not be a problem. A Consumer can be started before with the same PetitionStore to avoid overflowing the queue.
func (r *Recoverer) Recover() error {
	mylog.Debug("begin recoverer")
	db := r.SessionSeed.DB(r.Cfg.Database)
	petColl := db.C(r.Cfg.Instance + r.Cfg.PetitionsColl)
	p := Petition{}
	iter := petColl.Find(nil).Iter()
	for iter.Next(&p) {
		paux := p
		mylog.Debugf("re-enqueue petition %+v", paux)
		r.SendTo <- &paux
	}
	//iter.Err()
	if err := iter.Close(); err != nil {
		mylog.Alertf("error closing cursor %+v", err)
		return err
	}
	mylog.Debug("end recoverer")
	return nil
}

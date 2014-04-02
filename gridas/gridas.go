// Executable of grush
//
// grush.ini contains configuration data read when starting at run-time
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"gridas"
	"gridas/config"

	"labix.org/v2/mgo"
)

func main() {

	log.Println("Hello World!")
	cfg, err := config.ReadConfig("gridas.json")
	if err != nil {
		log.Fatalln("-", err)
	}

	reqChan := make(chan *gridas.Petition, cfg.QueueSize)
	session, err := mgo.Dial(cfg.Mongo)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	db := session.DB(cfg.Database)

	l := &gridas.Listener{SendTo: reqChan, PetitionStore: db.C(cfg.Instance + cfg.PetitionsColl)}
	c := &gridas.Consumer{GetFrom: reqChan,
		PetitionStore: db.C(cfg.Instance + cfg.PetitionsColl),
		ReplyStore:    db.C(cfg.ResponsesColl),
		ErrorStore:    db.C(cfg.ErrorsColl),
	}
	r := &gridas.Replyer{ReplyStore: db.C(cfg.ResponsesColl)}
	rcvr := &gridas.Recoverer{SendTo: reqChan, PetitionStore: db.C(cfg.Instance + cfg.PetitionsColl)}

	endConsumers := c.Start(cfg.Consumers)
	if err := rcvr.Recover(); err != nil {
		log.Fatalln("-", err)
	}
	http.Handle("/", l)
	http.Handle("/responses/", http.StripPrefix("/responses/", r))
	go func() {
		log.Fatalln("-", http.ListenAndServe(":"+cfg.Port, nil))
	}()
	onEnd(func() {
		log.Println("Shutting down gridas ...")
		l.Stop()
		c.Stop()
	})
	<-endConsumers
	log.Println("Bye World!")
}

func onEnd(f func()) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		f()
	}()
}

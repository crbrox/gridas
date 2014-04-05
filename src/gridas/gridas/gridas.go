// Executable of grush
//
// grush.ini contains configuration data read when starting at run-time
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gridas"
	"gridas/config"
	"gridas/mylog"

	"labix.org/v2/mgo"
)

func main() {
	cfg, err := config.ReadConfig("gridas.yaml")
	if err != nil {
		mylog.Alert(err)
		os.Exit(-1)
	}
	fmt.Printf("%+v\n", cfg)

	//mgo.SetLogger(mylog.Logger())
	//mgo.SetDebug(false)
	mylog.SetLevel(cfg.LogLevel)
	mylog.Alert("hello World!")
	reqChan := make(chan *gridas.Petition, cfg.QueueSize)
	session, err := mgo.Dial(cfg.Mongo)
	if err != nil {
		mylog.Alert(err)
		panic(err)
	}
	defer func() {
		session.Close()
		mylog.Debugf("mongo session closed %+v", session)
	}()

	mylog.Debugf("mongo session %+v", session)

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	mylog.Debug("mongo session mode set to monotonic")
	db := session.DB(cfg.Database)
	mylog.Debug("mongo database", db)
	l := &gridas.Listener{SendTo: reqChan, PetitionStore: db.C(cfg.Instance + cfg.PetitionsColl)}
	mylog.Debugf("consumer %+v", l)
	c := &gridas.Consumer{GetFrom: reqChan,
		PetitionStore: db.C(cfg.Instance + cfg.PetitionsColl),
		ReplyStore:    db.C(cfg.ResponsesColl),
		ErrorStore:    db.C(cfg.ErrorsColl),
	}
	mylog.Debugf("listener %+v", c)
	r := &gridas.Replyer{ReplyStore: db.C(cfg.ResponsesColl)}
	mylog.Debugf("replyer %+v", r)
	rcvr := &gridas.Recoverer{SendTo: reqChan, PetitionStore: db.C(cfg.Instance + cfg.PetitionsColl)}
	mylog.Debugf("recoverer %+v", rcvr)
	endConsumers := c.Start(cfg.Consumers)
	if err := rcvr.Recover(); err != nil {
		mylog.Alert(err)
		os.Exit(-1)
	}
	http.Handle("/", l)
	http.Handle("/responses/", http.StripPrefix("/responses/", r))
	go func() {
		mylog.Debug("starting HTTP server (listener)")
		err := http.ListenAndServe(":"+cfg.Port, nil)
		if err != nil {
			mylog.Alert(err)
			os.Exit(-1)
		}
	}()
	go func() {
		for {
			session.Refresh()
			time.Sleep(15 * time.Second)
		}
	}()

	onEnd(func() {
		mylog.Info("shutting down gridas ...")
		l.Stop()
		mylog.Debug("listener stopped")
		c.Stop()
		mylog.Debug("consumer stopped")
	})
	<-endConsumers
	mylog.Alert("bye World!")
}

func onEnd(f func()) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-sigCh
		mylog.Debugf("Signal %+v received", s)
		f()
	}()
}

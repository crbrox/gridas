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
	session.SetSocketTimeout(time.Duration(cfg.Timeout) * time.Second)
	session.SetSyncTimeout(time.Duration(cfg.Timeout) * time.Second)
	defer func() {
		session.Close()
		mylog.Debugf("mongo session closed %+v", session)
	}()

	mylog.Debugf("mongo session %+v", session)

	db := session.DB(cfg.Database)
	mylog.Debug("mongo database", db)
	listener := &gridas.Listener{SendTo: reqChan, Cfg: cfg, SessionSeed: session}
	mylog.Debugf("listener %+v", listener)
	consumer := &gridas.Consumer{GetFrom: reqChan, Cfg: cfg, SessionSeed: session}
	mylog.Debugf("consumer %+v", consumer)
	rplyr := &gridas.Replyer{Cfg: cfg, SessionSeed: session}
	mylog.Debugf("replyer %+v", rplyr)
	rcvr := &gridas.Recoverer{SendTo: reqChan, Cfg: cfg, SessionSeed: session}
	mylog.Debugf("recoverer %+v", rcvr)
	endConsumers := consumer.Start(cfg.Consumers)
	if err := rcvr.Recover(); err != nil {
		mylog.Alert(err)
		os.Exit(-1)
	}
	http.Handle("/", listener)
	http.Handle("/responses/", http.StripPrefix("/responses/", rplyr))
	go func() {
		mylog.Debug("starting HTTP server (listener)")
		err := http.ListenAndServe(":"+cfg.Port, nil)
		if err != nil {
			mylog.Alert(err)
			os.Exit(-1)
		}
	}()

	onEnd(func() {
		mylog.Info("shutting down gridas ...")
		listener.Stop()
		mylog.Debug("listener stopped")
		consumer.Stop()
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

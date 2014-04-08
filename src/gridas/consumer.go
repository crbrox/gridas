package gridas

import (
	"net/http"
	"sync"
	"time"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"gridas/config"
	"gridas/mylog"
)

//Consumer is in charge of taking up petitions from the "GetFrom" channel and
//making the actual request to the target host, saving the answer and deleting the
//petition after that.
type Consumer struct {
	//Channel for getting petitions
	GetFrom <-chan *Petition
	//Configuration object
	Cfg *config.Config
	//Session seed for mongo
	SessionSeed *mgo.Session
	//http.Client for making requests to target host
	Client http.Client
	//number of goroutines consuming petitions
	n int
	//channel for asking goroutines to finish
	endChan chan struct{}
	//WaitGroup for goroutines after been notified the should end
	wg sync.WaitGroup
}

//Start starts n goroutines for taking Petitions from the GetFrom channel.
//It returns a channel for notifying when the consumer has ended (hopefully after a Stop() method invocation).
func (c *Consumer) Start(n int) <-chan bool {
	mylog.Debugf("starting consumer %+v", c)
	c.n = n
	finalDone := make(chan bool)
	c.endChan = make(chan struct{})
	c.wg.Add(c.n)
	for i := 0; i < c.n; i++ {
		go c.relay()
	}
	go func() {
		c.wg.Wait()
		mylog.Debug("consumer waiting for children")
		finalDone <- true
		mylog.Debug("all consumer's children finished")
	}()

	return finalDone
}

//Loop of taking a petition and making the request it represents.
func (c *Consumer) relay() {
	defer c.wg.Done()
SERVE:
	for {
		select {
		case <-c.endChan:
			break SERVE
		default:
			select {
			case req := <-c.GetFrom:
				mylog.Debugf("extracted petition %+v", req)
				c.process(req)
			case <-c.endChan:
				break SERVE
			}
		}
	}
}

//process recreates the request that should be sent to the target host
//it stores the response in the store of replies.
func (c *Consumer) process(petition *Petition) {
	var (
		req   *http.Request
		resp  *http.Response
		reply *Reply
		start = bson.Now()
	)

	db := c.SessionSeed.DB(c.Cfg.Database)
	petColl := db.C(c.Cfg.Instance + c.Cfg.PetitionsColl)
	replyColl := db.C(c.Cfg.ResponsesColl)
	errColl := db.C(c.Cfg.ErrorsColl)

	mylog.Debugf("processing petition %+v", petition)
	req, err := petition.Request()
	if err != nil {
		mylog.Alert(petition.ID, err)
		return
	}
	mylog.Debugf("restored request %+v", req)
	mylog.Debug("before making request", petition.ID)
	resp, err = c.doRequest(req, petition.ID)
	if err == nil {
		mylog.Debug("after making request", petition.ID)
		defer func() {
			mylog.Debug("closing response body", petition.ID)
			resp.Body.Close()
		}()
	}

	reply = newReply(resp, petition, err)
	reply.Created = start
	mylog.Debugf("created reply %+v", reply)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e := errColl.Insert(reply)
		if e != nil {
			mylog.Alert("ERROR inserting erroneous reply", petition.ID, err)
			c.SessionSeed.Refresh()
		}
	}
	mylog.Debugf("before insert reply %+v", reply)
	err = replyColl.Insert(reply)
	mylog.Debugf("after insert reply %+v", reply)
	if err != nil {
		mylog.Alert("ERROR inserting reply", petition.ID, err)
		c.SessionSeed.Refresh()
	}
	mylog.Debugf("before remove petition %+v", petition)
	err = petColl.Remove(bson.M{"id": petition.ID})
	mylog.Debugf("after remove petition %+v", petition)
	if err != nil {
		mylog.Alert("ERROR removing petition", petition.ID, err)
		c.SessionSeed.Refresh()
	}

}

//Stop asks consumer to stop taking petitions. When the stop is complete,
//the fact will be notified through the channel returned by the Start() method.
func (c *Consumer) Stop() {
	mylog.Debug("closing consumer end channel")
	close(c.endChan)
}

//doRequest does and retries the request as many times as is set, increasing the time between retries, doubling the initial time
func (c *Consumer) doRequest(req *http.Request, petid string) (resp *http.Response, err error) {
	resp, err = c.Client.Do(req)
	if err == nil && resp.StatusCode != 503 { //Good, not error and non challenging response
		return resp, nil
	}
	mylog.Debug("error making request", petid, err)
	var retryTime = time.Duration(c.Cfg.RetryTime) * time.Millisecond
	var retries = c.Cfg.Retries
	for i := 0; i < retries; i++ {
		time.Sleep(retryTime)
		mylog.Debugf("retrying request %v retry #%v after %v error %v", petid, i+1, retryTime, err)
		resp, err = c.Client.Do(req)
		if err == nil && resp.StatusCode != 503 {
			break
		}
		retryTime *= 2
	}
	return resp, err
}

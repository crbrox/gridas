package gridas

import (
	"net/http"
	"sync"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"gridas/mylog"
)

//Consumer is in charge of taking up petitions from the "GetFrom" channel and
//making the actual request to the target host, saving the answer and deleting the
//petition after that.
type Consumer struct {
	//Channel for getting petitions
	GetFrom <-chan *Petition
	//Store of petitions, for removing when done
	PetitionStore *mgo.Collection
	//Store of replies, for saving responses
	ReplyStore *mgo.Collection
	//Store of replies with error, including not 200 status code
	ErrorStore *mgo.Collection
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
	mylog.Debugf("processing petition %+v", petition)
	req, err := petition.Request()
	if err != nil {
		mylog.Alert(petition.ID, err)
	} else {
		mylog.Debugf("restored request %+v", req)
		mylog.Debug("before making request", petition.ID)
		resp, err = c.Client.Do(req)
		if err != nil {
			mylog.Info("error making request", petition.ID, err)

		} else {
			mylog.Debug("after making request", petition.ID)
			defer func() {
				mylog.Debug("closing response body", petition.ID)
				resp.Body.Close()
			}()
		}
	}
	reply = newReply(resp, petition, err)
	reply.Created = start
	mylog.Debugf("created reply %+v", reply)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e := c.ErrorStore.Insert(reply)
		if e != nil {
			mylog.Alert("ERROR inserting erroneous reply", petition.ID, err)
		}
	}
	mylog.Debugf("before insert reply %+v", reply)
	err = c.ReplyStore.Insert(reply)
	mylog.Debugf("after insert reply %+v", reply)
	if err != nil {
		mylog.Alert("ERROR inserting reply", petition.ID, err)
	}
	mylog.Debugf("before remove petition %+v", petition)
	err = c.PetitionStore.Remove(bson.M{"id": petition.ID})
	mylog.Debugf("after remove petition %+v", petition)
	if err != nil {
		mylog.Alert("ERROR removing petition", petition.ID, err)
	}

}

//Stop asks consumer to stop taking petitions. When the stop is complete,
//the fact will be notified through the channel returned by the Start() method.
func (c *Consumer) Stop() {
	mylog.Debug("closing consumer end channel")
	close(c.endChan)
}

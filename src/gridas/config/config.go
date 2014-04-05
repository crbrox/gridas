/*
Package config provides types and a function for getting grush configuration.

Configuration is read the file grush.ini. An possible content of such file could be
	[default]
	port=8080
	queueSize=100000
	consumers=1000
	storeType=redis


The storeType chooses an store and implies that there is a section in the configuration
file for the type chosen
*/
package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v1"
)

//General configuration data
type Config struct {
	//Port to listen to
	Port string
	//Maximun number of request enqueued, waiting for being processed
	QueueSize int
	//Maximun number of concurrent requests being processed
	Consumers int
	//MongoDB host
	Mongo string
	//Database
	Database string
	//Petitions collection
	PetitionsColl string
	//Responses collection
	ResponsesColl string
	//Errors collection
	ErrorsColl string
	//Instance ID for isolating recoverers
	Instance string
	//Debug mgo
	DebugMgo bool
	//Log level: alert, info, debug
	LogLevel string
}

//ReadConfig reads configuration from file with name filename.
func ReadConfig(filename string) (*Config, error) {

	cfg := Config{}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	//TODO: Check values!!
	return &cfg, nil
}

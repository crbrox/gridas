package gridas

import (
	"testing"

	"github.com/crbrox/gridas/config"

	"labix.org/v2/mgo"
)

var (
	sessionTest  *mgo.Session
	databaseTest *mgo.Database
)
var cfgTest *config.Config

func setUp(t *testing.T) {
	var err error
	cfgTest, err = config.ReadConfig("gridas_test.yaml")
	if err != nil {
		t.Fatal("setUp:", err)
	}
	t.Logf("test configuration: %+v\n", cfgTest)
	sessionTest, err = mgo.Dial(cfgTest.Mongo)
	if err != nil {
		t.Fatal("setUp:", err)
	}
	sessionTest.SetMode(mgo.Monotonic, true)
	databaseTest = sessionTest.DB(cfgTest.Database)

}
func tearDown(t *testing.T) {
	err := databaseTest.DropDatabase()
	if err != nil {
		t.Fatal("setUp:", err)
	}
	sessionTest.Close()

}

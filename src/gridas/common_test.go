package gridas

import (
	"testing"

	"gridas/config"

	"labix.org/v2/mgo"
)

var (
	sessionTest  *mgo.Session
	databaseTest *mgo.Database

	petitionStoreTest, responseStoreTest, errorStoreTest *mgo.Collection
)
var cfg config.Config

func setUp(t *testing.T) {
	cfg, err := config.ReadConfig("gridas_test.yaml")
	if err != nil {
		t.Fatal("setUp:", err)
	}
	t.Logf("test configuration: %+v\n", cfg)
	sessionTest, err = mgo.Dial(cfg.Mongo)
	if err != nil {
		t.Fatal("setUp:", err)
	}
	sessionTest.SetMode(mgo.Monotonic, true)
	databaseTest = sessionTest.DB(cfg.Database)

	petitionStoreTest = databaseTest.C(cfg.PetitionsColl)
	if petitionStoreTest == nil {
		t.Fatal("setUp: petitionStoreTest == nil")
	}
	responseStoreTest = databaseTest.C(cfg.ResponsesColl)
	if responseStoreTest == nil {
		t.Fatal("setUp: responseStoreTest == nil")
	}
	errorStoreTest = databaseTest.C(cfg.ErrorsColl)
	if errorStoreTest == nil {
		t.Fatal("setUp: errorStoreTest == nil")
	}
	err = petitionStoreTest.EnsureIndexKey("id")
	if err != nil {
		t.Fatal("setUp:", err)
	}
	err = responseStoreTest.EnsureIndexKey("id")
	if err != nil {
		t.Fatal("setUp:", err)
	}
	err = errorStoreTest.EnsureIndexKey("id")
	if err != nil {
		t.Fatal("setUp:", err)
	}
}
func tearDown(t *testing.T) {
	var err = petitionStoreTest.DropCollection()
	if err != nil {
		t.Fatal("setUp:", err)
	}
	err = responseStoreTest.DropCollection()
	if err != nil {
		t.Fatal("setUp:", err)
	}
	err = errorStoreTest.DropCollection()
	if err != nil {
		t.Fatal("setUp:", err)
	}
	sessionTest.Close()

}

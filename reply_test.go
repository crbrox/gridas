package gridas

import (
	"errors"
	"reflect"
	"testing"
)

func TestReplyWithError(t *testing.T) {
	id := "1234_abcd"
	p := Petition{ID: id}
	e := errors.New("error for testing Reply")
	rpl := newReply(nil, &p, e)
	if rpl.ID != id {
		t.Error("reply id should be equal to provided")
	}
	if reflect.DeepEqual(rpl.Error, e) {
		t.Errorf("reply error should be equal to provided %#v %#v", rpl.Error, e)
	}
	if reflect.DeepEqual(rpl.Petition, p) {
		t.Errorf("reply petition should be equal to provided %#v %#v", rpl.Petition, p)
	}
}

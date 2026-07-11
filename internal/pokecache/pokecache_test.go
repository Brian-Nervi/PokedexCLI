package pokecache

import (
	"testing"
	"time"
)

func TestAddGet(t *testing.T) {
	ch := NewCache(5 * time.Second)
	ch.Add("test.com", []byte("testdata"))
	val, ok := ch.Get("test.com")
	if !ok {
		t.Errorf("get failed")
	}
	if string(val) != "testdata" {
		t.Errorf("val get error")
	}
}

func TestReapLoop(t *testing.T) {
	ch := NewCache(10 * time.Millisecond)
	ch.Add("test.com", []byte("testdata"))
	val, ok := ch.Get("test.com")
	if !ok {
		t.Errorf("get failed")
	}
	if string(val) != "testdata" {
		t.Errorf("val get error")
	}
	time.Sleep(11 * time.Millisecond)
	_, ok = ch.Get("test.com")
	if ok {
		t.Errorf("get failed")
	}
}

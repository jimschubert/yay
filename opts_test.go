package yay

import (
	"testing"
)

func TestNewOptions_Defaults(t *testing.T) {
	o := &opts{}
	fn := NewOptions()
	fn(o)
	if o.skipDocumentCheck != false {
		t.Errorf("expected skipDocumentCheck to be false, got %v", o.skipDocumentCheck)
	}
	if o.initialized != true {
		t.Errorf("expected initialized to be true, got %v", o.initialized)
	}
}

func TestWithSkipDocumentCheck(t *testing.T) {
	o := &opts{}
	fn := NewOptions().WithSkipDocumentCheck(true)
	fn(o)
	if o.skipDocumentCheck != true {
		t.Errorf("expected skipDocumentCheck to be true, got %v", o.skipDocumentCheck)
	}
}

func TestFnOptions_Chaining(t *testing.T) {
	o := &opts{}
	fn := NewOptions().WithSkipDocumentCheck(true).WithSkipDocumentCheck(false)
	fn(o)
	if o.skipDocumentCheck != false {
		t.Errorf("expected skipDocumentCheck to be false after chaining, got %v", o.skipDocumentCheck)
	}
}


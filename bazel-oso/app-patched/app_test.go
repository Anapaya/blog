package main_test

import (
	"testing"

	"github.com/osohq/go-oso"
)

func TestPolicy(t *testing.T) {
	oso, err := oso.NewOso()
	if err != nil {
		t.Fatal(err)
	}
	if err := oso.LoadFiles([]string{"testdata/policy.polar"}); err != nil {
		t.Fatal(err)
	}
	if err := oso.Authorize("actor", "action", "resource"); err != nil {
		t.Fatal(err)
	}
}

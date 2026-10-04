package platform

import (
	"os"
	"testing"
)

func TestSignallable(t *testing.T) {
	alive, permitted := Signallable(os.Getpid())
	if !alive || !permitted {
		t.Fatalf("self: alive=%v permitted=%v", alive, permitted)
	}
	if alive, _ := Signallable(1 << 30); alive {
		t.Fatal("bogus pid must not be alive")
	}
	if !Alive(os.Getpid()) {
		t.Fatal("self must be alive")
	}
}

package jobq

import (
	"log"
	"runtime"
	"testing"
)

func TestMain(b *testing.M) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Lshortfile)

	v := b.Run()
	_ = v
}

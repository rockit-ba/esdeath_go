package main

import (
	cmap "github.com/orcaman/concurrent-map/v2"
	rand2 "math/rand/v2"
	"sync"
	"testing"
)

func TestPullServiceLock(t *testing.T) {
	wg := &sync.WaitGroup{}
	count := 0
	lock := &PullServiceLock{groups: cmap.New[*ServiceLock]()}
	serviceLk := newServiceLock("test", "test")
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			lk := lock.getOrAddLock(serviceLk)
			lk.Lock()
			defer func() {
				lk.Unlock()
				wg.Done()
			}()
			count++
		}()
	}
	wg.Wait()
	if count != 10 {
		t.Errorf("count: %d", count)
	}
	t.Log("count: ", count)
}

func TestRand(t *testing.T) {
	for i := 0; i < 20; i++ {
		t.Log(rand2.Int64N(5000))
	}
}

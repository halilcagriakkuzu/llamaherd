package main

import (
	"sync"
	"testing"

	"llamaherd/schema"
)

// TestConcurrentStateAccess is the regression test for the unsynchronized App
// fields: config writes, snapshots and the read-only API surface run together.
// Run it with -race (CI does).
func TestConcurrentStateAccess(t *testing.T) {
	a := NewApp()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(4)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				a.UpdateConfig(schema.Config{Basic: schema.Basic{Context: 1024 + n*j%4096}})
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = a.GetState()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = a.Models()
				_ = a.Presets("")
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = a.SnapshotJSON()
			}
		}()
	}
	wg.Wait()
}

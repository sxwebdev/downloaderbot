package util

import (
	"sync"
	"testing"
)

func TestSliceAdd_Concurrent(t *testing.T) {
	const n = 1000
	s := NewSlice[int]()

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			s.Add(v)
		}(i)
	}
	wg.Wait()

	if got := s.Len(); got != n {
		t.Fatalf("Len()=%d, want %d", got, n)
	}
	if got := len(s.GetAll()); got != n {
		t.Fatalf("len(GetAll())=%d, want %d", got, n)
	}
}

func TestSliceAddToIndex_Concurrent(t *testing.T) {
	const n = 100
	s := NewSliceWithLength[int](n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			s.AddToIndex(idx, idx*2)
		}(i)
	}
	wg.Wait()

	got := s.GetAll()
	for i := range n {
		if got[i] != i*2 {
			t.Fatalf("got[%d]=%d, want %d", i, got[i], i*2)
		}
	}
}

func TestSliceAddMany(t *testing.T) {
	s := NewSlice[string]()
	s.AddMany([]string{"a", "b"})
	s.AddMany([]string{"c"})

	got := s.GetAll()
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Fatalf("got[%d]=%q, want %q", i, got[i], v)
		}
	}
}

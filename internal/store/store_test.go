package store

import (
	"fmt"
	"sync"
	"testing"
)

func TestStoreCRUD(t *testing.T) {
	s := New()

	created := s.Create("first")
	if created.ID == "" {
		t.Fatal("expected id")
	}

	got, ok := s.Get(created.ID)
	if !ok || got.Title != "first" {
		t.Fatalf("expected todo to exist, got: %+v", got)
	}

	newTitle := "updated"
	completed := true
	updated, ok := s.Update(created.ID, UpdateInput{Title: &newTitle, Completed: &completed})
	if !ok || updated.Title != "updated" || !updated.Completed {
		t.Fatalf("unexpected update result: %+v", updated)
	}

	if !s.Delete(created.ID) {
		t.Fatal("expected delete to succeed")
	}
	if _, ok := s.Get(created.ID); ok {
		t.Fatal("expected item to be deleted")
	}
}

func TestStoreConcurrentCreate(t *testing.T) {
	s := New()
	const workers = 25

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Create(fmt.Sprintf("todo-%d", i))
		}(i)
	}
	wg.Wait()

	items := s.List()
	if len(items) != workers {
		t.Fatalf("expected %d items, got %d", workers, len(items))
	}
}

package services

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestSkillStore_PutAndGet(t *testing.T) {
	s := NewSkillStore()
	task := &SkillTask{ID: "t-1", Skill: "research", Status: "running", CreatedAt: time.Now()}
	s.Put(task)

	got := s.Get("t-1")
	if got == nil {
		t.Fatal("Get returned nil for present task")
	}
	if got != task {
		t.Errorf("Get returned a different pointer than Put")
	}

	if s.Get("missing") != nil {
		t.Errorf("Get(missing) must return nil")
	}
}

func TestSkillStore_Put_Overwrites(t *testing.T) {
	s := NewSkillStore()
	s.Put(&SkillTask{ID: "t-1", Status: "running", CreatedAt: time.Now()})
	s.Put(&SkillTask{ID: "t-1", Status: "completed", CreatedAt: time.Now()})

	got := s.Get("t-1")
	if got == nil || got.Status != "completed" {
		t.Errorf("Put should overwrite by ID, got %+v", got)
	}
}

func TestSkillStore_ConcurrentAccess(t *testing.T) {
	// Detect data races (when run with -race) on the in-memory store.
	s := NewSkillStore()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		id := strconv.Itoa(i)
		go func() {
			defer wg.Done()
			s.Put(&SkillTask{ID: id, Status: "running", CreatedAt: time.Now()})
		}()
		go func() {
			defer wg.Done()
			s.Get(id)
		}()
	}
	wg.Wait()
	// Spot-check some entries landed
	if s.Get("0") == nil || s.Get("99") == nil {
		t.Errorf("expected entries to be present after concurrent puts")
	}
}

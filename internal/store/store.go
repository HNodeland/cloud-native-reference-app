package store

import (
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

type Todo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type UpdateInput struct {
	Title     *string `json:"title,omitempty"`
	Completed *bool   `json:"completed,omitempty"`
}

type Store struct {
	mu     sync.RWMutex
	todos  map[string]Todo
	nextID atomic.Int64
}

func New() *Store {
	return &Store{
		todos: make(map[string]Todo),
	}
}

func (s *Store) List() []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		items = append(items, todo)
	}
	sort.Slice(items, func(i, j int) bool {
		return idToInt(items[i].ID) < idToInt(items[j].ID)
	})
	return items
}

func (s *Store) Create(title string) Todo {
	id := strconv.FormatInt(s.nextID.Add(1), 10)
	todo := Todo{ID: id, Title: title, Completed: false}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos[id] = todo
	return todo
}

func (s *Store) Get(id string) (Todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	todo, ok := s.todos[id]
	return todo, ok
}

func (s *Store) Update(id string, input UpdateInput) (Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	todo, ok := s.todos[id]
	if !ok {
		return Todo{}, false
	}
	if input.Title != nil {
		todo.Title = *input.Title
	}
	if input.Completed != nil {
		todo.Completed = *input.Completed
	}
	s.todos[id] = todo
	return todo, true
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.todos[id]; !ok {
		return false
	}
	delete(s.todos, id)
	return true
}

func idToInt(value string) int64 {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

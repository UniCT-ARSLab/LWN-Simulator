package scheduler

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

type scheduledItem struct {
	event types.ScheduledEvent
	index int
}

type scheduledQueue []*scheduledItem

func (q scheduledQueue) Len() int { return len(q) }

func (q scheduledQueue) Less(i int, j int) bool {
	if q[i].event.At == q[j].event.At {
		return q[i].event.ID < q[j].event.ID
	}
	return q[i].event.At < q[j].event.At
}

func (q scheduledQueue) Swap(i int, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *scheduledQueue) Push(value any) {
	item := value.(*scheduledItem)
	item.index = len(*q)
	*q = append(*q, item)
}

func (q *scheduledQueue) Pop() any {
	old := *q
	item := old[len(old)-1]
	old[len(old)-1] = nil
	item.index = -1
	*q = old[:len(old)-1]
	return item
}

type Scheduler struct {
	mu    sync.Mutex
	queue scheduledQueue
	items map[string]*scheduledItem
}

func New() *Scheduler {
	s := &Scheduler{items: make(map[string]*scheduledItem)}
	heap.Init(&s.queue)
	return s
}

func (s *Scheduler) Schedule(event types.ScheduledEvent) error {
	if event.ID == "" {
		return fmt.Errorf("scheduled event id is required")
	}
	if event.At < 0 {
		return fmt.Errorf("scheduled event time cannot be negative")
	}
	if !event.Type.Valid() {
		return fmt.Errorf("unsupported scheduled event type %q", event.Type)
	}
	if event.Message == "" {
		return fmt.Errorf("scheduled event message is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[event.ID]; exists {
		return fmt.Errorf("scheduled event %q already exists", event.ID)
	}

	item := &scheduledItem{event: event}
	s.items[event.ID] = item
	heap.Push(&s.queue, item)
	return nil
}

func (s *Scheduler) Cancel(id string) (types.ScheduledEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.items[id]
	if !exists {
		return types.ScheduledEvent{}, false
	}

	heap.Remove(&s.queue, item.index)
	delete(s.items, id)
	return item.event, true
}

func (s *Scheduler) PopDue(now time.Duration) (types.ScheduledEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 || s.queue[0].event.At > now {
		return types.ScheduledEvent{}, false
	}

	item := heap.Pop(&s.queue).(*scheduledItem)
	delete(s.items, item.event.ID)
	return item.event, true
}

func (s *Scheduler) Peek() (types.ScheduledEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 {
		return types.ScheduledEvent{}, false
	}
	return s.queue[0].event, true
}

func (s *Scheduler) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue)
}

// Events returns a stable copy of all pending events for checkpointing.
func (s *Scheduler) Events() []types.ScheduledEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := make([]types.ScheduledEvent, 0, len(s.queue))
	for _, item := range s.queue {
		events = append(events, item.event)
	}
	return events
}

func (s *Scheduler) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = scheduledQueue{}
	s.items = make(map[string]*scheduledItem)
	heap.Init(&s.queue)
}

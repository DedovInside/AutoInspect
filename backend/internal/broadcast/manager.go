package broadcast

import (
	"context"
	"sync"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/notify"
	"github.com/google/uuid"
)

type Manager struct {
	mu     sync.RWMutex
	byUser map[uuid.UUID]map[uuid.UUID]chan notify.JobEvent
}

func NewManager() *Manager {
	return &Manager{
		byUser: make(map[uuid.UUID]map[uuid.UUID]chan notify.JobEvent),
	}
}

func (m *Manager) SubscribeUser(userID uuid.UUID, buffer int) (subscriberID uuid.UUID, ch <-chan notify.JobEvent, unsubscribe func()) {
	if buffer <= 0 {
		buffer = 1
	}
	subscriberID = uuid.New()
	raw := make(chan notify.JobEvent, buffer)

	m.mu.Lock()
	if _, ok := m.byUser[userID]; !ok {
		m.byUser[userID] = make(map[uuid.UUID]chan notify.JobEvent)
	}
	m.byUser[userID][subscriberID] = raw
	m.mu.Unlock()

	unsubscribe = func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		subs, ok := m.byUser[userID]
		if !ok {
			return
		}
		ch, exists := subs[subscriberID]
		if exists {
			delete(subs, subscriberID)
			close(ch)
		}
		if len(subs) == 0 {
			delete(m.byUser, userID)
		}
	}
	return subscriberID, raw, unsubscribe
}

func (m *Manager) NotifyJobEvent(_ context.Context, event *notify.JobEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	if event.UserID != uuid.Nil {
		if userSubs, ok := m.byUser[event.UserID]; ok {
			for _, ch := range userSubs {
				select {
				case ch <- *event:
				default:
				}
			}
		}
	}
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, subs := range m.byUser {
		for _, ch := range subs {
			close(ch)
		}
	}
	m.byUser = nil
}

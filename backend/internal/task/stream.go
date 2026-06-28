package task

import (
	"context"
	"database/sql"
	"errors"
)

func (m *Manager) SubscribeMetrics(ctx context.Context, taskID int64) (<-chan MetricPoint, func(), error) {
	_, err := m.queries.GetTask(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrTaskNotFound
		}
		return nil, nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.running[taskID]; !ok {
		return nil, nil, ErrTaskNotRunning
	}
	ch := make(chan MetricPoint, 16)

	if m.subscribers[taskID] == nil {
		m.subscribers[taskID] = make(map[chan MetricPoint]struct{})
	}
	m.subscribers[taskID][ch] = struct{}{}

	unsubscribe := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		delete(m.subscribers[taskID], ch)
		close(ch)

		if len(m.subscribers[taskID]) == 0 {
			delete(m.subscribers, taskID)
		}
	}
	return ch, unsubscribe, nil
}

func (m *Manager) publishMetric(taskID int64, point MetricPoint) {
	m.mu.Lock()
	chs := m.subscribers[taskID]
	m.mu.Unlock()

	for ch := range chs {
		select {
		case ch <- point:
		default:
		}
	}
}

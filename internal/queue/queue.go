package queue

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Message struct {
	ID                 string
	Payload            []byte
	CreatedAt          time.Time
	DeliveryCount      int
	VisibilityDeadline time.Time
}

type QueueStats struct {
	Pending    int
	InFlight   int
	DeadLetter int
}

type InMemoryQueue struct {
	name             string
	pending          []*Message
	inFlight         map[string]*Message
	deadLetter       []*Message
	mu               sync.Mutex
	cond             *sync.Cond
	maxDeliveryCount int
	leaseDuration    time.Duration
}

// NewInMemoryQueue creates a new queue instance.
func NewInMemoryQueue(name string, lease time.Duration, maxDelivery int) *InMemoryQueue {
	q := &InMemoryQueue{
		name:             name,
		pending:          []*Message{},
		inFlight:         make(map[string]*Message),
		deadLetter:       []*Message{},
		leaseDuration:    lease,
		maxDeliveryCount: maxDelivery,
	}
	q.cond = sync.NewCond(&q.mu)

	// Start background requeue goroutine
	go q.requeueExpired()
	return q
}

// generateMessageID creates a unique ID
func generateMessageID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (q *InMemoryQueue) Publish(payload []byte) (string, error) {
	id, err := generateMessageID()

	if err != nil {
		return "", err
	}

	msg := &Message{
		ID:        id,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	q.mu.Lock()
	q.pending = append(q.pending, msg)
	q.cond.Signal() //wake up blocked consumer
	q.mu.Unlock()
	return id, nil
}

// Claim blocks up to wait duration for a message
func (q *InMemoryQueue) Claim(wait time.Duration) (*Message, error) {
	deadline := time.Now().Add(wait)
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		if len(q.pending) > 0 {
			msg := q.pending[0]
			q.pending = q.pending[1:]
			msg.DeliveryCount++
			msg.VisibilityDeadline = time.Now().Add(q.leaseDuration)

			if msg.DeliveryCount > q.maxDeliveryCount {
				q.deadLetter = append(q.deadLetter, msg)
				continue
			}
			q.inFlight[msg.ID] = msg
			return msg, nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, errors.New("timeout: no messages available")
		}

		q.cond.Wait()
	}
}

func (q *InMemoryQueue) Ack(messageID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.inFlight[messageID]; !ok {
		return fmt.Errorf("message not found in-flight: %s", messageID)
	}
	delete(q.inFlight, messageID)
	return nil
}

func (q *InMemoryQueue) Stats() QueueStats {
	q.mu.Lock()
	defer q.mu.Unlock()
	return QueueStats{
		Pending:    len(q.pending),
		InFlight:   len(q.inFlight),
		DeadLetter: len(q.deadLetter),
	}
}

func (q *InMemoryQueue) requeueExpired() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	batchSize := 100

	for range ticker.C {
		now := time.Now()
		cnt := 0
		q.mu.Lock()
		for id, msg := range q.inFlight {
			if cnt >= batchSize {
				break
			}
			if now.After(msg.VisibilityDeadline) {
				delete(q.inFlight, id)
				if msg.DeliveryCount > q.maxDeliveryCount {
					q.deadLetter = append(q.deadLetter, msg)
				} else {
					q.pending = append(q.pending, msg)
					q.cond.Signal()
				}
				cnt++
			}

		}
		q.mu.Unlock()
	}
}

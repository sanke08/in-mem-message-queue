package broker

import (
	"sync"
	"time"

	"github.com/sanke08/in-mem-message-queue/internal/queue"
)

type Broker struct {
	queues        map[string]*queue.InMemoryQueue
	mu            sync.RWMutex
	maxDelivery   int
	leaseDuration time.Duration
}

func NewBroker(leaseDuration time.Duration, maxDelivery int) *Broker {
	return &Broker{
		queues:        make(map[string]*queue.InMemoryQueue),
		maxDelivery:   maxDelivery,
		leaseDuration: leaseDuration,
	}
}

func (b *Broker) getQueue(fullQueueName string) *queue.InMemoryQueue {
	b.mu.Lock()
	defer b.mu.Unlock()

	q, ok := b.queues[fullQueueName]

	if !ok {
		q = queue.NewInMemoryQueue(fullQueueName, b.leaseDuration, b.maxDelivery)
		b.queues[fullQueueName] = q
	}
	return q
}

func (b *Broker) Publish(fullQueueName string, payload []byte) (string, error) {
	return b.getQueue(fullQueueName).Publish(payload)
}

func (b *Broker) Claim(fullQueueName string, wait time.Duration) (*queue.Message, error) {
	return b.getQueue(fullQueueName).Claim(wait)
}

func (b *Broker) Ack(fullQueueName string, messageID string) error {
	return b.getQueue(fullQueueName).Ack(messageID)
}

// func (b *Broker) ExtendLease(fullQueueName string, messageID string, extra time.Duration) error {
// 	return b.getQueue(fullQueueName).ExtendLease(messageID, extra)
// }

func (b *Broker) Stats(fullQueueName string) queue.QueueStats {
	return b.getQueue(fullQueueName).Stats()
}

# Thread-Safe In-Memory Pub/Sub Broker

## Problem Statement

Design and implement a **thread-safe in-memory Publish/Subscribe broker in Go**.

Publishers can publish messages to a topic, and all subscribers currently subscribed to that topic should receive the message.

## Requirements

- Support multiple topics.
- Each topic can have multiple subscribers.
- Multiple publishers can publish concurrently.
- Subscribers can subscribe and unsubscribe concurrently.
- Each subscriber must have its own bounded message buffer.
- A slow subscriber must not block other subscribers.
- If a subscriber's buffer is full, the message is dropped **only for that subscriber**.
- Message dropping for one subscriber must not affect delivery to other subscribers.
- `Unsubscribe` must be safe while messages are being published.
- The implementation must prevent `send on closed channel` panics.
- When a subscriber is disconnected, its channel must be closed safely.
- Messages already buffered in a subscriber's channel may still be consumed after the channel is closed, until the buffer is drained.
- `Shutdown` must stop accepting new subscriptions and publications.
- Shutdown must safely disconnect all existing subscribers.
- The implementation must not have data races, deadlocks, or goroutine leaks.
- The broker must be completely in-memory.
- Message persistence and guaranteed delivery are out of scope.

## Interface

```go
type Broker interface {
    Subscribe(topic string) (*Subscriber, error)
    Publish(topic string, msg Message) error
    Unsubscribe(topic string, sub *Subscriber) error
    Shutdown()
}
```

```go
type Message struct {
    ID   string
    Data string
}
```

## Example

```text
Topic: orders.created

              Publish(MSG1)
                    |
                    v
              orders.created
               /    |    \
              v     v     v
             A      B      C
```

If all subscriber buffers have available capacity, `MSG1` should be delivered to all three subscribers.

```text
             A      B      C
             ✓      ✓      ✓
```

If subscriber `B` is slow and its buffer is full:

```text
              Publish(MSG2)
                    |
                    v
              orders.created
               /    |    \
              v     v     v
             A      B      C
             ✓    DROP     ✓
```

`MSG2` is dropped only for subscriber `B`.

Subscribers `A` and `C` must continue receiving messages without being blocked by `B`.

## Unsubscribe Behavior

When a subscriber unsubscribes, it must stop receiving newly published messages and its message channel must be safely closed.

Closing the channel does not discard messages that were already buffered.

For example:

```text
Before Unsubscribe:

Subscriber B
    |
    v
[M1][M2][M3]

        Unsubscribe()
             |
             v
       Channel Closed
             |
             v
Consumer may still read:

M1 -> M2 -> M3 -> channel drained
```

Once the buffered messages have been consumed, the consumer observes that the channel has been closed.

## Scope

This broker provides **best-effort in-memory delivery**.

A successful publish does not guarantee that every subscriber received the message. A subscriber whose buffer is full may lose that particular message without affecting other subscribers.

Messages are not persisted. If guaranteed delivery, retries, acknowledgements, replay, or recovery after process failure are required, a durable storage mechanism would be needed. Those features are outside the scope of this problem.
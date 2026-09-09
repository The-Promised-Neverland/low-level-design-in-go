package main

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

type ConnStatus string

const (
	StatusIdle  ConnStatus = "IDLE"
	StatusInUse ConnStatus = "IN_USE"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "HEALTHLY"
	StatusUnhealthy HealthStatus = "UN_HEALTHLY"
)

type ConnectionPool interface {
	Acquire(ctx context.Context) (*Connection, error) // Borrow a connection
	Release(conn *Connection) error                   // Release it back to the connection Pool
	Shutdown()                                        // Graceful shutdown
}

type Connection struct {
	ID       string       // Randomized 6 characters
	Health   HealthStatus // In test we shall randomly mark a connection as unhealthy to simulate real world linkages
	Status   ConnStatus
	LastUsed time.Time
}

func newConnection() *Connection {
	return &Connection{
		ID:       randomIDGenerator(),
		Health:   StatusHealthy,
		Status:   StatusIdle, // This is not yet given to the user. User might just cancel it before it gets in_use
		LastUsed: time.Now(),
	}
}

type Pool struct {
	ctx              context.Context
	cancel           context.CancelFunc
	maxConnections   int // Max connections limit
	idleConnections  int // idle connections limit
	totalConnections int // in-use + idle
	idleTimeout      time.Duration
	idleConnChan     chan *Connection
	connections      map[*Connection]struct{}
	connWG           sync.WaitGroup
	closed           bool
	mu               sync.Mutex
}

func NewConnectionPool(maxConnections int, idleTimeout time.Duration, idleConnections int) ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		ctx:             ctx,
		cancel:          cancel,
		maxConnections:  maxConnections,
		idleConnections: idleConnections,
		idleTimeout:     idleTimeout,
		idleConnChan:    make(chan *Connection, idleConnections),
		connections:     make(map[*Connection]struct{}),
	}
}

func randomIDGenerator() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	id := make([]byte, 6)
	for i := range id {
		id[i] = chars[rand.Intn(len(chars))]
	}
	return string(id)
}

// Acquire provides a connection to the caller.
func (p *Pool) Acquire(ctx context.Context) (*Connection, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		case conn := <-p.idleConnChan: // Channels are inherently thread safe
			p.mu.Lock()
			if p.closed {
				p.mu.Unlock()
				return nil, context.Canceled
			}
			if conn.Health == StatusHealthy && time.Since(conn.LastUsed) <= p.idleTimeout && conn.Status == StatusIdle {
				conn.Status = StatusInUse
				p.connWG.Add(1)
				p.mu.Unlock()
				return conn, nil
			} else {
				delete(p.connections, conn)
				p.totalConnections-- // Because that connection was idle and timed out
			}
			p.mu.Unlock()
		default: // no idle connections. Let us try our luck and see if new connection be created
		}
		p.mu.Lock()
		if p.totalConnections < p.maxConnections && !p.closed {
			conn := newConnection()
			conn.Status = StatusInUse
			p.totalConnections++
			p.connWG.Add(1)
			p.connections[conn] = struct{}{}
			p.mu.Unlock()
			return conn, nil
		}
		p.mu.Unlock()
		// DOOMED: Wait for a connection to be released back to the pool
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		case conn := <-p.idleConnChan:
			p.mu.Lock()
			if p.closed {
				p.mu.Unlock()
				return nil, context.Canceled
			}
			if conn.Health == StatusHealthy && time.Since(conn.LastUsed) <= p.idleTimeout && conn.Status == StatusIdle {
				conn.Status = StatusInUse
				p.connWG.Add(1)
				p.mu.Unlock()
				return conn, nil
			} else {
				delete(p.connections, conn)
				p.totalConnections-- // Because that connection was idle and timed out
			}
			p.mu.Unlock()
		}
	}
}

// Dump the connection in idle mode
func (p *Pool) Release(conn *Connection) error {
	if conn == nil {
		return errors.New("connection is nil")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.connections[conn]; !exists {
		return errors.New("connection does not belong to this pool")
	}
	if conn.Status != StatusInUse {
		return errors.New("connection is not in use")
	}
	conn.Status = StatusIdle
	p.connWG.Done()
	if p.closed {
		delete(p.connections, conn)
		p.totalConnections--
		return nil
	}
	conn.LastUsed = time.Now()
	if len(p.idleConnChan) == p.idleConnections {
		delete(p.connections, conn)
		p.totalConnections--
		return nil
	}
	p.idleConnChan <- conn
	return nil
}

func (p *Pool) Shutdown() {
	p.mu.Lock()
	p.closed = true
	p.cancel() // Trigger pool wide intent of stopping. Stop accepting new connections
	p.mu.Unlock()
	p.connWG.Wait()
	for {
		select {
		case conn := <-p.idleConnChan:
			p.mu.Lock()
			delete(p.connections, conn)
			p.totalConnections--
			p.mu.Unlock()
		default:
			return
		}
	}
}

func main() {

}

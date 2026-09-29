package main

import (
	"cmp"
	"container/heap"
	"errors"
	"slices"
	"sync"
	"time"
)

var ErrInvalidRequest = errors.New("invalid request")

type Direction string

const (
	Up   Direction = "UP"
	Down Direction = "DOWN"
	Idle Direction = "IDLE"
)

const (
	timePerFloorChange = 3 // seconds to move one floor
	doorStayTime       = 5 // seconds doors stay open
)

type DoorState string

const (
	DoorOpen   DoorState = "OPEN"
	DoorClosed DoorState = "CLOSED"
)

type RequestType string

const (
	External RequestType = "EXTERNAL"
	Internal RequestType = "INTERNAL"
)

type Request struct {
	Floor     int
	Direction Direction
	Type      RequestType
}

type ElevatorState struct {
	ID           string
	CurrentFloor int
	Direction    Direction
	DoorState    DoorState
	Pending      []Request
}

type Elevator interface {
	Call(floor int, direction Direction) error
	SelectFloor(floor int) error
	State() ElevatorState
}

type SchedulingStrategy interface {
	AddRequest(request Request)
	NextRequest(current ElevatorState) (Request, bool)
	Pending() []Request
}

type Directional struct {
	up      *MinHeap
	down    *MaxHeap
	sweep   Direction
	pending map[Request]bool
	mu      sync.Mutex
}

func NewDirectional() *Directional {
	return &Directional{
		up:      &MinHeap{},
		down:    &MaxHeap{},
		sweep:   Up,
		pending: make(map[Request]bool),
	}
}

func (s *Directional) AddRequest(request Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending[request] {
		return
	}
	s.pending[request] = true
	if request.Direction == Down {
		heap.Push(s.down, request)
	} else {
		heap.Push(s.up, request)
	}
}

func (s *Directional) NextRequest(current ElevatorState) (Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rebucket(current.CurrentFloor)
	for range 2 {
		if s.sweep == Up && s.up.Len() > 0 {
			return s.take(heap.Pop(s.up).(Request)), true
		}
		if s.sweep == Down && s.down.Len() > 0 {
			return s.take(heap.Pop(s.down).(Request)), true
		}
		s.sweep = opposite(s.sweep)
	}
	return Request{}, false
}

func (s *Directional) rebucket(floor int) {
	for s.up.Len() > 0 && s.up.requestHeap[0].Floor < floor {
		heap.Push(s.down, heap.Pop(s.up))
	}
	for s.down.Len() > 0 && s.down.requestHeap[0].Floor > floor {
		heap.Push(s.up, heap.Pop(s.down))
	}
}

func (s *Directional) take(request Request) Request {
	delete(s.pending, request)
	return request
}

func (s *Directional) Pending() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	up := slices.Clone(s.up.requestHeap)
	slices.SortFunc(up, func(a, b Request) int { return cmp.Compare(a.Floor, b.Floor) })
	down := slices.Clone(s.down.requestHeap)
	slices.SortFunc(down, func(a, b Request) int { return cmp.Compare(b.Floor, a.Floor) })
	if s.sweep == Down {
		return append(down, up...)
	}
	return append(up, down...)
}

func opposite(d Direction) Direction {
	if d == Up {
		return Down
	}
	return Up
}

type FIFO struct {
	requests []Request
	pending  map[Request]bool
	mu       sync.Mutex
}

func (s *FIFO) AddRequest(request Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending[request] {
		return
	}
	s.pending[request] = true
	s.requests = append(s.requests, request)
}

func (s *FIFO) NextRequest(current ElevatorState) (Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requests) == 0 {
		return Request{}, false
	}
	request := s.requests[0]
	s.requests = s.requests[1:]
	delete(s.pending, request)
	return request, true
}

func (s *FIFO) Pending() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Request, len(s.requests))
	copy(out, s.requests)
	return out
}

type ElevatorSystem struct {
	id       string
	floors   int
	elevator ElevatorState
	workCh   chan struct{}
	strategy SchedulingStrategy
	mu       sync.RWMutex
	shutdown chan struct{}
}

func NewElevatorSystem(id string, floors int) *ElevatorSystem {
	system := &ElevatorSystem{
		id:     id,
		floors: floors,
		elevator: ElevatorState{
			ID:           id,
			CurrentFloor: 1,
			Direction:    Idle,
			DoorState:    DoorClosed,
		},
		workCh:   make(chan struct{}, 1),
		strategy: NewDirectional(),
		shutdown: make(chan struct{}),
	}
	go system.processor()
	return system
}

func boundaryChecks(floor int, direction Direction, totalFloors int) bool {
	if floor < 1 || floor > totalFloors {
		return false
	}
	if floor == 1 && direction == Down {
		return false
	}
	if floor == totalFloors && direction == Up {
		return false
	}
	return true
}

func (e *ElevatorSystem) Call(floor int, direction Direction) error {
	if !boundaryChecks(floor, direction, e.floors) {
		return ErrInvalidRequest
	}
	request := Request{
		Floor:     floor,
		Direction: direction,
		Type:      External,
	}
	e.strategy.AddRequest(request)
	e.signalWork()
	return nil
}

func (e *ElevatorSystem) SelectFloor(floor int) error {
	if floor < 1 || floor > e.floors {
		return ErrInvalidRequest
	}
	e.mu.RLock()
	current := e.elevator.CurrentFloor
	e.mu.RUnlock()
	dir := Up
	if floor < current {
		dir = Down
	}
	request := Request{
		Floor:     floor,
		Direction: dir,
		Type:      Internal,
	}
	e.strategy.AddRequest(request)
	e.signalWork()
	return nil
}

func (e *ElevatorSystem) signalWork() {
	select {
	case e.workCh <- struct{}{}:
	default:
	}
}

func (e *ElevatorSystem) processor() {
	for {
		select {
		case <-e.shutdown:
			return
		case <-e.workCh:
			for {
				nextreq, available := e.strategy.NextRequest(e.State())
				if !available {
					break
				}
				e.moveTo(nextreq.Floor)
			}
		}
	}
}

func (e *ElevatorSystem) moveTo(toFloor int) {
	e.mu.RLock()
	currentFloor := e.elevator.CurrentFloor
	e.mu.RUnlock()
	distance := toFloor - currentFloor
	step := 0
	e.mu.Lock()
	if distance < 0 {
		e.elevator.Direction = Down
		step = -1
		distance = -distance
	} else if distance > 0 {
		step = 1
		e.elevator.Direction = Up
	} else {
		e.elevator.Direction = Idle
	}
	e.mu.Unlock()
	for range distance {
		select {
		case <-e.shutdown:
			return
		case <-time.After(time.Duration(timePerFloorChange) * time.Second):
		}
		e.mu.Lock()
		e.elevator.CurrentFloor += step
		e.mu.Unlock()
	}
	e.mu.Lock()
	e.elevator.DoorState = DoorOpen
	e.elevator.Direction = Idle
	e.mu.Unlock()
	select {
	case <-e.shutdown:
		return
	case <-time.After(time.Duration(doorStayTime) * time.Second):
	}
	e.mu.Lock()
	e.elevator.DoorState = DoorClosed
	e.mu.Unlock()
}

func (e *ElevatorSystem) State() ElevatorState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	state := e.elevator
	state.Pending = e.strategy.Pending()
	return state
}

func (e *ElevatorSystem) Shutdown() {
	select {
	case <-e.shutdown:
	default:
		close(e.shutdown)
	}
}

Automated Elevator System

Problem Statement

Design an in-memory automated elevator system for a multi-floor
building.

The building contains multiple elevators operating independently and
concurrently. Each floor can be served by multiple elevator shafts. A
passenger waiting on a floor requests an elevator by specifying the
direction they want to travel, and the system decides which elevator
should serve that request.

The system must simulate realistic elevator behavior including movement
time, door operations, passenger weight, overload conditions, concurrent
requests, and configurable elevator dispatch strategies.

Building

The building contains multiple floors and multiple elevators.

Each elevator operates in its own shaft and can serve the floors
configured for that elevator. A floor may therefore have multiple
elevator doors.

Passengers waiting on a floor do not select a particular elevator. They
request an elevator using the floor controls and the system assigns an
elevator to them.

For example, a floor with three elevators may look conceptually like:

Floor 5

        ↑       ↓

    ┌──────┐ ┌──────┐ ┌──────┐
    │  E1  │ │  E2  │ │  E3  │
    │ DOOR │ │ DOOR │ │ DOOR │
    └──────┘ └──────┘ └──────┘

Floor Requests

A passenger waiting on a floor can request an elevator by providing:

Current floor

Intended direction: UP or DOWN

Example:

Floor: 7
Direction: UP

The passenger does not specify which elevator should arrive.

The system is responsible for selecting an elevator to serve the
request.

Multiple floor requests may occur concurrently.

Destination Selection

After entering an elevator, a passenger can select a destination floor.

Example:

Elevator: E2
Current Floor: 7
Destination: 14

An elevator may have multiple pending stops.

New destination requests may be received while the elevator is already
moving or processing other stops.

Elevator Movement

Movement is not instantaneous.

Each elevator has configurable timing for:

Moving between two adjacent floors

Opening its doors

Keeping its doors open

Closing its doors

For example, an elevator travelling from Floor 2 to Floor 5 must
physically progress through the intermediate floors:

2 → 3
3 → 4
4 → 5
Open doors
Wait
Close doors

While one elevator is moving or operating its doors, other elevators
continue operating independently and new requests may continue arriving.

Elevator Direction

An elevator may currently be:

Moving upward

Moving downward

Idle

An elevator can have multiple scheduled stops while additional requests
arrive.

For example:

Elevator: E1
Current Floor: 4
Direction: UP
Pending Stops: 6, 8, 10

If a passenger on Floor 5 requests an elevator going upward, the system
must decide whether E1 should serve that request or whether another
elevator is more appropriate.

Elevator Capacity

Every elevator has a maximum supported weight.

The system tracks the elevator's actual current load.

Example:

Maximum Capacity: 800 kg
Current Load:     620 kg

An elevator must not move while its actual measured load exceeds its
maximum capacity.

Passenger Weight Estimation

When a passenger requests an elevator from a floor, the system does not
know the passenger's actual weight.

For dispatch decisions, the system uses a configurable standard
estimated passenger weight.

For example:

Standard Estimated Passenger Weight: 100 kg

Consider an elevator with:

Maximum Capacity: 800 kg
Current Load:     690 kg

The elevator may be considered capable of accepting another passenger
because:

690 + 100 = 790 kg

The standard passenger weight is only an estimate used when deciding
which elevator should serve a floor request.

Actual Weight and Overload

Once a passenger enters an elevator, the actual measured weight may
differ from the estimated passenger weight used during dispatch.

For example:

Maximum Capacity:            800 kg
Load before passenger:       690 kg
Estimated passenger weight:  100 kg
Estimated total:             790 kg

Actual passenger weight:     140 kg
Actual total:                830 kg

The elevator is now overloaded.

An overloaded elevator must not begin moving.

Movement can resume only after the actual measured load is within the
elevator's configured maximum capacity.

Dispatch Strategies

The system must support multiple elevator dispatch strategies.

A dispatch strategy determines which elevator should serve a floor
request.

A strategy may consider information such as:

Current elevator floor

Requested floor

Requested direction

Current elevator direction

Existing pending stops

Current elevator workload

Current load

Maximum weight capacity

Estimated passenger weight

For example:

Request:
    Floor: 7
    Direction: UP

E1:
    Floor: 5
    Direction: UP
    Current Load: 700 / 800 kg

E2:
    Floor: 10
    Direction: IDLE
    Current Load: 200 / 800 kg

E3:
    Floor: 8
    Direction: DOWN
    Current Load: 400 / 800 kg

Different strategies may choose different elevators for the same
request.

The active dispatch strategy must be switchable while the elevator
system is running.

The exact dispatch strategies and the rules used by them are part of the
design exercise.

Concurrent Operation

The system must support multiple elevators operating concurrently.

At any moment, situations such as the following may occur:

Floor 3  → requests UP
Floor 8  → requests DOWN
Floor 14 → requests DOWN

E1 → moving
E2 → opening doors
E3 → processing existing stops

The system must remain correct when:

Multiple floor requests arrive concurrently

Multiple passengers select destinations concurrently

Multiple elevators move concurrently

Elevator doors operate concurrently

Passenger loads change

New requests arrive while elevators are moving

Requests arrive while doors are opening or closing

The active dispatch strategy changes while the system is running

Elevator State

The current state of an elevator must be observable.

The system should be able to provide information such as:

Elevator: E1
Current Floor: 8
Direction: UP
Current Load: 540 kg
Maximum Load: 800 kg
Pending Stops: 10, 12, 15

The system must support retrieving the state of an individual elevator
as well as the state of all elevators.

Shutdown

The elevator system must support clean shutdown.

Once shutdown begins:

New requests should no longer be accepted

Elevator operations should terminate safely

The system should not leave background operations running
indefinitely

Expected Operations

At a behavioral level, the system should support operations equivalent
to:

Request an elevator from a floor

Select a destination floor from inside an elevator

Update passenger load

Retrieve the state of an elevator

Retrieve the state of all elevators

Switch the active dispatch strategy

Shut down the elevator system

The exact APIs, data structures, interfaces, scheduling algorithms,
synchronization mechanisms, concurrency model, and internal architecture
are intentionally left as part of the Low-Level Design exercise.

---

## Implementation

### Files

| File | Contents |
|------|----------|
| `elevator.go` | Core types, `Elevator` and `SchedulingStrategy` interfaces, `Directional` and `FIFO` strategies, `ElevatorSystem` |
| `heap.go` | `MinHeap` / `MaxHeap` of requests ordered by floor (used by `Directional`) |
| `main.go` | Console demo that issues calls and prints elevator state every second |

### Run

```
go run .
```

### Core types

- `Request` — a stop: `Floor`, `Direction`, and `Type` (`EXTERNAL` hall call or `INTERNAL` cabin selection).
- `ElevatorState` — observable snapshot: `ID`, `CurrentFloor`, `Direction` (`UP` / `DOWN` / `IDLE`), `DoorState` (`OPEN` / `CLOSED`), and `Pending` stops.

### Interfaces

```go
type Elevator interface {
    Call(floor int, direction Direction) error // hall call from a floor
    SelectFloor(floor int) error               // destination from inside the cabin
    State() ElevatorState
}

type SchedulingStrategy interface {
    AddRequest(request Request)
    NextRequest(current ElevatorState) (Request, bool)
    Pending() []Request
}
```

The elevator only moves the car; which stop to serve next is delegated to the strategy.

### Scheduling strategies

**Directional (LOOK)** — the default.

- `up` is a min-heap: while sweeping up, the lowest floor at or above the car is served next.
- `down` is a max-heap: while sweeping down, the highest floor at or below the car is served next.
- Before picking a stop, `rebucket` moves any stop the car has already passed into the other heap, so a request is always judged against the car's live floor rather than where it was when the request arrived.
- The strategy keeps its own `sweep` direction and reverses only when nothing is left ahead.
- Duplicate requests are ignored while one is still pending.

Example — car at floor 5, pending 8, 3↑, 6, 2↓, 9; floor 7 selected mid-trip:

```
6 → 8 → 9 → 7 → 3 → 2
```

**FIFO** — serves requests strictly in arrival order. Kept as a simple baseline.

### Concurrency

- `Call` and `SelectFloor` validate the floor, hand the request to the strategy, and signal a buffered `workCh` (size 1, non-blocking send) so many callers can wake the processor without blocking.
- A single `processor` goroutine per elevator drains the strategy and calls `moveTo`, which advances one floor per `timePerFloorChange` seconds, opens the doors, waits `doorStayTime` seconds, then closes them.
- `ElevatorSystem.mu` (RWMutex) guards elevator state; each strategy has its own mutex guarding its queues.
- `Shutdown` closes the `shutdown` channel; the processor and any in-progress move or door wait exit promptly.

### Not yet implemented

Compared with the problem statement above:

- Only one elevator — there is no dispatcher assigning hall calls across multiple elevators.
- No passenger load, capacity, estimated weight, or overload handling.
- The strategy is fixed at construction; it cannot be switched at runtime.
- `moveTo` travels straight to the chosen stop, so a request for a floor the car is passing mid-trip is not served until a later pass.
- Shutdown stops the processor but `Call` / `SelectFloor` do not yet reject requests after shutdown.
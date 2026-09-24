package main

import (
	"fmt"
	"sync"
	"time"

	"parking_management/models"
)

// --------------
// cursor required helpers — observability / API / simulation
// --------------

type ParkingObserver interface {
	OnParkAccepted(ticketID string, vehicle Vehicle, toFloor int)
	OnParkRejected(vehicle Vehicle, reason string)
	OnUnparkAccepted(ticketID string, vehicle Vehicle, physicalFloor int)
	OnUnparkRejected(ticketID string, reason string)
	OnStrategyChanged(strategy string)
	OnRearrangementPlanned(comps []ShuffleComponent)
	OnParkingStarted(robotID string, job ParkingJob)
	OnParkingCompleted(robotID string, job ParkingJob)
	OnUnparkingStarted(robotID string, ticketID string, vehicle Vehicle, fromFloor int)
	OnUnparkingCompleted(robotID string, ticketID string, vehicle Vehicle, fromFloor int)
	OnShuffleStarted(robotID string, job VehicleMove, vehicleReg string)
	OnShuffleCompleted(robotID string, job VehicleMove, vehicleReg string)
}

type FloorSnapshot struct {
	Number                 int
	Capacity               int
	PhysicallyUsedCapacity int
	ReservedCapacity       int
}

type VehicleSnapshot struct {
	TicketID        string
	CustomerName    string
	VehicleRegNo    string
	VehicleType     VehicleType
	ParkingStatus   ParkingStatus
	UnparkRequested bool
	PhysicalFloor   int
	TargetFloor     int
	EntryTime       time.Time
}

type SystemSnapshot struct {
	Strategy            string
	ShufflingInProgress bool
	Floors              []FloorSnapshot
	Vehicles            []VehicleSnapshot
}

func strategyName(strategy ParkingStrategy) string {
	switch strategy.(type) {
	case NearestFloorStrategy:
		return "NEAREST"
	case LoadBalancedStrategy:
		return "LOAD_BALANCED"
	case ZoneBalancedStrategy:
		return "ZONE_BALANCED"
	case CompactionStrategy:
		return "COMPACTION"
	default:
		return "UNKNOWN"
	}
}

func (p *AutomatedParkingSystem) Snapshot() SystemSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()

	floors := make([]FloorSnapshot, len(p.floors))
	for i, f := range p.floors {
		floors[i] = FloorSnapshot{
			Number:                 f.Number,
			Capacity:               f.Capacity,
			PhysicallyUsedCapacity: f.PhysicallyUsedCapacity,
			ReservedCapacity:       f.ReservedCapacity,
		}
	}
	vehicles := make([]VehicleSnapshot, 0, len(p.recordsByTktID))
	for _, rec := range p.recordsByTktID {
		vehicles = append(vehicles, VehicleSnapshot{
			TicketID:        rec.TicketID,
			CustomerName:    rec.CustomerName,
			VehicleRegNo:    rec.Vehicle.VehicleRegNo,
			VehicleType:     rec.Vehicle.Type,
			ParkingStatus:   rec.ParkingStatus,
			UnparkRequested: rec.UnparkRequested,
			PhysicalFloor:   rec.PhysicalFloorNumber,
			TargetFloor:     rec.TargetFloorNumber,
			EntryTime:       rec.EntryTime,
		})
	}
	return SystemSnapshot{
		Strategy:            strategyName(p.strategy),
		ShufflingInProgress: p.shufflingInProgress,
		Floors:              floors,
		Vehicles:            vehicles,
	}
}

const (
	robotStatusIdle = "IDLE"
	robotStatusBusy = "BUSY"

	robotTypeParking   = "PARKING"
	robotTypeUnparking = "UNPARKING"
	robotTypeShuffle   = "SHUFFLE"
)

type RobotState struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	TicketID     string `json:"ticketId,omitempty"`
	VehicleRegNo string `json:"vehicleRegNo,omitempty"`
	FromFloor    *int   `json:"fromFloor,omitempty"`
	ToFloor      *int   `json:"toFloor,omitempty"`
}

type StagingVehicle struct {
	TicketID     string      `json:"ticketId"`
	VehicleRegNo string      `json:"vehicleRegNo"`
	VehicleType  VehicleType `json:"vehicleType"`
	ToFloor      int         `json:"toFloor"`
}

type QueueVehicle struct {
	TicketID     string      `json:"ticketId"`
	VehicleRegNo string      `json:"vehicleRegNo"`
	VehicleType  VehicleType `json:"vehicleType"`
	FromFloor    *int        `json:"fromFloor,omitempty"`
	ToFloor      *int        `json:"toFloor,omitempty"`
}

type SystemEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type"`
	TicketID     string    `json:"ticketId,omitempty"`
	VehicleRegNo string    `json:"vehicleRegNo,omitempty"`
	RobotID      string    `json:"robotId,omitempty"`
	FromFloor    *int      `json:"fromFloor,omitempty"`
	ToFloor      *int      `json:"toFloor,omitempty"`
	Message      string    `json:"message,omitempty"`
}

type MetricsDTO struct {
	VehiclesArrived  int `json:"vehiclesArrived"`
	VehiclesAccepted int `json:"vehiclesAccepted"`
	VehiclesRejected int `json:"vehiclesRejected"`
	VehiclesExited   int `json:"vehiclesExited"`
	ActiveVehicles   int `json:"activeVehicles"`
}

type FloorDTO struct {
	FloorNumber            int          `json:"floorNumber"`
	Capacity               int          `json:"capacity"`
	ReservedCapacity       int          `json:"reservedCapacity"`
	PhysicallyUsedCapacity int          `json:"physicallyUsedCapacity"`
	Zone                   string       `json:"zone"`
	Vehicles               []VehicleDTO `json:"vehicles"`
}

type VehicleDTO struct {
	TicketID        string        `json:"ticketId"`
	VehicleRegNo    string        `json:"vehicleRegNo"`
	VehicleType     VehicleType   `json:"vehicleType"`
	CustomerName    string        `json:"customerName,omitempty"`
	ParkingStatus   ParkingStatus `json:"parkingStatus"`
	UnparkRequested bool          `json:"unparkRequested"`
	PhysicalFloor   int           `json:"physicalFloor"`
	TargetFloor     int           `json:"targetFloor"`
	EntryTime       time.Time     `json:"entryTime"`
}

type QueuesDTO struct {
	Staging           []StagingVehicle `json:"staging"`
	WaitingForUnpark  []QueueVehicle   `json:"waitingForUnpark"`
	WaitingForShuffle []QueueVehicle   `json:"waitingForShuffle"`
}

type RobotsDTO struct {
	Parking   []RobotState `json:"parking"`
	Unparking []RobotState `json:"unparking"`
	Shuffle   []RobotState `json:"shuffle"`
}

type StateDTO struct {
	Strategy            string       `json:"strategy"`
	ShufflingInProgress bool         `json:"shufflingInProgress"`
	Metrics             MetricsDTO   `json:"metrics"`
	Queues              QueuesDTO    `json:"queues"`
	Robots              RobotsDTO    `json:"robots"`
	Floors              []FloorDTO   `json:"floors"`
	Vehicles            []VehicleDTO `json:"vehicles"`
}

type observabilityStore struct {
	mu sync.Mutex

	robots          map[string]*RobotState
	stagingByTicket map[string]StagingVehicle

	vehiclesArrived  int
	vehiclesAccepted int
	vehiclesRejected int
	vehiclesExited   int

	events []SystemEvent
	cap    int
	next   int
	count  int
}

func newObservabilityStore(eventCap int) *observabilityStore {
	if eventCap <= 0 {
		eventCap = 200
	}
	return &observabilityStore{
		robots:          make(map[string]*RobotState),
		stagingByTicket: make(map[string]StagingVehicle),
		events:          make([]SystemEvent, eventCap),
		cap:             eventCap,
	}
}

func (o *observabilityStore) initRobots(cfg RobotPoolConfig) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for i := 1; i <= cfg.ParkingRobots; i++ {
		id := fmt.Sprintf("PARK-%d", i)
		o.robots[id] = &RobotState{ID: id, Type: robotTypeParking, Status: robotStatusIdle}
	}
	for i := 1; i <= cfg.UnparkingRobots; i++ {
		id := fmt.Sprintf("UNPARK-%d", i)
		o.robots[id] = &RobotState{ID: id, Type: robotTypeUnparking, Status: robotStatusIdle}
	}
	for i := 1; i <= cfg.ShuffleRobots; i++ {
		id := fmt.Sprintf("SHUFFLE-%d", i)
		o.robots[id] = &RobotState{ID: id, Type: robotTypeShuffle, Status: robotStatusIdle}
	}
}

func (o *observabilityStore) appendEventLocked(ev SystemEvent) {
	ev.Timestamp = time.Now()
	o.events[o.next] = ev
	o.next = (o.next + 1) % o.cap
	if o.count < o.cap {
		o.count++
	}
}

func intPtr(v int) *int { return &v }

func (o *observabilityStore) setRobotBusyLocked(id, ticketID, reg string, from, to *int) {
	r, ok := o.robots[id]
	if !ok {
		return
	}
	r.Status = robotStatusBusy
	r.TicketID = ticketID
	r.VehicleRegNo = reg
	r.FromFloor = from
	r.ToFloor = to
}

func (o *observabilityStore) setRobotIdleLocked(id string) {
	r, ok := o.robots[id]
	if !ok {
		return
	}
	r.Status = robotStatusIdle
	r.TicketID = ""
	r.VehicleRegNo = ""
	r.FromFloor = nil
	r.ToFloor = nil
}

func (o *observabilityStore) OnParkAccepted(ticketID string, vehicle Vehicle, toFloor int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.vehiclesAccepted++
	o.stagingByTicket[ticketID] = StagingVehicle{
		TicketID: ticketID, VehicleRegNo: vehicle.VehicleRegNo, VehicleType: vehicle.Type, ToFloor: toFloor,
	}
	o.appendEventLocked(SystemEvent{
		Type: "PARK_ACCEPTED", TicketID: ticketID, VehicleRegNo: vehicle.VehicleRegNo, ToFloor: intPtr(toFloor),
	})
}

func (o *observabilityStore) OnParkRejected(vehicle Vehicle, reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.vehiclesRejected++
	o.appendEventLocked(SystemEvent{Type: "PARK_REJECTED", VehicleRegNo: vehicle.VehicleRegNo, Message: reason})
}

func (o *observabilityStore) OnUnparkAccepted(ticketID string, vehicle Vehicle, physicalFloor int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.appendEventLocked(SystemEvent{
		Type: "UNPARK_REQUESTED", TicketID: ticketID, VehicleRegNo: vehicle.VehicleRegNo, FromFloor: intPtr(physicalFloor),
	})
}

func (o *observabilityStore) OnUnparkRejected(ticketID string, reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.appendEventLocked(SystemEvent{Type: "UNPARK_REJECTED", TicketID: ticketID, Message: reason})
}

func (o *observabilityStore) OnStrategyChanged(strategy string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.appendEventLocked(SystemEvent{Type: "STRATEGY_CHANGED", Message: strategy})
}

func (o *observabilityStore) OnRearrangementPlanned(comps []ShuffleComponent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	n := 0
	for _, c := range comps {
		n += len(c.Moves)
	}
	o.appendEventLocked(SystemEvent{Type: "REARRANGEMENT_PLANNED", Message: fmt.Sprintf("%d moves in %d components", n, len(comps))})
}

func (o *observabilityStore) OnParkingStarted(robotID string, job ParkingJob) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.stagingByTicket, job.TicketID)
	o.setRobotBusyLocked(robotID, job.TicketID, job.Vehicle.VehicleRegNo, nil, intPtr(job.ToFloorNumber))
	o.appendEventLocked(SystemEvent{
		Type: "PARKING_STARTED", TicketID: job.TicketID, VehicleRegNo: job.Vehicle.VehicleRegNo,
		RobotID: robotID, ToFloor: intPtr(job.ToFloorNumber),
	})
}

func (o *observabilityStore) OnParkingCompleted(robotID string, job ParkingJob) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.setRobotIdleLocked(robotID)
	o.appendEventLocked(SystemEvent{
		Type: "PARKING_COMPLETED", TicketID: job.TicketID, VehicleRegNo: job.Vehicle.VehicleRegNo,
		RobotID: robotID, ToFloor: intPtr(job.ToFloorNumber),
	})
}

func (o *observabilityStore) OnUnparkingStarted(robotID string, ticketID string, vehicle Vehicle, fromFloor int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.setRobotBusyLocked(robotID, ticketID, vehicle.VehicleRegNo, intPtr(fromFloor), nil)
	o.appendEventLocked(SystemEvent{
		Type: "UNPARK_STARTED", TicketID: ticketID, VehicleRegNo: vehicle.VehicleRegNo,
		RobotID: robotID, FromFloor: intPtr(fromFloor),
	})
}

func (o *observabilityStore) OnUnparkingCompleted(robotID string, ticketID string, vehicle Vehicle, fromFloor int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.vehiclesExited++
	o.setRobotIdleLocked(robotID)
	o.appendEventLocked(SystemEvent{
		Type: "UNPARK_COMPLETED", TicketID: ticketID, VehicleRegNo: vehicle.VehicleRegNo,
		RobotID: robotID, FromFloor: intPtr(fromFloor),
	})
}

func (o *observabilityStore) OnShuffleStarted(robotID string, job VehicleMove, vehicleReg string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.setRobotBusyLocked(robotID, job.TicketID, vehicleReg, intPtr(job.FromFloorNumber), intPtr(job.ToFloorNumber))
	o.appendEventLocked(SystemEvent{
		Type: "SHUFFLE_STARTED", TicketID: job.TicketID, VehicleRegNo: vehicleReg, RobotID: robotID,
		FromFloor: intPtr(job.FromFloorNumber), ToFloor: intPtr(job.ToFloorNumber),
	})
}

func (o *observabilityStore) OnShuffleCompleted(robotID string, job VehicleMove, vehicleReg string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.setRobotIdleLocked(robotID)
	o.appendEventLocked(SystemEvent{
		Type: "SHUFFLE_COMPLETED", TicketID: job.TicketID, VehicleRegNo: vehicleReg, RobotID: robotID,
		FromFloor: intPtr(job.FromFloorNumber), ToFloor: intPtr(job.ToFloorNumber),
	})
}

func (o *observabilityStore) recordArrival(vehicle Vehicle) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.vehiclesArrived++
	o.appendEventLocked(SystemEvent{Type: "VEHICLE_ARRIVED", VehicleRegNo: vehicle.VehicleRegNo, Message: string(vehicle.Type)})
}

func (o *observabilityStore) Events() []SystemEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]SystemEvent, 0, o.count)
	if o.count == 0 {
		return out
	}
	start := 0
	if o.count == o.cap {
		start = o.next
	}
	for i := 0; i < o.count; i++ {
		out = append(out, o.events[(start+i)%o.cap])
	}
	return out
}

func floorZoneLabel(floorCount, floorNumber int) string {
	truckEnd := floorCount * models.TruckZonePercent / 100
	bikeStart := floorCount - (floorCount * models.BikeZonePercent / 100)
	if floorNumber < truckEnd {
		return "TRUCK"
	}
	if floorNumber >= bikeStart {
		return "BIKE"
	}
	return "CAR"
}

func (o *observabilityStore) buildState(snap SystemSnapshot) StateDTO {
	o.mu.Lock()
	defer o.mu.Unlock()

	busyUnpark := map[string]bool{}
	busyShuffle := map[string]bool{}
	parking, unparking, shuffle := []RobotState{}, []RobotState{}, []RobotState{}
	for _, r := range o.robots {
		cp := *r
		if cp.FromFloor != nil {
			v := *cp.FromFloor
			cp.FromFloor = &v
		}
		if cp.ToFloor != nil {
			v := *cp.ToFloor
			cp.ToFloor = &v
		}
		switch cp.Type {
		case robotTypeParking:
			parking = append(parking, cp)
		case robotTypeUnparking:
			unparking = append(unparking, cp)
			if cp.Status == robotStatusBusy {
				busyUnpark[cp.TicketID] = true
			}
		case robotTypeShuffle:
			shuffle = append(shuffle, cp)
			if cp.Status == robotStatusBusy {
				busyShuffle[cp.TicketID] = true
			}
		}
	}

	staging := make([]StagingVehicle, 0, len(o.stagingByTicket))
	for _, s := range o.stagingByTicket {
		staging = append(staging, s)
	}

	vehicles := make([]VehicleDTO, 0, len(snap.Vehicles))
	waitingUnpark := []QueueVehicle{}
	waitingShuffle := []QueueVehicle{}
	floorVehicles := map[int][]VehicleDTO{}

	for _, v := range snap.Vehicles {
		dto := VehicleDTO{
			TicketID: v.TicketID, VehicleRegNo: v.VehicleRegNo, VehicleType: v.VehicleType,
			CustomerName: v.CustomerName, ParkingStatus: v.ParkingStatus, UnparkRequested: v.UnparkRequested,
			PhysicalFloor: v.PhysicalFloor, TargetFloor: v.TargetFloor, EntryTime: v.EntryTime,
		}
		vehicles = append(vehicles, dto)
		if v.PhysicalFloor >= 0 {
			floorVehicles[v.PhysicalFloor] = append(floorVehicles[v.PhysicalFloor], dto)
		}
		if v.UnparkRequested && !busyUnpark[v.TicketID] {
			from := v.PhysicalFloor
			waitingUnpark = append(waitingUnpark, QueueVehicle{
				TicketID: v.TicketID, VehicleRegNo: v.VehicleRegNo, VehicleType: v.VehicleType, FromFloor: &from,
			})
		}
		if !v.UnparkRequested && v.PhysicalFloor >= 0 && v.PhysicalFloor != v.TargetFloor && !busyShuffle[v.TicketID] {
			from, to := v.PhysicalFloor, v.TargetFloor
			waitingShuffle = append(waitingShuffle, QueueVehicle{
				TicketID: v.TicketID, VehicleRegNo: v.VehicleRegNo, VehicleType: v.VehicleType,
				FromFloor: &from, ToFloor: &to,
			})
		}
	}

	floors := make([]FloorDTO, len(snap.Floors))
	n := len(snap.Floors)
	for i, f := range snap.Floors {
		vs := floorVehicles[f.Number]
		if vs == nil {
			vs = []VehicleDTO{}
		}
		floors[i] = FloorDTO{
			FloorNumber: f.Number, Capacity: f.Capacity, ReservedCapacity: f.ReservedCapacity,
			PhysicallyUsedCapacity: f.PhysicallyUsedCapacity, Zone: floorZoneLabel(n, f.Number), Vehicles: vs,
		}
	}

	return StateDTO{
		Strategy:            snap.Strategy,
		ShufflingInProgress: snap.ShufflingInProgress,
		Metrics: MetricsDTO{
			VehiclesArrived: o.vehiclesArrived, VehiclesAccepted: o.vehiclesAccepted,
			VehiclesRejected: o.vehiclesRejected, VehiclesExited: o.vehiclesExited,
			ActiveVehicles: len(snap.Vehicles),
		},
		Queues: QueuesDTO{Staging: staging, WaitingForUnpark: waitingUnpark, WaitingForShuffle: waitingShuffle},
		Robots: RobotsDTO{Parking: parking, Unparking: unparking, Shuffle: shuffle},
		Floors: floors, Vehicles: vehicles,
	}
}

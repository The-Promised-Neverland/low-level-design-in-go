package main

import (
	cryptorand "crypto/rand"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"
)

type AutomatedParkingSystem struct {
	floors              []*Floor
	recordsByTktID      map[string]*ParkingRecord
	recordsByRegNo      map[string]*ParkingRecord
	strategy            ParkingStrategy
	shufflingInProgress bool
	ShuffleJobWG        sync.WaitGroup
	parkingJobCh        chan ParkingJob
	unparkJobCh         chan UnparkJob
	shuffleJobCh        chan ShuffleComponent
	shuffleWake         *sync.Cond
	mu                  sync.Mutex
	observer            ParkingObserver
}

func NewAutomatedParkingSystem(floorsCnt int, floorSpace int, robotConfig RobotPoolConfig) *AutomatedParkingSystem {
	floors := make([]*Floor, floorsCnt)
	for i := range floorsCnt {
		floors[i] = &Floor{
			Number:   i,
			Capacity: floorSpace,
		}
	}
	aps := &AutomatedParkingSystem{
		floors:              floors,
		recordsByTktID:      make(map[string]*ParkingRecord),
		recordsByRegNo:      make(map[string]*ParkingRecord),
		shufflingInProgress: false,
		strategy:            NearestFloorStrategy{},
		parkingJobCh:        make(chan ParkingJob, 1000),
		unparkJobCh:         make(chan UnparkJob, 1000),
		shuffleJobCh:        make(chan ShuffleComponent, 1000),
	}
	aps.shuffleWake = sync.NewCond(&aps.mu)
	for _, floor := range aps.floors {
		floor.Cond = sync.NewCond(&aps.mu)
	}
	aps.spinupParkingRobotsPool(robotConfig.ParkingRobots)
	aps.spinupUnparkingRobotsPool(robotConfig.UnparkingRobots)
	aps.spinupShuffleRobotsPool(robotConfig.ShuffleRobots)
	return aps
}

func (p *AutomatedParkingSystem) Park(vehicle Vehicle, custName string) (*Ticket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	decision := p.strategy.Allocate(vehicle, p.floors)
	if decision == nil {
		if p.observer != nil {
			p.observer.OnParkRejected(vehicle, "Cannot park vehicle")
		}
		return nil, fmt.Errorf("Cannot park vehicle")
	}
	floorNumber := decision.FloorNumber
	entryTime := time.Now()
	ticketID := generateTicketID()
	ticket := &Ticket{
		TicketID:       ticketID,
		VehicleDetails: vehicle,
		EntryTime:      entryTime,
	}
	parkingJob := ParkingJob{
		TicketID:      ticketID,
		Vehicle:       vehicle,
		ToFloorNumber: floorNumber,
	}
	select {
	case p.parkingJobCh <- parkingJob:
		parkingRecord := &ParkingRecord{
			TicketID:            ticketID,
			CustomerName:        custName,
			Vehicle:             vehicle,
			PhysicalFloorNumber: -1,
			TargetFloorNumber:   floorNumber,
			EntryTime:           entryTime,
			ParkingStatus:       ParkingPending,
		}
		p.recordsByRegNo[vehicle.VehicleRegNo] = parkingRecord
		p.recordsByTktID[ticketID] = parkingRecord
		p.floors[floorNumber].ReservedCapacity += getVehicleSpace(vehicle.Type)
		if p.observer != nil {
			p.observer.OnParkAccepted(ticketID, vehicle, floorNumber)
		}
		return ticket, nil
	default:
		if p.observer != nil {
			p.observer.OnParkRejected(vehicle, "Staging area is filled up. Cannot park")
		}
		return nil, fmt.Errorf("Staging area is filled up. Cannot park")
	}
}

func (p *AutomatedParkingSystem) Unpark(ticketID string) (*Receipt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	parkingRecord, ok := p.recordsByTktID[ticketID]
	if !ok {
		if p.observer != nil {
			p.observer.OnUnparkRejected(ticketID, "Invalid Ticket")
		}
		return nil, fmt.Errorf("Invalid Ticket")
	}
	if parkingRecord.UnparkRequested {
		if p.observer != nil {
			p.observer.OnUnparkRejected(ticketID, "unpark already requested")
		}
		return nil, fmt.Errorf("unpark already requested")
	}
	exitTime := time.Now()
	vehicleType := parkingRecord.Vehicle.Type
	duration := exitTime.Sub(parkingRecord.EntryTime)
	hours := math.Ceil(duration.Hours())
	if hours < 1 {
		hours = 1
	}
	parkingRecord.UnparkRequested = true
	p.shuffleWake.Broadcast()
	if parkingRecord.ParkingStatus == Parked {
		select {
		case p.unparkJobCh <- UnparkJob{
			TicketID: ticketID,
		}:
		default:
			parkingRecord.UnparkRequested = false
			if p.observer != nil {
				p.observer.OnUnparkRejected(ticketID, "unparking is busy. Please wait and try again after some time")
			}
			return nil, fmt.Errorf("unparking is busy. Please wait and try again after some time")
		}
	}
	totalAmount := getVehicleRatePerHour(vehicleType) * hours
	p.floors[parkingRecord.TargetFloorNumber].ReservedCapacity -= getVehicleSpace(vehicleType)
	if p.observer != nil {
		p.observer.OnUnparkAccepted(ticketID, parkingRecord.Vehicle, parkingRecord.PhysicalFloorNumber)
	}
	return &Receipt{
		TicketID:       ticketID,
		VehicleDetails: parkingRecord.Vehicle,
		EntryAt:        parkingRecord.EntryTime,
		ExitAt:         exitTime,
		Amount:         totalAmount,
	}, nil
}

func (p *AutomatedParkingSystem) SwitchStrategy(strategy ParkingStrategy) error {
	p.mu.Lock()
	if p.shufflingInProgress {
		p.mu.Unlock()
		return fmt.Errorf("shuffling in progress")
	}
	var comps []ShuffleComponent
	if rearrangementStrategy, ok := strategy.(FloorsRearrangementStrategy); ok {
		plan := rearrangementStrategy.PlanRearrangement(p.floors, p.recordsByTktID)
		if plan == nil {
			p.mu.Unlock()
			return fmt.Errorf("rearrangement blueprint invalid")
		}
		order := BuildMovementOrder(plan.Moves)
		comps = p.commitBlueprint(order)
	}
	p.strategy = strategy
	name := strategyName(strategy)
	if len(comps) > 0 {
		p.shufflingInProgress = true
	}
	p.mu.Unlock()
	if p.observer != nil {
		p.observer.OnStrategyChanged(name)
	}
	if len(comps) == 0 {
		return nil
	}
	if p.observer != nil {
		p.observer.OnRearrangementPlanned(comps)
	}
	p.ShuffleJobWG.Add(len(comps))
	go func() {
		for _, comp := range comps {
			p.shuffleJobCh <- comp
		}
		p.ShuffleJobWG.Wait()
		p.mu.Lock()
		p.shufflingInProgress = false
		p.mu.Unlock()
	}()
	return nil
}

func (p *AutomatedParkingSystem) commitBlueprint(order MovementOrder) []ShuffleComponent {
	comps := make([]ShuffleComponent, 0)
	for i, component := range order.Moves {
		moves := make([]VehicleMove, 0)
		loop := i < len(order.Loop) && order.Loop[i]
		for _, move := range component {
			record, ok := p.recordsByTktID[move.TicketID]
			if !ok || record.UnparkRequested {
				continue
			}
			requiredSpace := getVehicleSpace(record.Vehicle.Type)
			oldTarget := record.TargetFloorNumber
			newTarget := move.ToFloorNumber
			if oldTarget != newTarget {
				p.floors[oldTarget].ReservedCapacity -= requiredSpace
				p.floors[newTarget].ReservedCapacity += requiredSpace
				record.TargetFloorNumber = newTarget
			}
			if record.PhysicalFloorNumber >= 0 && record.PhysicalFloorNumber == newTarget {
				continue
			}
			from := record.PhysicalFloorNumber
			if from < 0 {
				from = move.FromFloorNumber
			}
			if from == newTarget {
				continue
			}
			moves = append(moves, VehicleMove{
				TicketID:        record.TicketID,
				FromFloorNumber: from,
				ToFloorNumber:   newTarget,
			})
		}
		if len(moves) > 0 {
			comps = append(comps, ShuffleComponent{Moves: moves, Loop: loop})
		}
	}
	return comps
}

// Format for ticket - TKT-DDMMYY-5 DIGITS RANDOM
func generateTicketID() string {
	date := time.Now().Format("020106")
	var randomDigits string
	for i := 0; i < 5; i++ {
		n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(10))
		if err != nil {
			i--
			continue
		}
		randomDigits += n.String()
	}
	return fmt.Sprintf("TKT-%s-%s", date, randomDigits)
}

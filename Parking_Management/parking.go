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
	recordsByTktID      map[string]*ParkingRecord // do not delete for 7 days. Maintain the database
	recordsByRegNo      map[string]*ParkingRecord // do not delete for 7 days. Maintain the database
	strategy            ParkingStrategy
	shufflingInProgress bool
	ShuffleJobWG        sync.WaitGroup
	parkingJobCh        chan ParkingJob
	unparkJobCh         chan UnparkJob
	shuffleJobCh        chan ShuffleJob
	rearrangeCh         chan struct{}
	mu                  sync.Mutex
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
		rearrangeCh:         make(chan struct{}, 1),
		parkingJobCh:        make(chan ParkingJob, 1000),
		unparkJobCh:         make(chan UnparkJob, 1000),
		shuffleJobCh:        make(chan ShuffleJob, 1000),
	}
	for _, floor := range aps.floors {
		floor.Cond = sync.NewCond(&aps.mu)
	}
	aps.spinupParkingRobotsPool(robotConfig.ParkingRobots)
	aps.spinupUnparkingRobotsPool(robotConfig.UnparkingRobots)
	aps.spinupShuffleRobotsPool(robotConfig.ShuffleRobots)
	go aps.rearrange()
	return aps
}

func (p *AutomatedParkingSystem) Park(vehicle Vehicle, custName string) (*Ticket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	decision := p.strategy.Allocate(vehicle, p.floors)
	if decision == nil {
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
		return ticket, nil
	default:
		return nil, fmt.Errorf("Staging area is filled up. Cannot park")
	}
}

func (p *AutomatedParkingSystem) Unpark(ticketID string) (*Receipt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	parkingRecord, ok := p.recordsByTktID[ticketID]
	if !ok {
		return nil, fmt.Errorf("Invalid Ticket")
	}
	if parkingRecord.UnparkRequested {
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
	if parkingRecord.ParkingStatus == Parked {
		select {
		case p.unparkJobCh <- UnparkJob{
			TicketID: ticketID,
		}:
		default:
			parkingRecord.UnparkRequested = false
			return nil, fmt.Errorf("unparking is busy. Please wait and try again after some time")
		}
	}
	totalAmount := getVehicleRatePerHour(vehicleType) * hours
	p.floors[parkingRecord.TargetFloorNumber].ReservedCapacity -= getVehicleSpace(vehicleType)
	return &Receipt{
		TicketID:       ticketID,
		VehicleDetails: parkingRecord.Vehicle,
		EntryAt:        parkingRecord.EntryTime,
		ExitAt:         exitTime,
		Amount:         totalAmount,
	}, nil
}

func (p *AutomatedParkingSystem) SwitchStrategy(strategy ParkingStrategy) {
	p.mu.Lock()
	if p.shufflingInProgress {
		p.mu.Unlock()
		return
	}
	p.strategy = strategy
	p.shufflingInProgress = true
	p.mu.Unlock()
	select {
	case p.rearrangeCh <- struct{}{}:
	default:
	}
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

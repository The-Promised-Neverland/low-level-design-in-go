package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"
)

type ParkingStatus string

const RobotParkingTime = 5 * time.Second

const (
	ParkingPending ParkingStatus = "PARKING_PENDING"
	Parked         ParkingStatus = "PARKED"
	Rearranging    ParkingStatus = "REARRANGING"
)

const (
	BikeHourlyPrice  float64 = 20
	CarHourlyPrice   float64 = 40
	TruckHourlyPrice float64 = 60
)

type VehicleType string

const (
	Bike  VehicleType = "BIKE"
	Car   VehicleType = "CAR"
	Truck VehicleType = "TRUCK"
)

type Vehicle struct {
	VehicleRegNo string
	Type         VehicleType
}

type SpaceUnit int

const (
	BikeSpace  SpaceUnit = 1
	CarSpace   SpaceUnit = 2
	TruckSpace SpaceUnit = 6
)

type Floor struct {
	Number                 int
	Capacity               int
	PhysicallyUsedCapacity int
	ReservedCapacity       int
	cond                   *sync.Cond
}

type Ticket struct {
	TicketID       string
	VehicleDetails Vehicle
	EntryTime      time.Time
}

type ParkingRecord struct {
	TicketID            string
	CustomerName        string // customer name is abstracted from ticket for privacy
	Vehicle             Vehicle
	PhysicalFloorNumber int
	TargetFloorNumber   int
	EntryTime           time.Time
	ParkingStatus       ParkingStatus
	UnparkRequested     bool
}

type Receipt struct {
	TicketID       string
	VehicleDetails Vehicle
	EntryAt        time.Time
	ExitAt         time.Time
	Amount         float64
}

type AllocationDecision struct {
	FloorNumber int
}

type ParkingStrategy interface {
	Allocate(vehicle Vehicle, floors []*Floor) *AllocationDecision
}

type NearestFloorStrategy struct{}

func (s NearestFloorStrategy) Allocate(vehicle Vehicle, floors []*Floor) *AllocationDecision {
	requiredSpace := getVehicleSpace(vehicle.Type)
	if requiredSpace == 0 {
		return nil
	}
	for _, floor := range floors {
		if requiredSpace > (floor.Capacity - floor.ReservedCapacity) {
			continue
		}
		return &AllocationDecision{
			FloorNumber: floor.Number,
		}
	}
	return nil
}

type LoadBalancedStrategy struct{}

func (s LoadBalancedStrategy) Allocate(vehicle Vehicle, floors []*Floor) *AllocationDecision {
	requiredSpace := getVehicleSpace(vehicle.Type)
	if requiredSpace == 0 {
		return nil
	}
	var selectedFloor *Floor
	for _, floor := range floors {
		if requiredSpace > (floor.Capacity - floor.ReservedCapacity) {
			continue
		}
		if selectedFloor == nil || selectedFloor.ReservedCapacity > floor.ReservedCapacity {
			selectedFloor = floor
		}
	}
	if selectedFloor == nil {
		return nil
	}
	return &AllocationDecision{
		FloorNumber: selectedFloor.Number,
	}
}

func (s LoadBalancedStrategy) PlanRearrangement(floors []*Floor, records map[string]*ParkingRecord) *ShufflePlan {
	plan := &ShufflePlan{
		Moves: make([]VehicleMove, 0),
	}
	if len(floors) == 0 {
		return plan
	}
	targetPlan := make([]int, len(floors))
	for _, record := range records {
		if record.UnparkRequested {  // do not plan for vehicles scheduled for unparking
			continue
		}
		requiredSpace := getVehicleSpace(record.Vehicle.Type)
		if requiredSpace == 0 {
			continue
		}
		selectedFloor := -1
		for i, floor := range floors {
			if targetPlan[i]+requiredSpace > floor.Capacity { // we cannot fit this vehicle here
				continue
			}
			if selectedFloor == -1 || targetPlan[i] < targetPlan[selectedFloor] {
				selectedFloor = i
			}
		}
		if selectedFloor == -1 {
			continue
		}
		targetPlan[selectedFloor] += requiredSpace
		if record.TargetFloorNumber != selectedFloor {
			plan.Moves = append(plan.Moves, VehicleMove{
				TicketID:      record.TicketID,
				ToFloorNumber: selectedFloor,
			})
		}
	}
	return plan
}

type VehicleMove struct {
	TicketID      string
	ToFloorNumber int
}

type ShufflePlan struct {
	Moves []VehicleMove
}

type FloorsRearrangementStrategy interface {
	PlanRearrangement(floors []*Floor, records map[string]*ParkingRecord) *ShufflePlan
}

type ParkingSystem interface {
	Park(vehicle Vehicle, custName string) (*Ticket, error)
	Unpark(ticketID string) (*Receipt, error)
	SwitchStrategy(strategy ParkingStrategy)
}

type ParkingJob struct {
	TicketID      string
	Vehicle       Vehicle
	ToFloorNumber int
}

type RobotPoolConfig struct {
	ParkingRobots   int
	UnparkingRobots int
	ShuffleRobots   int
}

type AutomatedParkingSystem struct {
	floors         []*Floor
	recordsByTktID map[string]*ParkingRecord // do not delete for 7 days. Maintain the database
	recordsByRegNo map[string]*ParkingRecord // do not delete for 7 days. Maintain the database
	strategy       ParkingStrategy
	parkingJobCh   chan ParkingJob
	unparkJobCh    chan UnparkJob
	shuffleJobCh   chan ShuffleJob
	rearrangeCh    chan struct{}
	mu             sync.Mutex
}

type UnparkJob struct {
	TicketID string
}

func NewAutomatedParkingSystem(floorsCnt int, floorSpace int, robotConfig RobotPoolConfig) ParkingSystem {
	floors := make([]*Floor, floorsCnt)
	for i := range floorsCnt {
		floors[i] = &Floor{
			Number:   i,
			Capacity: floorSpace,
		}
	}
	aps := &AutomatedParkingSystem{
		floors:         floors,
		recordsByTktID: make(map[string]*ParkingRecord),
		recordsByRegNo: make(map[string]*ParkingRecord),
		strategy:       NearestFloorStrategy{},
		rearrangeCh:    make(chan struct{}, 1),
		parkingJobCh:   make(chan ParkingJob, 1000),
		unparkJobCh:    make(chan UnparkJob, 1000),
		shuffleJobCh:   make(chan ShuffleJob, 1000),
	}
	for _, floor := range aps.floors {
		floor.cond = sync.NewCond(&aps.mu)
	}
	aps.spinupParkingRobotsPool(robotConfig.ParkingRobots)
	aps.spinupUnparkingRobotsPool(robotConfig.UnparkingRobots)
	aps.spinupShuffleRobotsPool(robotConfig.ShuffleRobots)
	go aps.rearrange()
	return aps
}

func (p *AutomatedParkingSystem) spinupUnparkingRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		go p.unparkers()
	}
}

func (p *AutomatedParkingSystem) spinupParkingRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		go p.parkers()
	}
}

func (p *AutomatedParkingSystem) spinupShuffleRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		go p.shuffleRobot()
	}
}

func (p *AutomatedParkingSystem) shuffleRobot() {
	for job := range p.shuffleJobCh {
		p.mu.Lock()
		record, ok := p.recordsByTktID[job.TicketID]
		if !ok {
			p.mu.Unlock()
			continue
		}
		if record.UnparkRequested == true { //  no need to shuffle. its schedule for unpark
			p.mu.Unlock()
			continue
		}
		if record.TargetFloorNumber != job.ToFloorNumber {
			p.mu.Unlock()
			continue
		}
		if record.PhysicalFloorNumber != job.FromFloorNumber {
			p.mu.Unlock()
			continue
		}
		requiredSpace := getVehicleSpace(record.Vehicle.Type)
		fromFloor := p.floors[job.FromFloorNumber]
		toFloor := p.floors[job.ToFloorNumber]
		stale := false
		for toFloor.Capacity-toFloor.PhysicallyUsedCapacity < requiredSpace {
			toFloor.cond.Wait()
			if record.TargetFloorNumber != job.ToFloorNumber || record.UnparkRequested {
				stale = true
				break
			}
		}
		if stale {
			p.mu.Unlock()
			continue
		}
		toFloor.PhysicallyUsedCapacity += requiredSpace
		record.ParkingStatus = Rearranging // it has now been handed to the shuffler bots
		p.mu.Unlock()
		time.Sleep(RobotParkingTime)
		p.mu.Lock()
		record, ok = p.recordsByTktID[job.TicketID]
		if !ok {
			toFloor.PhysicallyUsedCapacity -= requiredSpace
			toFloor.cond.Broadcast()
			p.mu.Unlock()
			continue
		}
		fromFloor.PhysicallyUsedCapacity -= requiredSpace
		record.PhysicalFloorNumber = job.ToFloorNumber
		record.ParkingStatus = Parked
		fromFloor.cond.Broadcast()
		shouldUnpark := record.UnparkRequested
		p.mu.Unlock()
		if shouldUnpark {
			p.unparkJobCh <- UnparkJob{
				TicketID: job.TicketID,
			}
		}
	}
}

type ShuffleJob struct {
	TicketID        string
	FromFloorNumber int
	ToFloorNumber   int
}

func (p *AutomatedParkingSystem) rearrange() {
	for range p.rearrangeCh {
		p.mu.Lock()
		rearrangementStrategy, ok := p.strategy.(FloorsRearrangementStrategy)
		if !ok {
			p.mu.Unlock()
			continue
		}
		plan := rearrangementStrategy.PlanRearrangement(p.floors, p.recordsByTktID)
		jobs := make([]ShuffleJob, 0, len(plan.Moves))
		for _, move := range plan.Moves {
			record, ok := p.recordsByTktID[move.TicketID]
			if !ok {
				continue
			}
			if record.UnparkRequested { // no need to rearrange the vehicles scheduled for unparking
				continue
			}
			requiredSpace := getVehicleSpace(record.Vehicle.Type)
			oldTarget := record.TargetFloorNumber
			newTarget := move.ToFloorNumber
			if oldTarget == newTarget {
				continue
			}
			p.floors[oldTarget].ReservedCapacity -= requiredSpace
			p.floors[newTarget].ReservedCapacity += requiredSpace
			record.TargetFloorNumber = newTarget
			if record.PhysicalFloorNumber != -1 && record.PhysicalFloorNumber != newTarget {
				jobs = append(jobs, ShuffleJob{
					TicketID:        record.TicketID,
					FromFloorNumber: record.PhysicalFloorNumber,
					ToFloorNumber:   newTarget,
				})
			}
		}
		p.mu.Unlock()
		for _, job := range jobs {
			p.shuffleJobCh <- job
		}
	}
}

func (p *AutomatedParkingSystem) parkers() {
	for job := range p.parkingJobCh {
		requiredSpace := getVehicleSpace(job.Vehicle.Type)
		p.mu.Lock()
		floor := p.floors[job.ToFloorNumber]
		for floor.Capacity-floor.PhysicallyUsedCapacity < requiredSpace {
			floor.cond.Wait()
		}
		floor.PhysicallyUsedCapacity += requiredSpace
		p.mu.Unlock()
		time.Sleep(RobotParkingTime)
		p.mu.Lock()
		shouldUnpark := false
		if record, ok := p.recordsByTktID[job.TicketID]; ok {
			record.PhysicalFloorNumber = job.ToFloorNumber
			record.ParkingStatus = Parked // mark as parked
			if record.UnparkRequested {   // customer reqeusted unparking
				shouldUnpark = true
			}
		}
		p.mu.Unlock()
		if shouldUnpark {
			p.unparkJobCh <- UnparkJob{ // schedule for unpark
				TicketID: job.TicketID,
			}
		}
	}
}

func (p *AutomatedParkingSystem) unparkers() {
	for job := range p.unparkJobCh {
		p.mu.Lock()
		record, ok := p.recordsByTktID[job.TicketID]
		if !ok {
			p.mu.Unlock()
			continue
		}
		if !record.UnparkRequested {
			p.mu.Unlock()
			continue
		}
		fromFloorNumber := record.PhysicalFloorNumber
		requiredSpace := getVehicleSpace(record.Vehicle.Type)
		p.mu.Unlock()
		time.Sleep(RobotParkingTime)
		p.mu.Lock()
		fromFloor := p.floors[fromFloorNumber]
		fromFloor.PhysicallyUsedCapacity -= requiredSpace
		fromFloor.cond.Broadcast()
		record.PhysicalFloorNumber = -1
		delete(p.recordsByTktID, record.TicketID)
		delete(p.recordsByRegNo, record.Vehicle.VehicleRegNo)
		p.mu.Unlock()
	}
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
	p.strategy = strategy
	p.mu.Unlock()
	select {
	case p.rearrangeCh <- struct{}{}:
	default:
	}
}

func getVehicleSpace(vehicleType VehicleType) int {
	switch vehicleType {
	case Bike:
		return int(BikeSpace)
	case Car:
		return int(CarSpace)
	case Truck:
		return int(TruckSpace)
	default:
		return 0
	}
}

func getVehicleRatePerHour(VehicleType VehicleType) float64 {
	switch VehicleType {
	case Bike:
		return BikeHourlyPrice
	case Car:
		return CarHourlyPrice
	case Truck:
		return TruckHourlyPrice
	default:
		return 0.0
	}
}

// Format for ticket - TKT-DDMMYY-5 DIGITS RANDOM
func generateTicketID() string {
	date := time.Now().Format("020106")
	var randomDigits string
	for i := 0; i < 5; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			i--
			continue
		}
		randomDigits += n.String()
	}
	return fmt.Sprintf("TKT-%s-%s", date, randomDigits)
}

func main() {

}

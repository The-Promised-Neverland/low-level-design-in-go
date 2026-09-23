package models

import (
	"sync"
	"time"
)

type ParkingStatus string

const RobotParkingTime = 15 * time.Second

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
	Cond                   *sync.Cond
}

type Ticket struct {
	TicketID             string
	VehicleDetails       Vehicle
	EntryTime            time.Time
}

type ParkingRecord struct {
	TicketID             string
	CustomerName         string // customer name is abstracted from ticket for privacy
	Vehicle              Vehicle
	PhysicalFloorNumber  int
	TargetFloorNumber    int
	EntryTime            time.Time
	ParkingStatus        ParkingStatus
	UnparkRequested      bool
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

type VehicleMove struct {
	TicketID      string
	ToFloorNumber int
}

type ShufflePlan struct {
	Moves []VehicleMove
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

type UnparkJob struct {
	TicketID string
}

type ShuffleJob struct {
	TicketID        string
	FromFloorNumber int
	ToFloorNumber   int
}

func GetVehicleSpace(vehicleType VehicleType) int {
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

func GetVehicleRatePerHour(vehicleType VehicleType) float64 {
	switch vehicleType {
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

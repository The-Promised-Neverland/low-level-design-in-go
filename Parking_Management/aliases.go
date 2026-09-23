package main

import (
	"parking_management/models"
	arrangement "parking_management/parking_rearrangement"
	strategy "parking_management/parking_strategy"
)

type ParkingStatus = models.ParkingStatus
type VehicleType = models.VehicleType
type Vehicle = models.Vehicle
type SpaceUnit = models.SpaceUnit
type Floor = models.Floor
type Ticket = models.Ticket
type ParkingRecord = models.ParkingRecord
type Receipt = models.Receipt
type AllocationDecision = models.AllocationDecision
type VehicleMove = models.VehicleMove
type ShufflePlan = models.ShufflePlan
type ParkingJob = models.ParkingJob
type RobotPoolConfig = models.RobotPoolConfig
type UnparkJob = models.UnparkJob
type ShuffleJob = models.ShuffleJob

type ParkingStrategy = models.ParkingStrategy
type FloorsRearrangementStrategy = models.FloorsRearrangementStrategy
type ParkingSystem = models.ParkingSystem

const RobotParkingTime = models.RobotParkingTime

const (
	ParkingPending = models.ParkingPending
	Parked         = models.Parked
	Rearranging    = models.Rearranging
)

const (
	BikeHourlyPrice  = models.BikeHourlyPrice
	CarHourlyPrice   = models.CarHourlyPrice
	TruckHourlyPrice = models.TruckHourlyPrice
)

const (
	Bike  = models.Bike
	Car   = models.Car
	Truck = models.Truck
)

const (
	BikeSpace  = models.BikeSpace
	CarSpace   = models.CarSpace
	TruckSpace = models.TruckSpace
)

type NearestFloorStrategy = strategy.NearestFloorStrategy
type ZoneBalancedStrategy = strategy.ZoneBalancedStrategy
type CompactionStrategy = strategy.CompactionStrategy

// LoadBalancedStrategy combines allocation + arrangement (same behavior as before).
type LoadBalancedStrategy struct {
	alloc strategy.LoadBalancedStrategy
	arr   arrangement.LoadBalancedRearrangement
}

func (s LoadBalancedStrategy) Allocate(vehicle Vehicle, floors []*Floor) *AllocationDecision {
	return s.alloc.Allocate(vehicle, floors)
}

func (s LoadBalancedStrategy) PlanRearrangement(floors []*Floor, records map[string]*ParkingRecord) *ShufflePlan {
	return s.arr.PlanRearrangement(floors, records)
}

func getVehicleSpace(vehicleType VehicleType) int {
	return models.GetVehicleSpace(vehicleType)
}

func getVehicleRatePerHour(vehicleType VehicleType) float64 {
	return models.GetVehicleRatePerHour(vehicleType)
}

package models

const (
	BikeZonePercent  = 30
	CarZonePercent   = 50
	TruckZonePercent = 20
)

func ZoneRange(floorCount int, vehicleType VehicleType) (start, end int) {
	truckFloors := floorCount * TruckZonePercent / 100
	bikeFloors := floorCount * BikeZonePercent / 100
	carFloors := floorCount * CarZonePercent / 100
	carFloors += floorCount - truckFloors - bikeFloors - carFloors
	switch vehicleType {
	case Truck:
		return 0, truckFloors
	case Car:
		return truckFloors, truckFloors + carFloors
	case Bike:
		return truckFloors + carFloors, floorCount
	default:
		return 0, 0
	}
}

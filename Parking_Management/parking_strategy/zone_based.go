package strategy

import "parking_management/models"

type ZoneBalancedStrategy struct{}

func (s ZoneBalancedStrategy) Allocate(vehicle models.Vehicle, floors []*models.Floor) *models.AllocationDecision {
	requiredSpace := models.GetVehicleSpace(vehicle.Type)
	if requiredSpace == 0 || len(floors) == 0 {
		return nil
	}
	start, end := models.ZoneRange(len(floors), vehicle.Type)
	if start >= end {
		return nil
	}
	for i := start; i < end; i++ {
		floor := floors[i]
		if requiredSpace > (floor.Capacity - floor.ReservedCapacity) {
			continue
		}
		return &models.AllocationDecision{
			FloorNumber: floor.Number,
		}
	}
	return nil
}

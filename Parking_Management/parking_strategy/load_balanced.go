package strategy

import "parking_management/models"

type LoadBalancedStrategy struct{}

func (s LoadBalancedStrategy) Allocate(vehicle models.Vehicle, floors []*models.Floor) *models.AllocationDecision {
	requiredSpace := models.GetVehicleSpace(vehicle.Type)
	if requiredSpace == 0 {
		return nil
	}
	var selectedFloor *models.Floor
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
	return &models.AllocationDecision{
		FloorNumber: selectedFloor.Number,
	}
}

package strategy

import "parking_management/models"

type NearestFloorStrategy struct{}

func (s NearestFloorStrategy) Allocate(vehicle models.Vehicle, floors []*models.Floor) *models.AllocationDecision {
	requiredSpace := models.GetVehicleSpace(vehicle.Type)
	if requiredSpace == 0 {
		return nil
	}
	for _, floor := range floors {
		if requiredSpace > (floor.Capacity - floor.ReservedCapacity) {
			continue
		}
		return &models.AllocationDecision{
			FloorNumber: floor.Number,
		}
	}
	return nil
}

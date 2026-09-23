package strategy

import "parking_management/models"

type CompactionStrategy struct{}

// compaction percentage = reservedcapacity/capacity * 100.0

func (s CompactionStrategy) Allocate(vehicle models.Vehicle, floors []*models.Floor) *models.AllocationDecision {
	requiredSpace := models.GetVehicleSpace(vehicle.Type)
	if requiredSpace == 0 {
		return nil
	}
	var highestCompactFloor *models.Floor
	for _, floor := range floors {
		if floor.Capacity < requiredSpace+floor.ReservedCapacity {
			continue
		}
		if highestCompactFloor == nil {
			highestCompactFloor = floor
			continue
		}
		selectedCompactionPercentage := float64(highestCompactFloor.ReservedCapacity) / float64(highestCompactFloor.Capacity) * 100
		currentCompactionPercentage := float64(floor.ReservedCapacity) / float64(floor.Capacity) * 100
		if selectedCompactionPercentage < currentCompactionPercentage {
			highestCompactFloor = floor
		}
	}
	if highestCompactFloor == nil {
		return nil
	}
	return &models.AllocationDecision{
		FloorNumber: highestCompactFloor.Number,
	}
}

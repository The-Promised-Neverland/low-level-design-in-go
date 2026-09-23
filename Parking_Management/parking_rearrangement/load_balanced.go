package arrangement

import "parking_management/models"

type LoadBalancedRearrangement struct{}

func (s LoadBalancedRearrangement) PlanRearrangement(floors []*models.Floor, records map[string]*models.ParkingRecord) *models.ShufflePlan {
	plan := &models.ShufflePlan{
		Moves: make([]models.VehicleMove, 0),
	}
	if len(floors) == 0 {
		return plan
	}
	targetPlan := make([]int, len(floors))
	for _, record := range records {
		if record.UnparkRequested { // do not plan for vehicles scheduled for unparking
			continue
		}
		requiredSpace := models.GetVehicleSpace(record.Vehicle.Type)
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
			plan.Moves = append(plan.Moves, models.VehicleMove{
				TicketID:      record.TicketID,
				ToFloorNumber: selectedFloor,
			})
		}
	}
	return plan
}

package arrangement

import "parking_management/models"

type ZoneBalancedRearrangement struct{}

func (s ZoneBalancedRearrangement) PlanRearrangement(floors []*models.Floor, records map[string]*models.ParkingRecord) *models.ShufflePlan {
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
		start, end := models.ZoneRange(len(floors), record.Vehicle.Type)
		found := false
		for i := start; i < end; i++ {
			if targetPlan[i]+requiredSpace > floors[i].Capacity {
				continue
			}
			targetPlan[i] += requiredSpace
			if record.TargetFloorNumber != i {
				plan.Moves = append(plan.Moves, models.VehicleMove{
					TicketID:      record.TicketID,
					ToFloorNumber: i,
				})
			}
			found = true
			break
		}
		if !found {
			return nil // this stategy cannot be applied
		}
	}
	return plan
}

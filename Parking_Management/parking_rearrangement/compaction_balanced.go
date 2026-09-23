package arrangement

import "parking_management/models"

/*
DSA problem

You are given an array of positive integers vehicles, where vehicles[i] represents the amount of parking space required by the i-th vehicle.
You are also given:
k — the total number of available floors.
capacity — the maximum capacity of each floor.

Assign every vehicle to exactly one floor such that:
The total space occupied on any floor does not exceed capacity.
Every vehicle must be assigned to exactly one floor.
A vehicle cannot be split across multiple floors.
The number of used (non-empty) floors is minimized.
If it is impossible to accommodate all vehicles within the given k floors, return that no valid arrangement exists.

Example
Input:
vehicles = [6, 6, 4, 4, 3, 3, 2, 2]
k = 4
capacity = 10
One optimal arrangement is:
[
    [6, 4],
    [6, 4],
    [3, 3, 2, 2],
    []
]
*/

type CompactionBalancedRearrangement struct{}

func (s CompactionBalancedRearrangement) PlanRearrangement(floors []*models.Floor, records map[string]*models.ParkingRecord) *models.ShufflePlan {
	plan := &models.ShufflePlan{
		Moves: make([]models.VehicleMove, 0),
	}
	if len(floors) == 0 {
		return plan
	}
	// targetPlan := make([]int, len(floors))
	for _, record := range records {
		if record.UnparkRequested { // do not plan for vehicles scheduled for unparking
			continue
		}
		requiredSpace := models.GetVehicleSpace(record.Vehicle.Type)
		if requiredSpace == 0 {
			continue
		}
	}
	return plan
}

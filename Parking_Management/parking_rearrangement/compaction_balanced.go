package arrangement

import (
	"math"
	"parking_management/models"
	"sort"
)

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

However, its a Bin Packing problem with a NP-hard tag. There isn't a known polynomial-time algorithm
*/

type CompactionBalancedRearrangement struct{}

type VehicleSpace struct {
	TicketID string
	Space    int
}

func (s CompactionBalancedRearrangement) PlanRearrangement(floors []*models.Floor, records map[string]*models.ParkingRecord) *models.MoveList {
	plan := &models.MoveList{
		Moves: make([]models.VehicleMove, 0),
	}
	if len(floors) == 0 {
		return plan
	}
	vehicles := make([]VehicleSpace, 0)
	for ticketID, record := range records {
		if record.UnparkRequested {
			continue
		}
		requiredSpace := models.GetVehicleSpace(record.Vehicle.Type)
		if requiredSpace == 0 {
			continue
		}
		vehicles = append(vehicles, VehicleSpace{
			TicketID: ticketID,
			Space:    requiredSpace,
		})
	}
	sort.Slice(vehicles, func(i, j int) bool {
		return vehicles[i].Space > vehicles[j].Space
	})
	usedSpace := make([]int, len(floors))
	for _, vehicle := range vehicles {
		bestFloor := -1
		bestRemaining := math.MaxInt
		for i := 0; i < len(floors); i++ {
			remaining := floors[i].Capacity - usedSpace[i]
			if remaining >= vehicle.Space && remaining-vehicle.Space < bestRemaining {
				bestRemaining = remaining - vehicle.Space
				bestFloor = i
			}
		}
		if bestFloor == -1 {
			return nil // not able to compact balance
		}
		usedSpace[bestFloor] += vehicle.Space
		record := records[vehicle.TicketID]
		if record.TargetFloorNumber != bestFloor {
			plan.Moves = append(plan.Moves, models.VehicleMove{
				TicketID:        vehicle.TicketID,
				FromFloorNumber: record.TargetFloorNumber,
				ToFloorNumber:   bestFloor,
			})
		}
	}
	return plan
}

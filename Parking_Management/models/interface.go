package models

type ParkingStrategy interface {
	Allocate(vehicle Vehicle, floors []*Floor) *AllocationDecision
}

type FloorsRearrangementStrategy interface {
	PlanRearrangement(floors []*Floor, records map[string]*ParkingRecord) *ShufflePlan
}

type ParkingSystem interface {
	Park(vehicle Vehicle, custName string, expectedParkingHours *int) (*Ticket, error)
	Unpark(ticketID string) (*Receipt, error)
	SwitchStrategy(strategy ParkingStrategy)
}

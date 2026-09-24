package models

type ParkingStrategy interface {
	Allocate(vehicle Vehicle, floors []*Floor) *AllocationDecision
}

type FloorsRearrangementStrategy interface {
	PlanRearrangement(floors []*Floor, records map[string]*ParkingRecord) *MoveList
}

type ParkingSystem interface {
	Park(vehicle Vehicle, custName string) (*Ticket, error)
	Unpark(ticketID string) (*Receipt, error)
	SwitchStrategy(strategy ParkingStrategy) error
}

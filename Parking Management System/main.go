package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type VehicleType string

const (
	Bike       VehicleType = "BIKE"
	Car        VehicleType = "CAR"
	Commercial VehicleType = "TRUCK"
)

type Vehicle struct {
	registrationNo string
	vehicleType    VehicleType
}

func NewVehicle(reg string, vecType VehicleType) *Vehicle {
	return &Vehicle{
		registrationNo: reg,
		vehicleType:    vecType,
	}
}

type Ticket struct {
	id      uuid.UUID
	floor   int
	spot    int
	inTime  time.Time
	outTime time.Time
	vehicle *Vehicle
	fee     int
}

func NewTicket(vehicleType VehicleType, registrationNo string, hoursToPark int) *Ticket {
	now := time.Now()

	return &Ticket{
		id:      uuid.New(),
		vehicle: NewVehicle(registrationNo, vehicleType),
		inTime:  now,
		outTime: now.Add(time.Duration(hoursToPark) * time.Hour),
		fee:     hoursToPark * 20,
	}
}

type Spot struct {
	id          int
	vehicleType VehicleType
	occupied    bool
}

type Floor struct {
	level int
	spots []*Spot
}

type ParkingManager struct {
	floors  []*Floor
	tickets map[uuid.UUID]*Ticket
}

func NewParkingManager(floorCount int) *ParkingManager {

	pm := &ParkingManager{
		floors:  make([]*Floor, 0),
		tickets: make(map[uuid.UUID]*Ticket),
	}

	for floorLvl := 1; floorLvl <= floorCount; floorLvl++ {

		floor := &Floor{
			level: floorLvl,
			spots: make([]*Spot, 0, 12),
		}

		spotID := 1

		for j := 0; j < 5; j++ {
			floor.spots = append(floor.spots, &Spot{spotID, Bike, false})
			spotID++
		}

		for j := 0; j < 5; j++ {
			floor.spots = append(floor.spots, &Spot{spotID, Car, false})
			spotID++
		}

		for j := 0; j < 2; j++ {
			floor.spots = append(floor.spots, &Spot{spotID, Commercial, false})
			spotID++
		}

		pm.floors = append(pm.floors, floor)
	}

	return pm
}

func (pm *ParkingManager) PunchIn(vehType VehicleType, regNo string, hoursToPark int) (*Ticket, error) {

	for _, floor := range pm.floors {

		for _, spot := range floor.spots {

			if vehType == spot.vehicleType && !spot.occupied {

				spot.occupied = true

				ticket := NewTicket(vehType, regNo, hoursToPark)
				ticket.floor = floor.level
				ticket.spot = spot.id

				pm.tickets[ticket.id] = ticket

				return ticket, nil
			}
		}
	}

	return nil, errors.New("no spots available")
}

func (pm *ParkingManager) PunchOut(ticketID uuid.UUID) (int, error) {

	ticket, exists := pm.tickets[ticketID]

	if !exists {
		return 0, errors.New("ticket not found")
	}

	for _, floor := range pm.floors {

		if floor.level == ticket.floor {

			for _, spot := range floor.spots {

				if spot.id == ticket.spot {
					spot.occupied = false
				}
			}
		}
	}

	delete(pm.tickets, ticketID)

	now := time.Now()
	intendedAt := ticket.outTime

	if !now.After(intendedAt) {
		return 0, nil
	}

	delay := now.Sub(intendedAt)
	delayMinutes := int(delay.Minutes())

	blocks := (delayMinutes + 4) / 5

	return blocks * 20, nil
}

func main() {

	manager := NewParkingManager(3)

	for {

		fmt.Println("\n1. Park Vehicle")
		fmt.Println("2. Exit Vehicle")
		fmt.Println("3. Exit Program")

		var choice int
		fmt.Scanln(&choice)

		switch choice {

		case 1:

			var reg string
			var vtype string
			var hours int

			fmt.Print("Registration: ")
			fmt.Scanln(&reg)

			fmt.Print("Vehicle Type (BIKE/CAR/TRUCK): ")
			fmt.Scanln(&vtype)

			fmt.Print("Hours: ")
			fmt.Scanln(&hours)

			ticket, err := manager.PunchIn(VehicleType(vtype), reg, hours)

			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Println("Ticket issued:", ticket.id)
			fmt.Println("Price:", ticket.fee)
			fmt.Println("Floor:", ticket.floor, "Spot:", ticket.spot)

		case 2:

			var idStr string

			fmt.Print("Enter ticket ID: ")
			fmt.Scanln(&idStr)

			ticketID, err := uuid.Parse(idStr)

			if err != nil {
				fmt.Println("Invalid UUID")
				continue
			}

			charge, err := manager.PunchOut(ticketID)

			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Println("Extra charge:", charge)

		case 3:
			fmt.Println("Exiting...")
			return
		}
	}
}

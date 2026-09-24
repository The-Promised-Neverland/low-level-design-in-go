package main

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

type simConfig struct {
	MinArrivalGap time.Duration
	MaxArrivalGap time.Duration
	ExitInterval  time.Duration
	ExitsPerBatch int
}

func defaultSimConfig() simConfig {
	return simConfig{
		MinArrivalGap: 20 * time.Second, // ~3 / min
		MaxArrivalGap: 20 * time.Second,
		ExitInterval:  5 * time.Minute,
		ExitsPerBatch: 2,
	}
}

var customerNames = []string{
	"Asha", "Ravi", "Meera", "Arjun", "Neha", "Karan", "Priya", "Vikram", "Ananya", "Rohan",
}

func randomVehicleType() VehicleType {
	switch rand.IntN(10) {
	case 0, 1, 2:
		return Bike
	case 3, 4, 5, 6, 7:
		return Car
	default:
		return Truck
	}
}

func randomRegNo() string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return fmt.Sprintf("KA%02d%c%c%04d",
		rand.IntN(100),
		letters[rand.IntN(len(letters))],
		letters[rand.IntN(len(letters))],
		rand.IntN(10000),
	)
}

func randomCustomerName() string {
	return customerNames[rand.IntN(len(customerNames))]
}

func randomBetween(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int64N(int64(max-min)+1))
}

func startAutoSimulation(ps ParkingSystem, obs *observabilityStore, cfg simConfig) {
	var mu sync.Mutex
	activeTickets := make([]string, 0, 128)

	go func() {
		for {
			time.Sleep(randomBetween(cfg.MinArrivalGap, cfg.MaxArrivalGap))
			vehicle := Vehicle{VehicleRegNo: randomRegNo(), Type: randomVehicleType()}
			obs.recordArrival(vehicle)
			ticket, err := ps.Park(vehicle, randomCustomerName())
			if err != nil {
				continue
			}
			mu.Lock()
			activeTickets = append(activeTickets, ticket.TicketID)
			mu.Unlock()
		}
	}()

	go func() {
		exits := cfg.ExitsPerBatch
		if exits <= 0 {
			exits = 2
		}
		interval := cfg.ExitInterval
		if interval <= 0 {
			interval = 5 * time.Minute
		}
		for {
			time.Sleep(interval)
			mu.Lock()
			n := exits
			if n > len(activeTickets) {
				n = len(activeTickets)
			}
			batch := append([]string(nil), activeTickets[:n]...)
			activeTickets = activeTickets[n:]
			mu.Unlock()
			for _, id := range batch {
				_, _ = ps.Unpark(id)
			}
		}
	}()
}

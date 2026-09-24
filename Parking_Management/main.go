package main

import (
	"log"
	"net/http"
)

func main() {
	obs := newObservabilityStore(200)
	robotConfig := RobotPoolConfig{
		ParkingRobots:   2,
		UnparkingRobots: 2,
		ShuffleRobots:   2,
	}
	aps := NewAutomatedParkingSystem(10, 20, robotConfig)
	aps.observer = obs
	obs.initRobots(robotConfig)

	startAutoSimulation(aps, obs, defaultSimConfig())

	addr := ":8000"
	log.Printf("parking observability API listening on %s", addr)
	if err := http.ListenAndServe(addr, newAPIHandler(aps, obs)); err != nil {
		log.Fatal(err)
	}
}

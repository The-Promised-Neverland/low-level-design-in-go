package main

import "fmt"

func (p *AutomatedParkingSystem) spinupUnparkingRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		robotID := fmt.Sprintf("UNPARK-%d", i+1)
		go p.unparkers(robotID)
	}
}

func (p *AutomatedParkingSystem) spinupParkingRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		robotID := fmt.Sprintf("PARK-%d", i+1)
		go p.parkers(robotID)
	}
}

func (p *AutomatedParkingSystem) spinupShuffleRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		robotID := fmt.Sprintf("SHUFFLE-%d", i+1)
		go p.shuffleRobot(robotID)
	}
}

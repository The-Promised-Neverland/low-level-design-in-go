package main

func (p *AutomatedParkingSystem) spinupUnparkingRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		go p.unparkers()
	}
}

func (p *AutomatedParkingSystem) spinupParkingRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		go p.parkers()
	}
}

func (p *AutomatedParkingSystem) spinupShuffleRobotsPool(workers int) {
	for i := 0; i < workers; i++ {
		go p.shuffleRobot()
	}
}

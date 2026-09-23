package main

import "time"

func (p *AutomatedParkingSystem) processShuffleJob(job ShuffleJob) {
	p.mu.Lock()
	record, ok := p.recordsByTktID[job.TicketID]
	if !ok {
		p.mu.Unlock()
		return
	}
	if record.UnparkRequested == true { //  no need to shuffle. its schedule for unpark
		p.mu.Unlock()
		return
	}
	if record.TargetFloorNumber != job.ToFloorNumber {
		p.mu.Unlock()
		return
	}
	if record.PhysicalFloorNumber != job.FromFloorNumber {
		p.mu.Unlock()
		return
	}
	requiredSpace := getVehicleSpace(record.Vehicle.Type)
	fromFloor := p.floors[job.FromFloorNumber]
	toFloor := p.floors[job.ToFloorNumber]
	stale := false
	for toFloor.Capacity-toFloor.PhysicallyUsedCapacity < requiredSpace {
		toFloor.Cond.Wait()
		if record.TargetFloorNumber != job.ToFloorNumber || record.UnparkRequested {
			stale = true
			break
		}
	}
	if stale {
		p.mu.Unlock()
		return
	}
	toFloor.PhysicallyUsedCapacity += requiredSpace
	record.ParkingStatus = Rearranging // it has now been handed to the shuffler bots
	p.mu.Unlock()
	time.Sleep(RobotParkingTime)
	p.mu.Lock()
	record, ok = p.recordsByTktID[job.TicketID]
	if !ok {
		toFloor.PhysicallyUsedCapacity -= requiredSpace
		toFloor.Cond.Broadcast()
		p.mu.Unlock()
		return
	}
	fromFloor.PhysicallyUsedCapacity -= requiredSpace
	record.PhysicalFloorNumber = job.ToFloorNumber
	record.ParkingStatus = Parked
	fromFloor.Cond.Broadcast()
	shouldUnpark := record.UnparkRequested
	p.mu.Unlock()
	if shouldUnpark {
		p.unparkJobCh <- UnparkJob{
			TicketID: job.TicketID,
		}
	}
}

func (p *AutomatedParkingSystem) shuffleRobot() {
	for job := range p.shuffleJobCh {
		p.processShuffleJob(job)
		p.ShuffleJobWG.Done()
	}
}

func (p *AutomatedParkingSystem) rearrange() {
	for range p.rearrangeCh {
		p.mu.Lock()
		rearrangementStrategy, ok := p.strategy.(FloorsRearrangementStrategy)
		if !ok {
			p.mu.Unlock()
			continue
		}
		plan := rearrangementStrategy.PlanRearrangement(p.floors, p.recordsByTktID)
		jobs := make([]ShuffleJob, 0, len(plan.Moves))
		for _, move := range plan.Moves {
			record, ok := p.recordsByTktID[move.TicketID]
			if !ok {
				continue
			}
			if record.UnparkRequested { // no need to rearrange the vehicles scheduled for unparking
				continue
			}
			requiredSpace := getVehicleSpace(record.Vehicle.Type)
			oldTarget := record.TargetFloorNumber
			newTarget := move.ToFloorNumber
			if oldTarget == newTarget {
				continue
			}
			p.floors[oldTarget].ReservedCapacity -= requiredSpace
			p.floors[newTarget].ReservedCapacity += requiredSpace
			record.TargetFloorNumber = newTarget
			if record.PhysicalFloorNumber != -1 && record.PhysicalFloorNumber != newTarget {
				jobs = append(jobs, ShuffleJob{
					TicketID:        record.TicketID,
					FromFloorNumber: record.PhysicalFloorNumber,
					ToFloorNumber:   newTarget,
				})
			}
		}
		p.mu.Unlock()
		p.ShuffleJobWG.Add(len(jobs))
		for _, job := range jobs {
			p.shuffleJobCh <- job
		}
		p.ShuffleJobWG.Wait()
		p.mu.Lock()
		p.shufflingInProgress = false
		p.mu.Unlock()
	}
}

func (p *AutomatedParkingSystem) parkers() {
	for job := range p.parkingJobCh {
		requiredSpace := getVehicleSpace(job.Vehicle.Type)
		p.mu.Lock()
		floor := p.floors[job.ToFloorNumber]
		for floor.Capacity-floor.PhysicallyUsedCapacity < requiredSpace {
			floor.Cond.Wait()
		}
		floor.PhysicallyUsedCapacity += requiredSpace
		p.mu.Unlock()
		time.Sleep(RobotParkingTime)
		p.mu.Lock()
		shouldUnpark := false
		if record, ok := p.recordsByTktID[job.TicketID]; ok {
			record.PhysicalFloorNumber = job.ToFloorNumber
			record.ParkingStatus = Parked // mark as parked
			if record.UnparkRequested {   // customer reqeusted unparking
				shouldUnpark = true
			}
		}
		p.mu.Unlock()
		if shouldUnpark {
			p.unparkJobCh <- UnparkJob{ // schedule for unpark
				TicketID: job.TicketID,
			}
		}
	}
}

func (p *AutomatedParkingSystem) unparkers() {
	for job := range p.unparkJobCh {
		p.mu.Lock()
		record, ok := p.recordsByTktID[job.TicketID]
		if !ok {
			p.mu.Unlock()
			continue
		}
		if !record.UnparkRequested {
			p.mu.Unlock()
			continue
		}
		fromFloorNumber := record.PhysicalFloorNumber
		requiredSpace := getVehicleSpace(record.Vehicle.Type)
		p.mu.Unlock()
		time.Sleep(RobotParkingTime)
		p.mu.Lock()
		fromFloor := p.floors[fromFloorNumber]
		fromFloor.PhysicallyUsedCapacity -= requiredSpace
		fromFloor.Cond.Broadcast()
		record.PhysicalFloorNumber = -1
		delete(p.recordsByTktID, record.TicketID)
		delete(p.recordsByRegNo, record.Vehicle.VehicleRegNo)
		p.mu.Unlock()
	}
}

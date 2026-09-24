package main

import (
	"time"
)

func (p *AutomatedParkingSystem) processShuffleComponent(robotID string, comp ShuffleComponent) {
	p.mu.Lock()
	moves := p.waitComponentReadyLocked(comp.Moves)
	p.mu.Unlock()
	for _, move := range moves {
		p.processShuffleJob(robotID, move)
	}
}

func (p *AutomatedParkingSystem) waitComponentReadyLocked(planned []VehicleMove) []VehicleMove {
	for {
		ready := make([]VehicleMove, 0, len(planned))
		waitingOnPark := false
		for _, m := range planned {
			record, ok := p.recordsByTktID[m.TicketID]
			if !ok || record.UnparkRequested {
				continue
			}
			if record.ParkingStatus == ParkingPending {
				waitingOnPark = true
				continue
			}
			if record.ParkingStatus == Rearranging {
				waitingOnPark = true
				continue
			}
			if record.PhysicalFloorNumber < 0 {
				waitingOnPark = true
				continue
			}
			if record.PhysicalFloorNumber == record.TargetFloorNumber {
				continue
			}
			ready = append(ready, VehicleMove{
				TicketID:        record.TicketID,
				FromFloorNumber: record.PhysicalFloorNumber,
				ToFloorNumber:   record.TargetFloorNumber,
			})
		}
		if !waitingOnPark {
			return ready
		}
		p.shuffleWake.Wait() // minimizes cpu cycles by not continously checking
	}
}

func (p *AutomatedParkingSystem) processShuffleJob(robotID string, job VehicleMove) {
	p.mu.Lock()
	record, ok := p.recordsByTktID[job.TicketID]
	if !ok {
		p.mu.Unlock()
		return
	}
	if record.UnparkRequested {
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
	fromFloor.PhysicallyUsedCapacity -= requiredSpace
	fromFloor.Cond.Broadcast()
	record.PhysicalFloorNumber = -1
	record.ParkingStatus = Rearranging
	vehicleReg := record.Vehicle.VehicleRegNo
	p.mu.Unlock()
	if p.observer != nil {
		p.observer.OnShuffleStarted(robotID, job, vehicleReg)
	}
	time.Sleep(RobotParkingTime)
	p.mu.Lock()
	record, ok = p.recordsByTktID[job.TicketID]
	if !ok {
		p.mu.Unlock()
		if p.observer != nil {
			p.observer.OnShuffleCompleted(robotID, job, vehicleReg)
		}
		return
	}
	dest := record.TargetFloorNumber
	toFloor := p.floors[dest]
	for toFloor.Capacity-toFloor.PhysicallyUsedCapacity < requiredSpace {
		toFloor.Cond.Wait()
		if record.TargetFloorNumber != dest {
			dest = record.TargetFloorNumber
			toFloor = p.floors[dest]
		}
	}
	toFloor.PhysicallyUsedCapacity += requiredSpace
	record.PhysicalFloorNumber = dest
	record.ParkingStatus = Parked
	shouldUnpark := record.UnparkRequested
	vehicleReg = record.Vehicle.VehicleRegNo
	p.shuffleWake.Broadcast()
	p.mu.Unlock()
	if p.observer != nil {
		p.observer.OnShuffleCompleted(robotID, job, vehicleReg)
	}
	if shouldUnpark {
		p.unparkJobCh <- UnparkJob{
			TicketID: job.TicketID,
		}
	}
}

func (p *AutomatedParkingSystem) shuffleRobot(robotID string) {
	for comp := range p.shuffleJobCh {
		p.processShuffleComponent(robotID, comp)
		p.ShuffleJobWG.Done()
	}
}

func (p *AutomatedParkingSystem) parkers(robotID string) {
	for job := range p.parkingJobCh {
		if p.observer != nil {
			p.observer.OnParkingStarted(robotID, job)
		}
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
			record.ParkingStatus = Parked
			if record.UnparkRequested {
				shouldUnpark = true
			}
			p.shuffleWake.Broadcast()
		}
		p.mu.Unlock()
		if p.observer != nil {
			p.observer.OnParkingCompleted(robotID, job)
		}
		if shouldUnpark {
			p.unparkJobCh <- UnparkJob{
				TicketID: job.TicketID,
			}
		}
	}
}

func (p *AutomatedParkingSystem) unparkers(robotID string) {
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
		vehicle := record.Vehicle
		p.mu.Unlock()
		if p.observer != nil {
			p.observer.OnUnparkingStarted(robotID, job.TicketID, vehicle, fromFloorNumber)
		}
		time.Sleep(RobotParkingTime)
		p.mu.Lock()
		if fromFloorNumber >= 0 {
			fromFloor := p.floors[fromFloorNumber]
			fromFloor.PhysicallyUsedCapacity -= requiredSpace
			fromFloor.Cond.Broadcast()
		}
		record.PhysicalFloorNumber = -1
		delete(p.recordsByTktID, record.TicketID)
		delete(p.recordsByRegNo, record.Vehicle.VehicleRegNo)
		p.mu.Unlock()
		if p.observer != nil {
			p.observer.OnUnparkingCompleted(robotID, job.TicketID, vehicle, fromFloorNumber)
		}
	}
}

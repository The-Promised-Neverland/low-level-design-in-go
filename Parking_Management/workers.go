package main

import (
	"time"
)

func (p *AutomatedParkingSystem) processShuffleComponent(comp ShuffleComponent) {
	p.mu.Lock()
	moves := p.waitTillComponentReady(comp.Moves)
	p.mu.Unlock()
	if len(moves) == 0 {
		return
	}
	if !comp.Loop {
		for _, move := range moves {
			p.processShuffleJob(move)
		}
		return
	}
	p.mu.Lock()
	bufferID := p.claimBufferSlot()
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.releaseBufferSlot(bufferID)
		p.mu.Unlock()
	}()
	breaker := moves[0]
	p.processShuffleJob(VehicleMove{
		TicketID:        breaker.TicketID,
		FromFloorNumber: breaker.FromFloorNumber,
		ToFloorNumber:   bufferID,
	})
	remaining := make([]VehicleMove, 0, len(moves)-1)
	p.mu.Lock()
	for _, move := range moves[1:] {
		record, ok := p.recordsByTktID[move.TicketID]
		if !ok || record.UnparkRequested {
			continue
		}
		if record.BufferSlotIndex >= 0 || record.PhysicalFloorNumber < 0 {
			continue
		}
		if record.PhysicalFloorNumber == record.TargetFloorNumber {
			continue
		}
		remaining = append(remaining, VehicleMove{
			TicketID:        record.TicketID,
			FromFloorNumber: record.PhysicalFloorNumber,
			ToFloorNumber:   record.TargetFloorNumber,
		})
	}
	p.mu.Unlock()
	if len(remaining) > 0 {
		order := BuildMovementOrder(remaining)
		for _, sub := range order.Moves {
			for _, move := range sub {
				p.processShuffleJob(move)
			}
		}
	}
	p.mu.Lock()
	record, ok := p.recordsByTktID[breaker.TicketID]
	var home VehicleMove
	needHome := false
	if ok && record.BufferSlotIndex >= 0 {
		needHome = true
		home = VehicleMove{
			TicketID:        record.TicketID,
			FromFloorNumber: BufferSlotID(record.BufferSlotIndex),
			ToFloorNumber:   record.TargetFloorNumber,
		}
	}
	p.mu.Unlock()
	if needHome {
		p.processShuffleJob(home)
	}
}

func (p *AutomatedParkingSystem) waitTillComponentReady(planned []VehicleMove) []VehicleMove {
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
			if record.BufferSlotIndex >= 0 {
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
		p.shuffleWake.Wait()
	}
}

func (p *AutomatedParkingSystem) processShuffleJob(job VehicleMove) {
	p.mu.Lock()
	record, ok := p.recordsByTktID[job.TicketID]
	if !ok {
		p.mu.Unlock()
		return
	}
	fromBuffer := IsBufferSlotID(job.FromFloorNumber)
	if fromBuffer {
		if record.BufferSlotIndex != bufferSlotIndex(job.FromFloorNumber) {
			p.mu.Unlock()
			return
		}
	} else if record.BufferSlotIndex >= 0 || record.PhysicalFloorNumber != job.FromFloorNumber {
		p.mu.Unlock()
		return
	}
	requiredSpace := getVehicleSpace(record.Vehicle.Type)
	if fromBuffer {
		p.removeVehicleFromBufferSlot(job.FromFloorNumber)
		record.BufferSlotIndex = -1
	} else {
		fromFloor := p.floors[job.FromFloorNumber]
		fromFloor.PhysicallyUsedCapacity -= requiredSpace
		fromFloor.Cond.Broadcast()
	}
	record.PhysicalFloorNumber = -1
	record.ParkingStatus = Rearranging
	p.mu.Unlock()
	time.Sleep(RobotParkingTime)
	p.mu.Lock()
	record, ok = p.recordsByTktID[job.TicketID]
	if !ok {
		p.mu.Unlock()
		return
	}
	toBuffer := IsBufferSlotID(job.ToFloorNumber)
	if toBuffer {
		p.placeVehicleInBufferSlot(job.ToFloorNumber, job.TicketID)
		record.BufferSlotIndex = bufferSlotIndex(job.ToFloorNumber)
		record.PhysicalFloorNumber = -1
	} else {
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
		record.BufferSlotIndex = -1
	}
	record.ParkingStatus = Parked
	shouldUnpark := record.UnparkRequested && record.BufferSlotIndex < 0 && record.PhysicalFloorNumber >= 0
	p.shuffleWake.Broadcast()
	p.mu.Unlock()
	if shouldUnpark {
		p.unparkJobCh <- UnparkJob{
			TicketID: job.TicketID,
		}
	}
}

func (p *AutomatedParkingSystem) shuffleRobot() {
	for comp := range p.shuffleJobCh {
		p.processShuffleComponent(comp)
		p.ShuffleJobWG.Done()
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
			record.BufferSlotIndex = -1
			record.ParkingStatus = Parked
			if record.UnparkRequested {
				shouldUnpark = true
			}
			p.shuffleWake.Broadcast()
		}
		p.mu.Unlock()
		if shouldUnpark {
			p.unparkJobCh <- UnparkJob{
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
		if fromFloorNumber >= 0 {
			fromFloor := p.floors[fromFloorNumber]
			fromFloor.PhysicallyUsedCapacity -= requiredSpace
			fromFloor.Cond.Broadcast()
		}
		record.PhysicalFloorNumber = -1
		delete(p.recordsByTktID, record.TicketID)
		delete(p.recordsByRegNo, record.Vehicle.VehicleRegNo)
		p.mu.Unlock()
	}
}

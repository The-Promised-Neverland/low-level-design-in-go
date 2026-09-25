package main

const (
	BufferSlotCount  = 3
	BufferSlotIDBase = 1000000
)

type bufferSlot struct {
	claimed       bool
	vehicleTicket string
}

func BufferSlotID(slotIndex int) int {
	return BufferSlotIDBase + slotIndex
}

func IsBufferSlotID(floorNumber int) bool {
	return floorNumber >= BufferSlotIDBase && floorNumber < BufferSlotIDBase+BufferSlotCount
}

func bufferSlotIndex(floorNumber int) int {
	return floorNumber - BufferSlotIDBase
}

func (p *AutomatedParkingSystem) claimBufferSlot() int {
	for {
		for i := range p.bufferSlots {
			if !p.bufferSlots[i].claimed {
				p.bufferSlots[i].claimed = true
				return BufferSlotID(i)
			}
		}
		p.bufferSlotAvailable.Wait()
	}
}

func (p *AutomatedParkingSystem) releaseBufferSlot(bufferID int) {
	if !IsBufferSlotID(bufferID) {
		return
	}
	i := bufferSlotIndex(bufferID)
	p.bufferSlots[i] = bufferSlot{}
	p.bufferSlotAvailable.Broadcast()
}

func (p *AutomatedParkingSystem) removeVehicleFromBufferSlot(bufferID int) {
	if !IsBufferSlotID(bufferID) {
		return
	}
	i := bufferSlotIndex(bufferID)
	p.bufferSlots[i].vehicleTicket = ""
}

func (p *AutomatedParkingSystem) placeVehicleInBufferSlot(bufferID int, ticketID string) {
	i := bufferSlotIndex(bufferID)
	p.bufferSlots[i].vehicleTicket = ticketID
}

export type Robot = { id: string; type: "PARKING" | "UNPARKING" | "SHUFFLE"; status: "IDLE" | "BUSY"; ticketId?: string; vehicleRegNo?: string; fromFloor?: number | null; toFloor?: number | null };
export type Vehicle = { ticketId: string; vehicleRegNo: string; vehicleType: "BIKE" | "CAR" | "TRUCK"; customerName?: string; parkingStatus: "PARKING_PENDING" | "PARKED" | "REARRANGING"; unparkRequested: boolean; physicalFloor: number; targetFloor: number; entryTime: string };
export type QItem = { ticketId: string; vehicleRegNo: string; vehicleType: string; fromFloor?: number; toFloor?: number };
export type Floor = { floorNumber: number; capacity: number; reservedCapacity: number; physicallyUsedCapacity: number; zone: string; vehicles: Vehicle[] };
export type State = {
  strategy: string; shufflingInProgress: boolean;
  metrics: { vehiclesArrived: number; vehiclesAccepted: number; vehiclesRejected: number; vehiclesExited: number; activeVehicles: number };
  queues: { staging: QItem[]; waitingForUnpark: QItem[]; waitingForShuffle: QItem[] };
  robots: { parking: Robot[]; unparking: Robot[]; shuffle: Robot[] };
  floors: Floor[]; vehicles: Vehicle[];
};
export type Ev = { timestamp: string; type: string; ticketId?: string; vehicleRegNo?: string; robotId?: string; fromFloor?: number; toFloor?: number; message?: string };

package main

import (
	"fmt"
	"time"
)

func main() {
	elevator := NewElevatorSystem("E1", 10)
	defer elevator.Shutdown()

	go func() {
		for {
			state := elevator.State()
			fmt.Printf(
				"Floor: %2d | Direction: %-4s | Door: %-6s | Pending: %v\n",
				state.CurrentFloor,
				state.Direction,
				state.DoorState,
				state.Pending,
			)
			time.Sleep(time.Second)
		}
	}()

	fmt.Println("CALL: Floor 8, DOWN")
	elevator.Call(8, Down)

	time.Sleep(2 * time.Second)

	fmt.Println("CALL: Floor 3, UP")
	elevator.Call(3, Up)

	time.Sleep(2 * time.Second)

	fmt.Println("SELECT: Floor 10")
	elevator.SelectFloor(10)

	time.Sleep(60 * time.Second)
}

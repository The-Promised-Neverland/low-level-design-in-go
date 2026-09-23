# Automated Smart Parking System

## Problem Statement

Design an automated multi-floor parking system.

Unlike a traditional parking lot, customers do not drive around looking for a parking spot. A customer hands over their vehicle at an entry point, receives a ticket, and the automated parking system takes responsibility for storing and retrieving the vehicle.

The system should support multiple vehicle types, multiple floors, configurable parking strategies, automatic rearrangement of vehicles, billing, and concurrent parking and retrieval requests.

## Parking Facility

The parking facility consists of multiple numbered floors.

Each floor has a fixed capacity represented in space units.

All floors have the same capacity.

The system must ensure that the capacity of a floor is never exceeded.

## Vehicle Types

The following vehicle types are supported:

| Vehicle Type | Space Required |
| --- | ---: |
| Bike | 1 unit |
| Car | 2 units |
| Truck | 6 units |

For example, a floor with a capacity of 20 units may accommodate different combinations of bikes, cars, and trucks as long as their total required space does not exceed 20 units.

## Parking a Vehicle

When a customer arrives, the following information is provided:

- Customer name
- Vehicle registration number
- Vehicle type

The customer does not select a parking floor.

The system must automatically determine where the vehicle should be stored.

If sufficient parking capacity is not available, the parking request should be rejected.

If the request is accepted, the customer should immediately receive a parking ticket.

The actual movement and storage of the vehicle may take some time after the request has been accepted.

## Parking Ticket

A successful parking request generates a unique ticket.

The ticket should contain:

- Ticket ID
- Vehicle details
- Entry time

Example ticket format:

`TKT-DDMMYY-XXXXX`

The ticket should not expose the internal storage location of the vehicle.

The ticket ID must be used when requesting the vehicle back.

A vehicle registration number alone is not sufficient for retrieval.

## Parking Strategies

The parking system must support multiple parking allocation strategies.

### Nearest Floor Strategy

The vehicle should be assigned to the nearest available floor that has sufficient capacity.

Floors are considered in numerical order.

### Load Balanced Strategy

Vehicles should be distributed across the available floors so that parking usage remains reasonably balanced.

## Runtime Strategy Switching

The active parking strategy can be changed while the parking facility is operating.

For example:

`Nearest Floor -> Load Balanced`

New parking requests should follow the newly selected strategy.

A strategy change may also require vehicles already inside the facility to be rearranged.

## Automatic Vehicle Rearrangement

The automated parking facility is allowed to move vehicles between floors without customer involvement.

For example, vehicles may initially be concentrated on the lower floors while using the Nearest Floor strategy.

After switching to Load Balanced, the system may decide to redistribute some of those vehicles across other floors.

The customer should not need a new ticket when this happens.

A valid ticket must continue to identify the correct vehicle regardless of any internal rearrangement.

Vehicles that are already being retrieved should not be unnecessarily rearranged.

## Retrieving a Vehicle

A customer requests their vehicle using the ticket ID.

The system must validate the ticket and locate the corresponding vehicle.

The vehicle may currently be:

- Waiting to be stored
- Already stored
- In the process of being rearranged

The system must safely handle retrieval requests in all of these situations.

Once a retrieval request has been accepted, another retrieval request for the same parking session should not also be accepted.

The customer does not need to know where the vehicle is stored.

Once retrieval is complete, the parking space occupied by the vehicle becomes available for other vehicles.

## Parking Charges

Parking charges depend on the vehicle type and parking duration.

| Vehicle Type | Hourly Rate |
| --- | ---: |
| Bike | ₹20/hour |
| Car | ₹40/hour |
| Truck | ₹60/hour |

Partial hours are rounded up.

A minimum charge of one hour applies.

Examples:

`25 minutes -> 1 hour`

`1 hour 10 minutes -> 2 hours`

`2 hours 1 minute -> 3 hours`

## Receipt

A successful retrieval request should generate a receipt containing:

- Ticket ID
- Vehicle details
- Entry time
- Exit time
- Total amount

## Concurrent Operations

The parking facility can have multiple entry and exit points.

Multiple operations may therefore happen at the same time.

For example:

- Several vehicles may arrive simultaneously.
- Several customers may request their vehicles simultaneously.
- New vehicles may arrive while other vehicles are being retrieved.
- Vehicles may arrive while the parking strategy is being changed.
- A customer may request their vehicle shortly after parking it.
- A customer may request their vehicle while the facility is rearranging vehicles.

The system must remain consistent under these concurrent operations.

In particular:

- A floor must never exceed its capacity.
- A vehicle must not be assigned more than once for the same parking session.
- The same ticket must not result in multiple successful retrievals.
- Capacity must become reusable after a vehicle leaves.
- Strategy changes must not corrupt existing parking sessions.
- Internal vehicle movement must not invalidate a customer's ticket.

## Example

Consider a parking facility with:

`3 floors`

`20 capacity units per floor`

The following vehicles arrive:

`Car A    -> 2 units`

`Bike B   -> 1 unit`

`Truck C  -> 6 units`

`Car D    -> 2 units`

The system should accept the vehicles when sufficient capacity exists, assign their storage locations according to the active parking strategy, and return tickets to the customers.

While these vehicles are inside the facility, the operator may switch from the Nearest Floor strategy to the Load Balanced strategy.

The system may rearrange existing vehicles as necessary.

At the same time, new customers may arrive and existing customers may request their vehicles.

The implementation should correctly coordinate these operations while maintaining capacity constraints and valid parking sessions.
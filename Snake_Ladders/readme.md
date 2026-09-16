# Snake and Ladder

## Problem Statement

Design and implement a Snake and Ladder game.

The game is played on a board consisting of numbered cells. Multiple players take turns rolling dice and moving their pieces across the board.

The first player to reach the final cell wins the game.

## Requirements

- The board contains 100 cells, numbered from 1 to 100.

- The game must support two or more players.

- Each player starts at position `0`, before the first cell.

- Players take turns in a fixed order.

- The game must support one or more dice.

- Each standard die produces a value between `1` and `6`.

- When multiple dice are used, the values of all dice are added together to determine the player's movement.

- A player moves forward by the value obtained from the dice roll.

- The board can contain multiple snakes and ladders.

- If a player lands exactly on the head of a snake, the player moves to the snake's tail.

- If a player lands exactly at the bottom of a ladder, the player moves to the top of the ladder.

- A player must reach cell `100` exactly to win.

- If a dice roll would move a player beyond cell `100`, the player remains at their current position.

- After a player's turn is completed, the next player gets the turn.

- The game ends immediately when a player reaches cell `100`.

- Once the game has finished, no additional turns can be played.

## Board Configuration

Snakes and ladders are configured before the game starts.

A snake is represented by:

```text
Head -> Tail
```

For example:

```text
87 -> 24
```

A ladder is represented by:

```text
Start -> End
```

For example:

```text
12 -> 46
```

A snake must always move a player to a lower-numbered cell.

A ladder must always move a player to a higher-numbered cell.

## Interface

```go
type Game interface {
	AddPlayer(name string) error

	AddSnake(head, tail int) error
	AddLadder(start, end int) error

	Start() error
	PlayTurn() (*TurnResult, error)

	GetCurrentPlayer() *Player
	GetWinner() *Player
	GetStatus() GameStatus
}
```

```go
type GameStatus string

const (
	GameNotStarted GameStatus = "NOT_STARTED"
	GameInProgress GameStatus = "IN_PROGRESS"
	GameFinished   GameStatus = "FINISHED"
)
```

```go
type Player struct {
	ID       string
	Name     string
	Position int
}
```

```go
type TurnResult struct {
	Player       *Player
	DiceRoll     int
	FromPosition int
	ToPosition   int
}
```

## Example

Consider the following board configuration:

```text
Ladders:

12 -> 46
29 -> 74
41 -> 79

Snakes:

38 -> 15
67 -> 32
87 -> 24
```

Player A is currently at cell `25`.

```text
Player A
Position: 25

Roll: 4

25 + 4 = 29
```

Cell `29` contains a ladder:

```text
29 -> 74
```

Therefore:

```text
Start Position : 25
Dice Roll      : 4
Landed On      : 29
Final Position : 74
```

Player A's turn ends at cell `74`.

## Snake Example

Player B is currently at cell `81` and rolls a `6`.

```text
81 + 6 = 87
```

Cell `87` contains a snake:

```text
87 -> 24
```

Therefore:

```text
Start Position : 81
Dice Roll      : 6
Landed On      : 87
Final Position : 24
```

## Exact Winning Position

Suppose Player A is currently at cell `96`.

Player A rolls a `6`.

```text
96 + 6 = 102
```

Since the final cell is `100`, the move is not allowed.

The player's position remains:

```text
96
```

If Player A later rolls a `4`:

```text
96 + 4 = 100
```

Player A reaches the final cell and wins the game.

## Turn Order

For three players:

```text
Player A
   |
   v
Player B
   |
   v
Player C
   |
   v
Player A
   |
   ...
```

The order remains fixed throughout the game.

If Player B reaches cell `100` during their turn:

```text
Player A -> turn completed

Player B -> reaches 100
             |
             v
          WINNER
             |
             v
        GAME FINISHED
```

Player C does not receive another turn.

## Validation Rules

- At least two players must be added before the game can start.

- Player names must not be empty.

- A snake's head must be greater than its tail.

- A ladder's end must be greater than its start.

- Snake and ladder positions must be within the board boundaries.

- A snake or ladder cannot start at cell `100`.

- Players cannot be added after the game has started.

- Snakes and ladders cannot be added after the game has started.

- `Start()` cannot start an already running or completed game.

- `PlayTurn()` cannot be called before the game starts.

- `PlayTurn()` cannot be called after the game has finished.

## Scope

The game maintains its state entirely in memory.

The game is responsible for maintaining the board configuration, players, turn order, player positions, dice rolls, and winner.

Persistence of game state is outside the scope of this problem.

Networking, multiplayer communication over a network, authentication, matchmaking, game recovery, and distributed game execution are outside the scope of this problem.
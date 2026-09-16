package main

import (
	"testing"
)

// ---------------------------------------------------------
// Helpers
// ---------------------------------------------------------

func newTestGame(t *testing.T) *SnakeLadders {
	t.Helper()

	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddPlayer("Alice"); err != nil {
		t.Fatal(err)
	}

	if err := g.AddPlayer("Bob"); err != nil {
		t.Fatal(err)
	}

	return g
}

func startedGame(t *testing.T) *SnakeLadders {
	t.Helper()

	g := newTestGame(t)

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	return g
}

func setDiceSequence(g *SnakeLadders, rolls ...int) {
	i := 0

	g.rollDice = func() int {
		if i >= len(rolls) {
			panic("test dice sequence exhausted")
		}

		roll := rolls[i]
		i++

		return roll
	}
}

// ---------------------------------------------------------
// Game creation
// ---------------------------------------------------------

func TestNewGame(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if g.Status != GameNotStarted {
		t.Fatalf("expected NOT_STARTED, got %s", g.Status)
	}

	if len(g.Players) != 0 {
		t.Fatalf("expected 0 players, got %d", len(g.Players))
	}

	if g.CurrentTurn != 0 {
		t.Fatalf("expected current turn 0, got %d", g.CurrentTurn)
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatalf(
			"expected consecutive sixes 0, got %d",
			g.ConsecutiveSixes,
		)
	}

	if len(g.Snakes) != 101 {
		t.Fatalf("expected snake board size 101, got %d", len(g.Snakes))
	}

	if len(g.Ladders) != 101 {
		t.Fatalf("expected ladder board size 101, got %d", len(g.Ladders))
	}
}

// ---------------------------------------------------------
// Players
// ---------------------------------------------------------

func TestAddPlayer(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	err := g.AddPlayer("Alice")
	if err != nil {
		t.Fatal(err)
	}

	if len(g.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(g.Players))
	}

	player := g.Players[0]

	if player.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", player.Name)
	}

	if player.Position != 0 {
		t.Fatalf("expected starting position 0, got %d", player.Position)
	}

	if len(player.ID) != 6 {
		t.Fatalf("expected 6 character ID, got %q", player.ID)
	}
}

func TestMultiplePlayers(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddPlayer("Charlie")

	if len(g.Players) != 3 {
		t.Fatalf("expected 3 players, got %d", len(g.Players))
	}
}

func TestCannotAddPlayerAfterStart(t *testing.T) {
	g := startedGame(t)

	err := g.AddPlayer("Charlie")

	if err == nil {
		t.Fatal("expected error when adding player after game start")
	}

	if len(g.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(g.Players))
	}
}

// ---------------------------------------------------------
// Start
// ---------------------------------------------------------

func TestCannotStartWithoutPlayers(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	err := g.Start()

	if err == nil {
		t.Fatal("expected error when starting without players")
	}
}

func TestCannotStartWithOnePlayer(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")

	err := g.Start()

	if err == nil {
		t.Fatal("expected error when starting with one player")
	}
}

func TestStartWithTwoPlayers(t *testing.T) {
	g := newTestGame(t)

	err := g.Start()
	if err != nil {
		t.Fatal(err)
	}

	if g.Status != GameInProgress {
		t.Fatalf("expected IN_PROGRESS, got %s", g.Status)
	}
}

func TestCannotStartTwice(t *testing.T) {
	g := startedGame(t)

	err := g.Start()

	if err == nil {
		t.Fatal("expected error when starting game twice")
	}
}

// ---------------------------------------------------------
// Snake validation
// ---------------------------------------------------------

func TestAddSnake(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	err := g.AddSnake(50, 20)
	if err != nil {
		t.Fatal(err)
	}

	if g.Snakes[50] != 20 {
		t.Fatalf("expected snake 50 -> 20")
	}
}

func TestSnakeHeadMustBeGreaterThanTail(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddSnake(20, 50); err == nil {
		t.Fatal("expected invalid snake error")
	}

	if err := g.AddSnake(20, 20); err == nil {
		t.Fatal("expected invalid snake error")
	}
}

func TestInvalidSnakeBoundaries(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	tests := []struct {
		head int
		tail int
	}{
		{0, 1},
		{100, 50},
		{101, 50},
		{-1, 1},
		{50, 0},
		{50, -1},
	}

	for _, tc := range tests {
		err := g.AddSnake(tc.head, tc.tail)

		if err == nil {
			t.Fatalf(
				"expected invalid snake %d -> %d",
				tc.head,
				tc.tail,
			)
		}
	}
}

func TestDuplicateSnake(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddSnake(50, 20); err != nil {
		t.Fatal(err)
	}

	if err := g.AddSnake(50, 10); err == nil {
		t.Fatal("expected duplicate snake error")
	}
}

// ---------------------------------------------------------
// Ladder validation
// ---------------------------------------------------------

func TestAddLadder(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	err := g.AddLadder(10, 40)
	if err != nil {
		t.Fatal(err)
	}

	if g.Ladders[10] != 40 {
		t.Fatalf("expected ladder 10 -> 40")
	}
}

func TestLadderEndMustBeGreaterThanStart(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddLadder(50, 20); err == nil {
		t.Fatal("expected invalid ladder error")
	}

	if err := g.AddLadder(50, 50); err == nil {
		t.Fatal("expected invalid ladder error")
	}
}

func TestInvalidLadderBoundaries(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	tests := []struct {
		start int
		end   int
	}{
		{0, 20},
		{-1, 20},
		{100, 100},
		{101, 100},
		{50, 101},
	}

	for _, tc := range tests {
		err := g.AddLadder(tc.start, tc.end)

		if err == nil {
			t.Fatalf(
				"expected invalid ladder %d -> %d",
				tc.start,
				tc.end,
			)
		}
	}
}

func TestLadderCanEndAt100(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	err := g.AddLadder(90, 100)
	if err != nil {
		t.Fatal(err)
	}

	if g.Ladders[90] != 100 {
		t.Fatal("expected ladder 90 -> 100")
	}
}

func TestDuplicateLadder(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddLadder(10, 40); err != nil {
		t.Fatal(err)
	}

	if err := g.AddLadder(10, 50); err == nil {
		t.Fatal("expected duplicate ladder error")
	}
}

// ---------------------------------------------------------
// Snake / ladder collision
// ---------------------------------------------------------

func TestCannotPutSnakeWhereLadderStarts(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddLadder(20, 50); err != nil {
		t.Fatal(err)
	}

	if err := g.AddSnake(20, 5); err == nil {
		t.Fatal("expected collision error")
	}
}

func TestCannotPutLadderWhereSnakeStarts(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddSnake(20, 5); err != nil {
		t.Fatal(err)
	}

	if err := g.AddLadder(20, 50); err == nil {
		t.Fatal("expected collision error")
	}
}

func TestCannotAddSnakeAfterGameStarts(t *testing.T) {
	g := startedGame(t)

	if err := g.AddSnake(50, 20); err == nil {
		t.Fatal("expected error")
	}
}

func TestCannotAddLadderAfterGameStarts(t *testing.T) {
	g := startedGame(t)

	if err := g.AddLadder(20, 50); err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------
// Normal movement
// ---------------------------------------------------------

func TestNormalMovement(t *testing.T) {
	g := startedGame(t)

	setDiceSequence(g, 4)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.FromPosition != 0 {
		t.Fatalf("expected from 0, got %d", result.FromPosition)
	}

	if result.ToPosition != 4 {
		t.Fatalf("expected position 4, got %d", result.ToPosition)
	}

	if g.Players[0].Position != 4 {
		t.Fatalf(
			"expected player position 4, got %d",
			g.Players[0].Position,
		)
	}
}

func TestMovementFromExistingPosition(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 25

	setDiceSequence(g, 4)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.FromPosition != 25 {
		t.Fatalf("expected from 25, got %d", result.FromPosition)
	}

	if result.ToPosition != 29 {
		t.Fatalf("expected to 29, got %d", result.ToPosition)
	}
}

// ---------------------------------------------------------
// Turn rotation
// ---------------------------------------------------------

func TestTurnMovesToNextPlayer(t *testing.T) {
	g := startedGame(t)

	setDiceSequence(g, 3)

	_, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if g.CurrentTurn != 1 {
		t.Fatalf("expected Bob's turn, got index %d", g.CurrentTurn)
	}

	if g.GetCurrentPlayer().Name != "Bob" {
		t.Fatalf(
			"expected Bob, got %s",
			g.GetCurrentPlayer().Name,
		)
	}
}

func TestTurnWrapsAround(t *testing.T) {
	g := startedGame(t)

	setDiceSequence(g, 3, 4)

	_, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	_, err = g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if g.CurrentTurn != 0 {
		t.Fatalf("expected turn to wrap to Alice")
	}
}

// ---------------------------------------------------------
// Snake movement
// ---------------------------------------------------------

func TestLandingOnSnake(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddSnake(29, 10); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	g.Players[0].Position = 25

	setDiceSequence(g, 4)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.FromPosition != 25 {
		t.Fatalf("expected from 25")
	}

	if result.ToPosition != 10 {
		t.Fatalf(
			"expected snake to move player to 10, got %d",
			result.ToPosition,
		)
	}

	if g.Players[0].Position != 10 {
		t.Fatalf("expected player at 10")
	}
}

func TestPassingSnakeDoesNothing(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddSnake(7, 2); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	g.Players[0].Position = 6

	setDiceSequence(g, 5)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 11 {
		t.Fatalf(
			"expected position 11, got %d",
			result.ToPosition,
		)
	}
}

// ---------------------------------------------------------
// Ladder movement
// ---------------------------------------------------------

func TestLandingOnLadder(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddLadder(29, 74); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	g.Players[0].Position = 25

	setDiceSequence(g, 4)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 74 {
		t.Fatalf(
			"expected ladder to move player to 74, got %d",
			result.ToPosition,
		)
	}

	if g.Players[0].Position != 74 {
		t.Fatalf("expected player position 74")
	}
}

func TestPassingLadderDoesNothing(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddLadder(7, 50); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	g.Players[0].Position = 6

	setDiceSequence(g, 5)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 11 {
		t.Fatalf("expected position 11, got %d", result.ToPosition)
	}
}

// ---------------------------------------------------------
// Overshooting
// ---------------------------------------------------------

func TestOvershoot100(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 98

	setDiceSequence(g, 4)

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatal("expected overshoot error")
	}

	if result.ToPosition != 98 {
		t.Fatalf(
			"expected player to remain at 98, got %d",
			result.ToPosition,
		)
	}

	if g.Players[0].Position != 98 {
		t.Fatalf("player should remain at 98")
	}

	if g.CurrentTurn != 1 {
		t.Fatal("turn should pass to next player")
	}
}

// ---------------------------------------------------------
// Sixes
// ---------------------------------------------------------

func TestSingleSixGrantsAnotherRoll(t *testing.T) {
	g := startedGame(t)

	setDiceSequence(g, 6)

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatal("expected another-turn response")
	}

	if result.DiceRoll != 6 {
		t.Fatalf("expected dice 6")
	}

	if g.ConsecutiveSixes != 1 {
		t.Fatalf(
			"expected 1 consecutive six, got %d",
			g.ConsecutiveSixes,
		)
	}

	if g.CurrentTurn != 0 {
		t.Fatal("turn should remain with Alice")
	}

	if g.Players[0].Position != 0 {
		t.Fatal("position should not change until culmination")
	}
}

func TestTwoSixesKeepSamePlayer(t *testing.T) {
	g := startedGame(t)

	setDiceSequence(g, 6, 6)

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()

	if g.ConsecutiveSixes != 2 {
		t.Fatalf(
			"expected 2 consecutive sixes, got %d",
			g.ConsecutiveSixes,
		)
	}

	if g.CurrentTurn != 0 {
		t.Fatal("Alice should still have the turn")
	}

	if g.Players[0].Position != 0 {
		t.Fatal("position should remain unchanged")
	}
}

func TestTripleSixForfeitsEntireTurn(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 20

	setDiceSequence(g, 6, 6, 6)

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatal("expected triple-six error")
	}

	if result.ToPosition != 20 {
		t.Fatalf(
			"expected player to remain at 20, got %d",
			result.ToPosition,
		)
	}

	if g.Players[0].Position != 20 {
		t.Fatal("entire turn should be forfeited")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatal("six counter should reset")
	}

	if g.CurrentTurn != 1 {
		t.Fatal("turn should move to Bob")
	}
}

// ---------------------------------------------------------
// Culmination
// ---------------------------------------------------------

func TestSingleSixCulmination(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 10

	setDiceSequence(g, 6, 3)

	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	// 10 + 6 + 3 = 19

	if result.ToPosition != 19 {
		t.Fatalf(
			"expected culmination position 19, got %d",
			result.ToPosition,
		)
	}
}

func TestDoubleSixCulmination(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 10

	setDiceSequence(g, 6, 6, 3)

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	// 10 + 6 + 6 + 3 = 25

	if result.ToPosition != 25 {
		t.Fatalf(
			"expected culmination position 25, got %d",
			result.ToPosition,
		)
	}

	if g.Players[0].Position != 25 {
		t.Fatalf("expected player at 25")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatal("counter should reset after culmination")
	}
}

func TestSnakeOnlyChecksFinalCulminatedPosition(t *testing.T) {
	g := newTestGame(t)

	// During conceptual movement:
	// 10 + 6 = 16
	//
	// But under culmination rules the player does NOT land at 16.
	// Final result is 10 + 6 + 3 = 19.

	if err := g.AddSnake(16, 2); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	g.Players[0].Position = 10

	setDiceSequence(g, 6, 3)

	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 19 {
		t.Fatalf(
			"expected 19 because snake at 16 should be ignored, got %d",
			result.ToPosition,
		)
	}
}

func TestSnakeAtFinalCulminatedPosition(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddSnake(19, 5); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	g.Players[0].Position = 10

	setDiceSequence(g, 6, 3)

	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 5 {
		t.Fatalf(
			"expected snake to move player to 5, got %d",
			result.ToPosition,
		)
	}
}

func TestLadderAtFinalCulminatedPosition(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddLadder(19, 70); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	g.Players[0].Position = 10

	setDiceSequence(g, 6, 3)

	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 70 {
		t.Fatalf(
			"expected ladder to move player to 70, got %d",
			result.ToPosition,
		)
	}
}

// ---------------------------------------------------------
// Culmination + overshooting
// ---------------------------------------------------------

func TestCulminatedRollOvershoots(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 90

	setDiceSequence(g, 6, 6, 3)

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatal("expected overshoot error")
	}

	// 90 + 6 + 6 + 3 = 105
	// Player must remain at 90.

	if result.ToPosition != 90 {
		t.Fatalf(
			"expected position 90, got %d",
			result.ToPosition,
		)
	}

	if g.Players[0].Position != 90 {
		t.Fatal("player should remain at 90")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatal("six counter should reset")
	}
}

// ---------------------------------------------------------
// Winning
// ---------------------------------------------------------

func TestExact100Wins(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 96

	setDiceSequence(g, 4)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 100 {
		t.Fatalf("expected position 100")
	}

	if g.Status != GameFinished {
		t.Fatalf(
			"expected FINISHED, got %s",
			g.Status,
		)
	}

	winner := g.GetWinner()

	if winner == nil {
		t.Fatal("expected winner")
	}

	if winner.Name != "Alice" {
		t.Fatalf(
			"expected Alice to win, got %s",
			winner.Name,
		)
	}
}

func TestLadderTo100Wins(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddLadder(95, 100); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	g.Players[0].Position = 91

	setDiceSequence(g, 4)

	_, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if g.Status != GameFinished {
		t.Fatal("expected game to finish")
	}

	if g.Players[0].Position != 100 {
		t.Fatal("expected player at 100")
	}
}

func TestCulminatedRollCanWin(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 85

	setDiceSequence(g, 6, 6, 3)

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	// 85 + 6 + 6 + 3 = 100

	if result.ToPosition != 100 {
		t.Fatalf(
			"expected 100, got %d",
			result.ToPosition,
		)
	}

	if g.Status != GameFinished {
		t.Fatal("expected game to finish")
	}

	if g.GetWinner() == nil {
		t.Fatal("expected winner")
	}

	if g.GetWinner().Name != "Alice" {
		t.Fatal("expected Alice to win")
	}
}

func TestCannotPlayAfterWinner(t *testing.T) {
	g := startedGame(t)

	g.Players[0].Position = 96

	setDiceSequence(g, 4)

	_, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	_, err = g.PlayTurn()

	if err == nil {
		t.Fatal("expected error when playing finished game")
	}
}

// ---------------------------------------------------------
// Getters
// ---------------------------------------------------------

func TestGetCurrentPlayerBeforeStart(t *testing.T) {
	g := newTestGame(t)

	if g.GetCurrentPlayer() != nil {
		t.Fatal("expected nil before game starts")
	}
}

func TestGetCurrentPlayer(t *testing.T) {
	g := startedGame(t)

	player := g.GetCurrentPlayer()

	if player == nil {
		t.Fatal("expected current player")
	}

	if player.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", player.Name)
	}
}

func TestGetWinnerBeforeGameFinished(t *testing.T) {
	g := startedGame(t)

	if g.GetWinner() != nil {
		t.Fatal("expected nil winner")
	}
}

func TestGetStatus(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if g.GetStatus() != GameNotStarted {
		t.Fatal("expected NOT_STARTED")
	}

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.Start()

	if g.GetStatus() != GameInProgress {
		t.Fatal("expected IN_PROGRESS")
	}
}

// ---------------------------------------------------------
// Multi-player game
// ---------------------------------------------------------

func TestThreePlayerTurnRotation(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddPlayer("Charlie")

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	setDiceSequence(g, 1, 2, 3, 4)

	if g.GetCurrentPlayer().Name != "Alice" {
		t.Fatal("expected Alice")
	}

	_, _ = g.PlayTurn()

	if g.GetCurrentPlayer().Name != "Bob" {
		t.Fatal("expected Bob")
	}

	_, _ = g.PlayTurn()

	if g.GetCurrentPlayer().Name != "Charlie" {
		t.Fatal("expected Charlie")
	}

	_, _ = g.PlayTurn()

	if g.GetCurrentPlayer().Name != "Alice" {
		t.Fatal("expected turn to wrap to Alice")
	}
}

// ---------------------------------------------------------
// Full deterministic game
// ---------------------------------------------------------

func TestCompleteGame(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")

	_ = g.AddLadder(4, 25)
	_ = g.AddLadder(40, 70)

	_ = g.AddSnake(35, 15)
	_ = g.AddSnake(80, 50)

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	/*
		We'll eventually force Alice to win.

		The purpose here isn't to test randomness.
		It's to make sure a sequence of turns can
		progress through the game correctly.
	*/

	setDiceSequence(g,
		4, // Alice: 0 -> 4 -> ladder -> 25
		5, // Bob:   0 -> 5
		5, // Alice: 25 -> 30
		4, // Bob:   5 -> 9
		5, // Alice: 30 -> 35 -> snake -> 15
		3, // Bob:   9 -> 12
		5, // Alice: 15 -> 20
		4, // Bob:   12 -> 16
		5, // Alice: 20 -> 25
		3, // Bob:   16 -> 19
		5, // Alice: 25 -> 30
		4, // Bob:   19 -> 23
		4, // Alice: 30 -> 34
		3, // Bob:   23 -> 26
		5, // Alice: 34 -> 39
		4, // Bob:   26 -> 30
		1, // Alice: 39 -> 40 -> ladder -> 70
		3, // Bob:   30 -> 33
		5, // Alice: 70 -> 75
		4, // Bob:   33 -> 37
		4, // Alice: 75 -> 79
		3, // Bob:   37 -> 40 -> ladder -> 70
		5, // Alice: 79 -> 84
		4, // Bob:   70 -> 74
		5, // Alice: 84 -> 89
		3, // Bob:   74 -> 77
		5, // Alice: 89 -> 94
		4, // Bob:   77 -> 81
		5, // Alice: 94 -> 99
		3, // Bob:   81 -> 84
		1, // Alice: 99 -> 100
	)

	for g.GetStatus() == GameInProgress {
		_, err := g.PlayTurn()

		// Six-related errors don't occur in this sequence.
		if err != nil {
			t.Fatalf("unexpected play error: %v", err)
		}
	}

	winner := g.GetWinner()

	if winner == nil {
		t.Fatal("expected winner")
	}

	if winner.Name != "Alice" {
		t.Fatalf(
			"expected Alice to win, got %s",
			winner.Name,
		)
	}

	if winner.Position != 100 {
		t.Fatalf(
			"expected winner position 100, got %d",
			winner.Position,
		)
	}

	if g.GetStatus() != GameFinished {
		t.Fatal("expected game to be finished")
	}
}
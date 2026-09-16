package main

import (
	"strings"
	"testing"
)

func newTestGame(t *testing.T, rolls ...int) *SnakeLadders {
	t.Helper()

	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddPlayer("Alice"); err != nil {
		t.Fatal(err)
	}

	if err := g.AddPlayer("Bob"); err != nil {
		t.Fatal(err)
	}

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	i := 0
	g.rollDice = func() int {
		if i >= len(rolls) {
			t.Fatalf("unexpected dice roll")
		}
		r := rolls[i]
		i++
		return r
	}

	return g
}

// ============================================================
// Initialization
// ============================================================

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
		t.Fatalf("expected 0 consecutive sixes")
	}

	if g.Winner != nil {
		t.Fatalf("expected nil winner")
	}
}

// ============================================================
// Players
// ============================================================

func TestAddPlayer(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddPlayer("Alice"); err != nil {
		t.Fatal(err)
	}

	if len(g.Players) != 1 {
		t.Fatalf("expected 1 player")
	}

	if g.Players[0].Name != "Alice" {
		t.Fatalf("expected Alice")
	}

	if g.Players[0].Position != 0 {
		t.Fatalf("player should start at position 0")
	}

	if len(g.Players[0].ID) != 6 {
		t.Fatalf("expected 6 character ID")
	}
}

func TestMultiplePlayers(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddPlayer("Charlie")

	if len(g.Players) != 3 {
		t.Fatalf("expected 3 players")
	}
}

func TestPlayerIDIsAlphanumeric(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")

	id := g.Players[0].ID

	if len(id) != 6 {
		t.Fatalf("expected ID length 6")
	}

	for _, c := range id {
		valid :=
			(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9')

		if !valid {
			t.Fatalf("invalid character in ID: %c", c)
		}
	}
}

func TestCannotAddPlayerAfterStart(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddPlayer("Charlie"); err == nil {
		t.Fatalf("expected error")
	}
}

// ============================================================
// Start
// ============================================================

func TestCannotStartWithoutPlayers(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.Start(); err == nil {
		t.Fatalf("expected error")
	}

	if g.Status != GameNotStarted {
		t.Fatalf("game status should remain NOT_STARTED")
	}
}

func TestCannotStartWithOnePlayer(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")

	if err := g.Start(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestStartWithTwoPlayers(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")

	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	if g.Status != GameInProgress {
		t.Fatalf("expected IN_PROGRESS")
	}
}

func TestCannotStartTwice(t *testing.T) {
	g := newTestGame(t)

	if err := g.Start(); err == nil {
		t.Fatalf("expected error")
	}
}

// ============================================================
// Snakes
// ============================================================

func TestAddSnake(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddSnake(50, 20); err != nil {
		t.Fatal(err)
	}

	if g.Snakes[50] != 20 {
		t.Fatalf("expected snake 50 -> 20")
	}
}

func TestSnakeHeadMustBeGreaterThanTail(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddSnake(20, 50); err == nil {
		t.Fatalf("expected error")
	}

	if err := g.AddSnake(20, 20); err == nil {
		t.Fatalf("expected error")
	}
}

func TestInvalidSnakeBoundaries(t *testing.T) {
	tests := []struct {
		head int
		tail int
	}{
		{0, 1},
		{-1, 1},
		{100, 20},
		{101, 20},
		{50, 0},
		{50, -1},
	}

	for _, tc := range tests {
		g := NewSnakesLadders().(*SnakeLadders)

		if err := g.AddSnake(tc.head, tc.tail); err == nil {
			t.Fatalf(
				"expected error for snake %d -> %d",
				tc.head,
				tc.tail,
			)
		}
	}
}

func TestDuplicateSnake(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddSnake(50, 20)

	if err := g.AddSnake(50, 10); err == nil {
		t.Fatalf("expected duplicate snake error")
	}
}

func TestSnakeCanEndAt1(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddSnake(50, 1); err != nil {
		t.Fatal(err)
	}
}

func TestCannotAddSnakeAfterStart(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddSnake(50, 20); err == nil {
		t.Fatalf("expected error")
	}
}

// ============================================================
// Ladders
// ============================================================

func TestAddLadder(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddLadder(10, 50); err != nil {
		t.Fatal(err)
	}

	if g.Ladders[10] != 50 {
		t.Fatalf("expected ladder 10 -> 50")
	}
}

func TestLadderEndMustBeGreaterThanStart(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddLadder(50, 20); err == nil {
		t.Fatalf("expected error")
	}

	if err := g.AddLadder(50, 50); err == nil {
		t.Fatalf("expected error")
	}
}

func TestInvalidLadderBoundaries(t *testing.T) {
	tests := []struct {
		start int
		end   int
	}{
		{0, 50},
		{-1, 50},
		{100, 100},
		{101, 100},
		{20, 101},
	}

	for _, tc := range tests {
		g := NewSnakesLadders().(*SnakeLadders)

		if err := g.AddLadder(tc.start, tc.end); err == nil {
			t.Fatalf(
				"expected error for ladder %d -> %d",
				tc.start,
				tc.end,
			)
		}
	}
}

func TestLadderCanEndAt100(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddLadder(80, 100); err != nil {
		t.Fatal(err)
	}
}

func TestLadderCanStartAt1(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if err := g.AddLadder(1, 50); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateLadder(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddLadder(10, 50)

	if err := g.AddLadder(10, 60); err == nil {
		t.Fatalf("expected duplicate ladder error")
	}
}

// ============================================================
// Snake/Ladder collisions
// ============================================================

func TestCannotPutSnakeWhereLadderStarts(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddLadder(20, 60)

	if err := g.AddSnake(20, 10); err == nil {
		t.Fatalf("expected collision error")
	}
}

func TestCannotPutLadderWhereSnakeStarts(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddSnake(50, 20)

	if err := g.AddLadder(50, 80); err == nil {
		t.Fatalf("expected collision error")
	}
}

func TestCannotAddLadderAfterStart(t *testing.T) {
	g := newTestGame(t)

	if err := g.AddLadder(10, 50); err == nil {
		t.Fatalf("expected error")
	}
}

// ============================================================
// Normal movement
// ============================================================

func TestNormalMovement(t *testing.T) {
	g := newTestGame(t, 4)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.FromPosition != 0 {
		t.Fatalf("expected from position 0")
	}

	if result.ToPosition != 4 {
		t.Fatalf("expected position 4, got %d", result.ToPosition)
	}

	if g.Players[0].Position != 4 {
		t.Fatalf("Alice should be at 4")
	}
}

func TestMovementFromExistingPosition(t *testing.T) {
	g := newTestGame(t, 4)

	g.Players[0].Position = 20

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.FromPosition != 20 {
		t.Fatalf("expected from 20")
	}

	if result.ToPosition != 24 {
		t.Fatalf("expected 24, got %d", result.ToPosition)
	}
}

func TestTurnMovesToNextPlayer(t *testing.T) {
	g := newTestGame(t, 4)

	_, _ = g.PlayTurn()

	if g.CurrentTurn != 1 {
		t.Fatalf("expected Bob's turn")
	}
}

func TestTurnWrapsAround(t *testing.T) {
	g := newTestGame(t, 2, 3)

	_, _ = g.PlayTurn()

	if g.CurrentTurn != 1 {
		t.Fatalf("expected Bob")
	}

	_, _ = g.PlayTurn()

	if g.CurrentTurn != 0 {
		t.Fatalf("expected Alice")
	}
}

// ============================================================
// Snakes during play
// ============================================================

func TestLandingOnSnake(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddSnake(14, 5)
	_ = g.Start()

	g.Players[0].Position = 10
	g.rollDice = func() int { return 4 }

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 5 {
		t.Fatalf("expected snake to move player to 5")
	}

	if g.Players[0].Position != 5 {
		t.Fatalf("expected Alice at 5")
	}
}

func TestPassingSnakeDoesNothing(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddSnake(12, 5)
	_ = g.Start()

	g.Players[0].Position = 10
	g.rollDice = func() int { return 4 }

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 14 {
		t.Fatalf("expected 14, got %d", result.ToPosition)
	}
}

// ============================================================
// Ladders during play
// ============================================================

func TestLandingOnLadder(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddLadder(14, 50)
	_ = g.Start()

	g.Players[0].Position = 10
	g.rollDice = func() int { return 4 }

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 50 {
		t.Fatalf("expected ladder to move player to 50")
	}
}

func TestPassingLadderDoesNothing(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddLadder(12, 50)
	_ = g.Start()

	g.Players[0].Position = 10
	g.rollDice = func() int { return 4 }

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 14 {
		t.Fatalf("expected 14")
	}
}

// ============================================================
// Six behavior
// ============================================================

func TestSingleSixDoesNotMovePlayer(t *testing.T) {
	g := newTestGame(t, 6)

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatalf("expected another-turn result")
	}

	if result.ToPosition != 0 {
		t.Fatalf("player should not move yet")
	}

	if g.Players[0].Position != 0 {
		t.Fatalf("player should remain at 0")
	}

	if g.ConsecutiveSixes != 1 {
		t.Fatalf("expected 1 stored six")
	}

	if g.CurrentTurn != 0 {
		t.Fatalf("expected Alice to retain turn")
	}
}

func TestTwoSixesDoNotMovePlayer(t *testing.T) {
	g := newTestGame(t, 6, 6)

	_, _ = g.PlayTurn()
	result, err := g.PlayTurn()

	if err == nil {
		t.Fatalf("expected another-turn result")
	}

	if result.ToPosition != 0 {
		t.Fatalf("player should still be at 0")
	}

	if g.Players[0].Position != 0 {
		t.Fatalf("player should still be at 0")
	}

	if g.ConsecutiveSixes != 2 {
		t.Fatalf("expected 2 stored sixes")
	}

	if g.CurrentTurn != 0 {
		t.Fatalf("Alice should retain turn")
	}
}

func TestSingleSixCulmination(t *testing.T) {
	g := newTestGame(t, 6, 4)

	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 10 {
		t.Fatalf("expected 6 + 4 = 10, got %d", result.ToPosition)
	}

	if g.Players[0].Position != 10 {
		t.Fatalf("expected Alice at 10")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatalf("six counter should reset")
	}
}

func TestDoubleSixCulmination(t *testing.T) {
	g := newTestGame(t, 6, 6, 4)

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 16 {
		t.Fatalf("expected 6 + 6 + 4 = 16")
	}

	if g.Players[0].Position != 16 {
		t.Fatalf("expected Alice at 16")
	}
}

func TestTripleSixForfeitsEntireTurn(t *testing.T) {
	g := newTestGame(t, 6, 6, 6)

	g.Players[0].Position = 20

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatalf("expected triple-six error")
	}

	if !strings.Contains(err.Error(), "three sixes") {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FromPosition != 20 || result.ToPosition != 20 {
		t.Fatalf("player should remain at 20")
	}

	if g.Players[0].Position != 20 {
		t.Fatalf("Alice should remain at 20")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatalf("six counter should reset")
	}

	if g.CurrentTurn != 1 {
		t.Fatalf("turn should move to Bob")
	}
}

// ============================================================
// Overshoot behavior of THIS implementation
// ============================================================

func TestNormalRollOvershoots100(t *testing.T) {
	g := newTestGame(t, 4)

	g.Players[0].Position = 98

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatalf("expected overshoot error")
	}

	if result.FromPosition != 98 || result.ToPosition != 98 {
		t.Fatalf("player should remain at 98")
	}

	if g.Players[0].Position != 98 {
		t.Fatalf("position should remain 98")
	}

	if g.CurrentTurn != 1 {
		t.Fatalf("turn should move to Bob")
	}
}

func TestSixCanImmediatelyOvershoot(t *testing.T) {
	g := newTestGame(t, 6)

	g.Players[0].Position = 95

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatalf("expected overshoot")
	}

	if !strings.Contains(err.Error(), "Overshooting") {
		t.Fatalf("expected overshoot error, got %v", err)
	}

	if result.FromPosition != 95 || result.ToPosition != 95 {
		t.Fatalf("player should remain at 95")
	}

	if g.Players[0].Position != 95 {
		t.Fatalf("Alice should remain at 95")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatalf("six counter should reset")
	}

	if g.CurrentTurn != 1 {
		t.Fatalf("turn should move to Bob")
	}
}

func TestSecondSixCanImmediatelyOvershoot(t *testing.T) {
	g := newTestGame(t, 6, 6)

	g.Players[0].Position = 90

	_, err := g.PlayTurn()
	if err == nil {
		t.Fatalf("first six should grant another roll")
	}

	_, err = g.PlayTurn()
	if err == nil {
		t.Fatalf("second six should overshoot")
	}

	if !strings.Contains(err.Error(), "Overshooting") {
		t.Fatalf("expected overshoot error")
	}

	if g.Players[0].Position != 90 {
		t.Fatalf("Alice should remain at 90")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatalf("six counter should reset")
	}

	if g.CurrentTurn != 1 {
		t.Fatalf("turn should move to Bob")
	}
}

func TestCulminatedRollOvershoots(t *testing.T) {
	g := newTestGame(t, 6, 5)

	g.Players[0].Position = 90

	_, err := g.PlayTurn()
	if err == nil {
		t.Fatalf("six should grant another roll")
	}

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatalf("expected culmination overshoot")
	}

	if result.FromPosition != 90 || result.ToPosition != 90 {
		t.Fatalf("Alice should remain at 90")
	}

	if g.Players[0].Position != 90 {
		t.Fatalf("position should remain 90")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatalf("six counter should reset")
	}

	if g.CurrentTurn != 1 {
		t.Fatalf("turn should move to Bob")
	}
}

// ============================================================
// Culminated snake/ladder behavior
// ============================================================

func TestSnakeAtFinalCulminatedPosition(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddSnake(20, 5)
	_ = g.Start()

	g.Players[0].Position = 10

	rolls := []int{6, 4}
	i := 0
	g.rollDice = func() int {
		r := rolls[i]
		i++
		return r
	}

	_, _ = g.PlayTurn()
	result, err := g.PlayTurn()

	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 5 {
		t.Fatalf("expected snake to move player to 5")
	}
}

func TestIntermediateSnakeIgnoredDuringStoredSix(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddSnake(16, 2)
	_ = g.Start()

	g.Players[0].Position = 10

	rolls := []int{6, 4}
	i := 0
	g.rollDice = func() int {
		r := rolls[i]
		i++
		return r
	}

	_, _ = g.PlayTurn()
	result, err := g.PlayTurn()

	if err != nil {
		t.Fatal(err)
	}

	// 10 + 6 + 4 = 20.
	// Snake at intermediate position 16 is ignored.
	if result.ToPosition != 20 {
		t.Fatalf("expected 20, got %d", result.ToPosition)
	}
}

func TestLadderAtFinalCulminatedPosition(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddLadder(20, 70)
	_ = g.Start()

	g.Players[0].Position = 10

	rolls := []int{6, 4}
	i := 0
	g.rollDice = func() int {
		r := rolls[i]
		i++
		return r
	}

	_, _ = g.PlayTurn()
	result, err := g.PlayTurn()

	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 70 {
		t.Fatalf("expected ladder to move player to 70")
	}
}

// ============================================================
// Winning
// ============================================================

func TestExact100Wins(t *testing.T) {
	g := newTestGame(t, 2)

	g.Players[0].Position = 98

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 100 {
		t.Fatalf("expected 100")
	}

	if g.Status != GameFinished {
		t.Fatalf("expected FINISHED")
	}

	if g.Winner != &g.Players[0] {
		t.Fatalf("expected Alice to be winner")
	}

	if g.Players[0].Position != 100 {
		t.Fatalf("winner should remain at 100")
	}
}

func TestLadderTo100Wins(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddLadder(95, 100)
	_ = g.Start()

	g.Players[0].Position = 91
	g.rollDice = func() int { return 4 }

	_, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if g.Status != GameFinished {
		t.Fatalf("expected game finished")
	}

	if g.Winner != &g.Players[0] {
		t.Fatalf("expected Alice to win")
	}
}

func TestCulminatedRollCanWin(t *testing.T) {
	g := newTestGame(t, 6, 4)

	g.Players[0].Position = 90

	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 100 {
		t.Fatalf("expected 100")
	}

	if g.Status != GameFinished {
		t.Fatalf("expected FINISHED")
	}
}

func TestSnakePreventsWin(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddSnake(99, 50)
	_ = g.Start()

	g.Players[0].Position = 95
	g.rollDice = func() int { return 4 }

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.ToPosition != 50 {
		t.Fatalf("expected snake to move player to 50")
	}

	if g.Status != GameInProgress {
		t.Fatalf("game should continue")
	}

	if g.Winner != nil {
		t.Fatalf("winner should remain nil")
	}
}

func TestCannotPlayAfterWinner(t *testing.T) {
	g := newTestGame(t, 2)

	g.Players[0].Position = 98

	_, _ = g.PlayTurn()

	result, err := g.PlayTurn()

	if err == nil {
		t.Fatalf("expected error")
	}

	if result != nil {
		t.Fatalf("expected nil result")
	}
}

// ============================================================
// Getters
// ============================================================

func TestGetCurrentPlayerBeforeStart(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")

	if g.GetCurrentPlayer() != nil {
		t.Fatalf("expected nil before start")
	}
}

func TestGetCurrentPlayer(t *testing.T) {
	g := newTestGame(t)

	player := g.GetCurrentPlayer()

	if player == nil {
		t.Fatalf("expected player")
	}

	if player.Name != "Alice" {
		t.Fatalf("expected Alice")
	}
}

func TestGetCurrentPlayerAfterFinish(t *testing.T) {
	g := newTestGame(t, 1)

	g.Players[0].Position = 99

	_, _ = g.PlayTurn()

	if g.GetCurrentPlayer() != nil {
		t.Fatalf("expected nil after game finishes")
	}
}

func TestGetWinnerBeforeFinish(t *testing.T) {
	g := newTestGame(t)

	if g.GetWinner() != nil {
		t.Fatalf("expected nil winner")
	}
}

func TestGetWinnerAfterFinish(t *testing.T) {
	g := newTestGame(t, 1)

	g.Players[0].Position = 99

	_, _ = g.PlayTurn()

	winner := g.GetWinner()

	if winner == nil {
		t.Fatalf("expected winner")
	}

	if winner.Name != "Alice" {
		t.Fatalf("expected Alice")
	}
}

func TestGetStatus(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	if g.GetStatus() != GameNotStarted {
		t.Fatalf("expected NOT_STARTED")
	}

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.Start()

	if g.GetStatus() != GameInProgress {
		t.Fatalf("expected IN_PROGRESS")
	}
}

// ============================================================
// Multiple players
// ============================================================

func TestThreePlayerRotation(t *testing.T) {
	g := NewSnakesLadders().(*SnakeLadders)

	_ = g.AddPlayer("Alice")
	_ = g.AddPlayer("Bob")
	_ = g.AddPlayer("Charlie")
	_ = g.Start()

	rolls := []int{1, 2, 3}
	i := 0
	g.rollDice = func() int {
		r := rolls[i]
		i++
		return r
	}

	_, _ = g.PlayTurn()

	if g.GetCurrentPlayer().Name != "Bob" {
		t.Fatalf("expected Bob")
	}

	_, _ = g.PlayTurn()

	if g.GetCurrentPlayer().Name != "Charlie" {
		t.Fatalf("expected Charlie")
	}

	_, _ = g.PlayTurn()

	if g.GetCurrentPlayer().Name != "Alice" {
		t.Fatalf("expected Alice")
	}
}

func TestOtherPlayersRemainUnchanged(t *testing.T) {
	g := newTestGame(t, 4)

	_, _ = g.PlayTurn()

	if g.Players[0].Position != 4 {
		t.Fatalf("Alice should be at 4")
	}

	if g.Players[1].Position != 0 {
		t.Fatalf("Bob should remain at 0")
	}
}

// ============================================================
// TurnResult
// ============================================================

func TestTurnResultContainsActualPlayerPointer(t *testing.T) {
	g := newTestGame(t, 4)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.Player != &g.Players[0] {
		t.Fatalf("expected pointer to actual Alice")
	}
}

func TestTurnResultDiceRoll(t *testing.T) {
	g := newTestGame(t, 5)

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.DiceRoll != 5 {
		t.Fatalf("expected dice roll 5")
	}
}

func TestTripleSixResultDoesNotMovePlayer(t *testing.T) {
	g := newTestGame(t, 6, 6, 6)

	g.Players[0].Position = 10

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()
	result, _ := g.PlayTurn()

	if result.FromPosition != 10 {
		t.Fatalf("expected FromPosition 10")
	}

	if result.ToPosition != 10 {
		t.Fatalf("expected ToPosition 10")
	}
}

// ============================================================
// State reset between players
// ============================================================

func TestTripleSixDoesNotLeakToNextPlayer(t *testing.T) {
	g := newTestGame(t, 6, 6, 6, 3)

	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()
	_, _ = g.PlayTurn()

	if g.CurrentTurn != 1 {
		t.Fatalf("expected Bob's turn")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatalf("expected six counter reset")
	}

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.Player.Name != "Bob" {
		t.Fatalf("expected Bob")
	}

	if result.ToPosition != 3 {
		t.Fatalf("expected Bob at 3")
	}
}

func TestOvershootDoesNotLeakSixesToNextPlayer(t *testing.T) {
	g := newTestGame(t, 6, 6, 3)

	g.Players[0].Position = 90

	// First six stored.
	_, _ = g.PlayTurn()

	// Second six immediately overshoots in current implementation.
	_, err := g.PlayTurn()
	if err == nil {
		t.Fatalf("expected overshoot")
	}

	if g.CurrentTurn != 1 {
		t.Fatalf("expected Bob's turn")
	}

	if g.ConsecutiveSixes != 0 {
		t.Fatalf("expected six counter reset")
	}

	result, err := g.PlayTurn()
	if err != nil {
		t.Fatal(err)
	}

	if result.Player.Name != "Bob" {
		t.Fatalf("expected Bob")
	}

	if result.ToPosition != 3 {
		t.Fatalf("expected Bob at 3")
	}
}

// ============================================================
// diceGenerator
// ============================================================

func TestDiceGeneratorAlwaysReturnsOneThroughSix(t *testing.T) {
	for i := 0; i < 10000; i++ {
		dice := diceGenerator()

		if dice < 1 || dice > 6 {
			t.Fatalf("invalid dice value: %d", dice)
		}
	}
}
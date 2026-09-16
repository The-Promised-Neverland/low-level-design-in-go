package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

type GameStatus string

const (
	GameNotStarted GameStatus = "NOT_STARTED"
	GameInProgress GameStatus = "IN_PROGRESS"
	GameFinished   GameStatus = "FINISHED"
)

type Player struct {
	ID       string
	Name     string
	Position int
}

type TurnResult struct {
	Player       *Player
	DiceRoll     int
	FromPosition int
	ToPosition   int
}

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

type SnakeLadders struct {
	Players          []Player
	Snakes           []int
	Ladders          []int
	CurrentTurn      int
	ConsecutiveSixes int
	Status           GameStatus
	Winner           *Player
	rollDice         func() int
}

func NewSnakesLadders() Game {
	return &SnakeLadders{
		Players:          make([]Player, 0),
		Snakes:           make([]int, 101), // index tells the head, value tells the tail
		Ladders:          make([]int, 101), // index tells the bottom, value tells where to climb
		CurrentTurn:      0,
		ConsecutiveSixes: 0,
		Status:           GameNotStarted,
		rollDice:         diceGenerator,
	}
}

func generateRandomID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	id := make([]byte, 6)
	for i := range id {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		id[i] = chars[n.Int64()]
	}
	return string(id)
}

func diceGenerator() int {
	n, _ := rand.Int(rand.Reader, big.NewInt(6))
	return int(n.Int64()) + 1
}

func (g *SnakeLadders) AddPlayer(name string) error {
	if g.Status != GameNotStarted {
		return fmt.Errorf("Game Status: %s", g.Status)
	}
	player := Player{
		ID:       generateRandomID(),
		Name:     name,
		Position: 0,
	}
	g.Players = append(g.Players, player)
	return nil
}

func (g *SnakeLadders) AddSnake(head int, tail int) error {
	if g.Status != GameNotStarted {
		return fmt.Errorf("Game Status: %s", g.Status)
	}
	if head <= 0 || head >= 100 || tail <= 0 || tail >= head {
		return errors.New("invalid snake positions")
	}
	if g.Snakes[head] != 0 {
		return errors.New("Already snake on this point")
	}
	if g.Ladders[head] != 0 {
		return errors.New("Already a ladder starting at this point")
	}
	g.Snakes[head] = tail
	return nil
}

func (g *SnakeLadders) AddLadder(start int, end int) error {
	if g.Status != GameNotStarted {
		return fmt.Errorf("Game Status: %s", g.Status)
	}
	if start <= 0 || start >= 100 || end <= start || end > 100 {
		return errors.New("invalid ladder positions")
	}
	if g.Ladders[start] != 0 {
		return errors.New("Already ladder on this point")
	}
	if g.Snakes[start] != 0 {
		return errors.New("Already a snake head at this point")
	}
	g.Ladders[start] = end
	return nil
}

func (g *SnakeLadders) Start() error {
	if g.Status != GameNotStarted {
		return fmt.Errorf("Game Status: %s", g.Status)
	}
	if len(g.Players) < 2 {
		return errors.New("Need atleast two players")
	}
	g.Status = GameInProgress
	return nil
}

func (g *SnakeLadders) PlayTurn() (*TurnResult, error) {
	if g.Status != GameInProgress {
		return nil, fmt.Errorf("Game Status: %s", g.Status)
	}
	dice := g.rollDice()
	currentPlayer := &g.Players[g.CurrentTurn]
	TurnResult := &TurnResult{
		Player:       currentPlayer,
		DiceRoll:     dice,
		FromPosition: currentPlayer.Position,
		ToPosition:   currentPlayer.Position,
	}
	if dice == 6 {
		if g.ConsecutiveSixes == 2 {
			g.ConsecutiveSixes = 0
			g.CurrentTurn = (g.CurrentTurn + 1) % len(g.Players)
			return TurnResult, errors.New("Consecutive three sixes. Turn forfeited..")
		} else {
			g.ConsecutiveSixes++
			return TurnResult, errors.New("Six. Another turn granted")
		}
	}
	toPosition := 6*g.ConsecutiveSixes + dice + currentPlayer.Position
	g.ConsecutiveSixes = 0
	if toPosition > 100 {
		g.CurrentTurn = (g.CurrentTurn + 1) % len(g.Players)
		return TurnResult, errors.New("Overshooting. Turn forfieted")
	}
	// check if snake at this index
	if g.Snakes[toPosition] != 0 {
		toPosition = g.Snakes[toPosition]
	}
	// check if ladder at this index
	if g.Ladders[toPosition] != 0 {
		toPosition = g.Ladders[toPosition]
	}
	TurnResult.ToPosition = toPosition
	currentPlayer.Position = toPosition
	if TurnResult.ToPosition == 100 {
		g.Winner = currentPlayer
		g.Status = GameFinished
	} else {
		g.CurrentTurn = (g.CurrentTurn + 1) % len(g.Players)
	}
	return TurnResult, nil
}

func (g *SnakeLadders) GetCurrentPlayer() *Player {
	if g.Status != GameInProgress {
		return nil
	}
	return &g.Players[g.CurrentTurn]
}

func (g *SnakeLadders) GetWinner() *Player {
	if g.GetStatus() == GameFinished {
		return g.Winner
	}
	return nil
}

func (g *SnakeLadders) GetStatus() GameStatus {
	return g.Status
}

func main() {
}

package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
)

type User struct {
	UserID string
	Name   string
}

// ---------------------------------------------------------------------------
type Group struct {
	GroupID      string
	Name         string
	Participants map[string]*User
}

func (g *Group) AddMember(user *User) {
	g.Participants[user.UserID] = user
}

func (g *Group) RemoveMember(userID string) {
	delete(g.Participants, userID)
}

func (g *Group) View() []*User {
	users := make([]*User, 0)
	for _, value := range g.Participants {
		users = append(users, value)
	}
	return users
}

// ---------------------------------------------------------------------------
type Expense struct {
	ExpenseID    string
	Amount       float64
	Description  string
	GroupID      string
	Payers       map[string]float64
	Participants []string
	Strategy     SplitStrategy
	PerUserShare map[string]float64
}

func (e *Expense) CalculateSplit() (map[string]float64, error) {
	return e.Strategy.Split(e.Amount, e.Participants)
}

type SplitStrategy interface {
	Split(amount float64, participants []string) (map[string]float64, error)
	Type() string
}

type EqualSplit struct{}

func (e *EqualSplit) Type() string {
	return "EQUAL"
}

func (e *EqualSplit) Split(amount float64, participants []string) (map[string]float64, error) {
	splitShare := make(map[string]float64)
	totalMembers := len(participants)
	equalShare := amount / float64(totalMembers)
	for _, participant := range participants {
		splitShare[participant] = equalShare
	}
	return splitShare, nil
}

type SplitByAmount struct {
	Amounts map[string]float64
}

func (e *SplitByAmount) Split(amount float64, participants []string) (map[string]float64, error) {
	if len(participants) == 0 {
		return nil, fmt.Errorf("participants cannot be empty")
	}
	if len(e.Amounts) != len(participants) {
		return nil, fmt.Errorf("amounts must be provided for exactly the participants")
	}
	total := 0.0
	for _, userID := range participants {
		share, exists := e.Amounts[userID]
		if !exists {
			return nil, fmt.Errorf("share missing for participant %s", userID)
		}
		if share < 0 {
			return nil, fmt.Errorf("share cannot be negative for participant %s", userID)
		}
		total += share
	}
	if math.Abs(total-amount) > 0.001 {
		return nil, fmt.Errorf("split amounts must sum to the expense amount")
	}
	result := make(map[string]float64, len(participants))
	for _, userID := range participants {
		result[userID] = e.Amounts[userID]
	}
	return result, nil
}

func (e *SplitByAmount) Type() string {
	return "AMOUNT"
}

type SplitByPercentage struct {
	Percentages map[string]float64
}

func (e *SplitByPercentage) Split(amount float64, participants []string) (map[string]float64, error) {
	if len(participants) == 0 {
		return nil, fmt.Errorf("participants cannot be empty")
	}
	if len(e.Percentages) != len(participants) {
		return nil, fmt.Errorf("share must be provided for exactly the participants")
	}
	total := 0.0
	for _, userID := range participants {
		percent, exists := e.Percentages[userID]
		if !exists {
			return nil, fmt.Errorf("share missing for participant %s", userID)
		}
		if percent < 0 {
			return nil, fmt.Errorf("share cannot be negative for participant %s", userID)
		}
		total += (percent / 100.0 * amount)
	}
	if math.Abs(total-amount) > 0.001 {
		return nil, fmt.Errorf("split amounts must sum to the expense amount")
	}
	result := make(map[string]float64, len(participants))
	for _, userID := range participants {
		result[userID] = e.Percentages[userID]
	}
	return result, nil
}

func (e *SplitByPercentage) Type() string {
	return "PERCENTAGE"
}

type Splitwise interface {
	CreateUser(name string) (*User, error)
	CreateGroup(name string) (*Group, error)
	AddMember(groupID, userID string) error
	RemoveMember(groupID, userID string) error
	GetGroupMembers(groupID string) ([]*User, error)
	AddExpense(expense *Expense) error
	GetBalance(userID int) (map[int]float64, error)
	GetGroupBalances(groupID int) (map[int]float64, error)
	SettleDebt(fromUserID, toUserID int, amount float64) error
	SimplifyDebts() error
}

type splitwiseService struct {
	Users    map[string]*User
	Groups   map[string]*Group
	Expenses map[string]*Expense
}

func NewSplitwiseService() Splitwise {
	return &splitwiseService{
		Users:    make(map[string]*User),
		Groups:   make(map[string]*Group),
		Expenses: make(map[string]*Expense),
	}
}

func generateID() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

func (s *splitwiseService) CreateUser(name string) (*User, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("Please retry later...")
	}
	user := &User{
		UserID: id,
		Name:   name,
	}
	s.Users[user.UserID] = user
	return user, nil
}

func (s *splitwiseService) CreateGroup(name string) (*Group, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("Please retry later...")
	}
	group := &Group{
		GroupID:      id,
		Name:         name,
		Participants: make(map[string]*User),
	}
	s.Groups[group.GroupID] = group
	return group, nil
}

func (s *splitwiseService) checkIfExists(groupID, userID string) error {
	if _, exists := s.Groups[groupID]; !exists {
		return fmt.Errorf("No such group exists. Please create the group first")
	}
	if _, exists := s.Users[userID]; !exists {
		return fmt.Errorf("No such user exists. Please create the user first")
	}
	return nil
}

func (s *splitwiseService) AddMember(groupID, userID string) error {
	if err := s.checkIfExists(groupID, userID); err != nil {
		return err
	}
	user := s.Users[userID]
	group := s.Groups[groupID]
	group.AddMember(user)
	return nil
}

func (s *splitwiseService) RemoveMember(groupID, userID string) error {
	if err := s.checkIfExists(groupID, userID); err != nil {
		return err
	}
	user := s.Users[userID].UserID
	group := s.Groups[groupID]
	group.RemoveMember(user)
	return nil
}

func (s *splitwiseService) GetGroupMembers(groupID string) ([]*User, error) {
	if _, exists := s.Groups[groupID]; !exists {
		return nil, fmt.Errorf("No such group exists. Please create the group first")
	}
	group := s.Groups[groupID]
	users := group.View()
	return users, nil
}

func (s *splitwiseService) validateGroupExpense(expense *Expense) error {
	group, exists := s.Groups[expense.GroupID]
	if !exists {
		return fmt.Errorf("group not found: %s", expense.GroupID)
	}
	for _, userID := range expense.Participants {
		if _, exists := group.Participants[userID]; !exists {
			return fmt.Errorf(
				"user %s is not a member of group %s",
				userID,
				expense.GroupID,
			)
		}
	}
	return nil
}

func (s *splitwiseService) validateIndividualExpense(expense *Expense) error {
	seen := make(map[string]bool)
	for _, userID := range expense.Participants {
		if _, exists := s.Users[userID]; !exists {
			return fmt.Errorf("participant not found: %s", userID)
		}
		if seen[userID] {
			return fmt.Errorf("duplicate participant: %s", userID)
		}
		seen[userID] = true
	}
	return nil
}

func (s *splitwiseService) validatePayers(expense *Expense) error {
	var totalPaid float64
	for userID, amount := range expense.Payers {
		if _, exists := s.Users[userID]; !exists {
			return fmt.Errorf("payer not found: %s", userID)
		}
		if amount <= 0 {
			return fmt.Errorf("payer contribution must be positive: %s", userID)
		}
		totalPaid += amount
	}
	if math.Abs(totalPaid-expense.Amount) > 0.001 {
		return fmt.Errorf(
			"total payer contributions %.2f do not match expense amount %.2f",
			totalPaid,
			expense.Amount,
		)
	}
	return nil
}

func (s *splitwiseService) saveExpense(expense *Expense) error {
	if expense.ExpenseID == "" {
		for {
			id, err := generateID()
			if err != nil {
				return fmt.Errorf("failed to generate expense ID: %w", err)
			}
			if _, exists := s.Expenses[id]; !exists {
				expense.ExpenseID = id
				break
			}
		}
	} else {
		if _, exists := s.Expenses[expense.ExpenseID]; exists {
			return fmt.Errorf("expense ID already exists: %s", expense.ExpenseID)
		}
	}
	s.Expenses[expense.ExpenseID] = expense
	return nil
}

func (s *splitwiseService) AddExpense(expense *Expense) error {
	if expense == nil {
		return fmt.Errorf("expense cannot be nil")
	}
	if expense.Amount <= 0 {
		return fmt.Errorf("expense amount must be positive")
	}
	if expense.Strategy == nil {
		return fmt.Errorf("split strategy is required")
	}
	seen := make(map[string]bool)
	for _, userID := range expense.Participants {
		if _, exists := s.Users[userID]; !exists {
			return fmt.Errorf("participant not found: %s", userID)
		}
		if seen[userID] {
			return fmt.Errorf("duplicate participant: %s", userID)
		}
		seen[userID] = true
	}
	// it is a group expense and all participants shall belong to same group
	if expense.GroupID != "" {
		if err := s.validateGroupExpense(expense); err != nil {
			return err
		}
	}
	// We need to validate payer.
	// For example: P1,P2,P3 are from a group. but the final bill was paid off by P5, an individual who may or may not be from the group
	if err := s.validatePayers(expense); err != nil {
		return err
	}
	shares, err := expense.CalculateSplit()
	if err != nil {
		return err
	}
	expense.PerUserShare = shares
	return s.saveExpense(expense)
}

func (s *splitwiseService) GetBalance(userID int) (map[int]float64, error) {
	return nil, nil
}

func (s *splitwiseService) GetGroupBalances(groupID int) (map[int]float64, error) {
	return nil, nil
}

func (s *splitwiseService) SettleDebt(fromUserID, toUserID int, amount float64) error {
	return nil
}

func (s *splitwiseService) SimplifyDebts() error {
	return nil
}

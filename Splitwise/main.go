package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"
)

type UserGroupBalance struct {
	UserID   string
	Name     string
	IsMember bool
	Balance  float64
}

type AddExpenseRequest struct {
	Amount       float64
	Description  string
	Payers       map[string]float64
	Participants []string
	Strategy     SplitStrategy
}

// -----------------------------------------------------------------------
type User struct {
	UserID string
	Name   string
}

// ---------------------------------------------------------------------------
type Group struct {
	GroupID string
	Name    string
	Members map[string]*User
}

func (g *Group) AddMember(user *User) {
	g.Members[user.UserID] = user
}

func (g *Group) RemoveMember(userID string) {
	delete(g.Members, userID)
}

func (g *Group) View() []*User {
	users := make([]*User, 0)
	for _, value := range g.Members {
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
		result[userID] = amount * e.Percentages[userID] / 100.0
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
	AddGroupExpense(groupID string, req AddExpenseRequest) error
	AddIndividualExpense(req AddExpenseRequest) error
	GetBalance(userID string) (float64, error)
	GetGroupBalances(groupID string) ([]UserGroupBalance, error)
	SettleDebt(fromUserID, toUserID int, amount float64) error
	SimplifyDebts() error
}

type LedgerEntry struct {
	EntryID   string
	ExpenseID string
	GroupID   string // Empty for personal expenses
	UserID    string
	Type      LedgerEntryType
	Amount    float64
	CreatedAt time.Time
}

type LedgerEntryType string

const (
	Paid       LedgerEntryType = "PAID"
	Owed       LedgerEntryType = "OWED"
	Settlement LedgerEntryType = "SETTLEMENT"
)

type SettlementRecord struct {
	SettlementID string
	GroupID      string
	FromUserID   string
	ToUserID     string
	Amount       float64
	CreatedAt    time.Time
}

type splitwiseService struct {
	Users       map[string]*User    // key is userID
	Groups      map[string]*Group   // key is groupID
	Expenses    map[string]*Expense // key is expenseID
	Ledger      map[string]*LedgerEntry
	UserLedger  map[string][]string // userID -> ledger entry IDs
	GroupLedger map[string][]string // groupID -> ledger entry IDs
}

func NewSplitwiseService() Splitwise {
	return &splitwiseService{
		Users:       make(map[string]*User),
		Groups:      make(map[string]*Group),
		Expenses:    make(map[string]*Expense),
		Ledger:      make(map[string]*LedgerEntry),
		UserLedger:  make(map[string][]string),
		GroupLedger: make(map[string][]string),
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
		GroupID: id,
		Name:    name,
		Members: make(map[string]*User),
	}
	s.Groups[group.GroupID] = group
	return group, nil
}

func (s *splitwiseService) AddMember(groupID, userID string) error {
	if _, exists := s.Groups[groupID]; !exists {
		return fmt.Errorf("No such group exists. Please create the group first")
	}
	if _, exists := s.Users[userID]; !exists {
		return fmt.Errorf("No such user exists. Please create the user first")
	}
	user := s.Users[userID]
	group := s.Groups[groupID]
	group.AddMember(user)
	return nil
}

func (s *splitwiseService) RemoveMember(groupID, userID string) error {
	if _, exists := s.Groups[groupID]; !exists {
		return fmt.Errorf("No such group exists. Please create the group first")
	}
	if _, exists := s.Users[userID]; !exists {
		return fmt.Errorf("No such user exists. Please create the user first")
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

func (s *splitwiseService) AddGroupExpense(groupID string, req AddExpenseRequest) error {
	if req.Amount <= 0 {
		return fmt.Errorf("Expense amount cannot be negative or zero")
	}
	if req.Strategy == nil {
		return fmt.Errorf("Strategy cannot be nil")
	}
	if _, exists := s.Groups[groupID]; !exists {
		return fmt.Errorf("GroupID does not exists")
	}
	return nil
}

func (s *splitwiseService) AddIndividualExpense(req AddExpenseRequest) error {
	if req.Amount <= 0 {
		return fmt.Errorf("Expense amount cannot be negative or zero")
	}
	if req.Strategy == nil {
		return fmt.Errorf("Strategy cannot be nil")
	}
	return nil
}

func (s *splitwiseService) GetBalance(userID string) (float64, error) {
	if _, exists := s.Users[userID]; !exists {
		return 0.0, fmt.Errorf("User does not exits")
	}
	balance := 0.0
	for _, expense := range s.Expenses {
		balance += expense.Payers[userID]       // Will get this amount back
		balance -= expense.PerUserShare[userID] // Need to pay back
	}
	return balance, nil
}

func (s *splitwiseService) GetGroupBalances(groupID string) ([]UserGroupBalance, error) {
	if _, exists := s.Groups[groupID]; !exists {
		return nil, fmt.Errorf("Group does not exits")
	}
	balances := make(map[string]float64)
	for _, expense := range s.Expenses {
		if expense.GroupID != groupID {
			continue
		}
		for userID, amountPaid := range expense.Payers {
			balances[userID] += amountPaid
		}
		for userID, share := range expense.PerUserShare {
			balances[userID] -= share
		}
	}
	result := make([]UserGroupBalance, 0, len(balances))
	group := s.Groups[groupID]
	for userID, balance := range balances {
		user, exists := s.Users[userID]
		if !exists {
			return nil, fmt.Errorf("user not found: %s", userID)
		}
		_, isMember := group.Members[userID]
		result = append(result, UserGroupBalance{
			UserID:   userID,
			Name:     user.Name,
			IsMember: isMember,
			Balance:  balance,
		})
	}
	return result, nil
}

func (s *splitwiseService) SettleDebt(fromUserID, toUserID int, amount float64) error {
	return nil
}

func (s *splitwiseService) SimplifyDebts() error {
	return nil
}

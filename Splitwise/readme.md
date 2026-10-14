# Splitwise — Expense Sharing System

## Problem Statement

Design and implement an expense-sharing system that allows users to record shared expenses, track who owes whom, and settle outstanding balances.

The system must support expenses between individuals and expenses shared within groups.

## 1. Users

Each user has:
- A unique ID
- A name

Users can participate in multiple groups and expenses.

## 2. Groups

The system must support:
- Creating a group
- Adding users to a group
- Removing users from a group
- Viewing group members

An expense may belong to a group or be shared directly among individual users.

## 3. Expenses

Each expense has:
- A description
- A total amount
- One or more payers
- A set of participants
- A split describing each participant's share

The system must support three split types.

### Equal Split

Divide the total expense equally among all participants.

**Example:** A ₹1,200 dinner shared by Alice, Bob, and Charlie gives each person a share of ₹400.

### Exact-Amount Split

Specify each participant's share directly.

**Example:**
- Alice: ₹500
- Bob: ₹400
- Charlie: ₹300

The shares must add up to the total expense.

### Percentage Split

Specify each participant's share as a percentage of the total.

**Example:**
- Alice: 50%
- Bob: 30%
- Charlie: 20%

The percentages must add up to 100%.

## 4. Multiple Payers

An expense may be paid for by multiple users.

**Example:** For a ₹1,200 dinner, Alice pays ₹700 and Bob pays ₹500.

The system must account for both the amounts paid and each participant's share. The total contributions from all payers must equal the total expense.

## 5. Balances

The system must track net balances between users and provide ways to view:

- How much one user owes another
- Who owes money to a particular user
- A user's total outstanding debt
- The total amount owed to a user
- Balances within a group

Balances must reflect all recorded expenses and settlements.

## 6. Settlements

Users can record payments made toward outstanding balances.

**Example:**

Alice owes Bob ₹500. Alice pays Bob ₹300.

The remaining balance is ₹200 owed by Alice to Bob.

Recording a settlement must update the outstanding balances accordingly.

## 7. Debt Simplification

The system must offer an option to simplify outstanding debts by reducing unnecessary intermediate payments while preserving each user's net amount owed or receivable.

**Example:**

Before:
- Alice owes Bob ₹500.
- Bob owes Charlie ₹500.

After simplification:
- Alice owes Charlie ₹500.

## 8. Example Scenario

A group contains Alice, Bob, and Charlie.

### Expense 1: Dinner

Alice pays ₹1,200 for dinner, split equally among all three participants.

Each person's share is ₹400.

Since Alice paid the full amount:
- Bob owes Alice ₹400.
- Charlie owes Alice ₹400.

### Expense 2: Cab

Bob pays ₹600 for a cab shared equally between Alice and Bob.

Each person's share is ₹300.

Since Bob paid the full amount:
- Alice owes Bob ₹300.

The system must calculate the resulting net balances between all users, accounting for both expenses.

## 9. Functional Requirements

The system must support:

1. Creating users.
2. Creating groups.
3. Adding and removing group members.
4. Recording expenses.
5. Equal, exact-amount, and percentage splits.
6. Expenses with multiple payers.
7. Viewing user balances.
8. Viewing group balances.
9. Recording settlements.
10. Simplifying outstanding debts.

## 10. Validation and Correctness

The system should reject invalid operations, including:

- Negative or zero expense amounts.
- Exact splits that do not add up to the total expense.
- Percentage splits that do not add up to 100%.
- Invalid users, payers, or participants.
- Payer contributions that do not add up to the total expense.
- Invalid group membership for a group expense.
- Settlements exceeding the amount owed, unless overpayments are explicitly supported.

Monetary calculations must preserve exact amounts. Rounding rules must be applied consistently when a split cannot be divided evenly.

## 11. Concurrency

The system must be safe to use when multiple users record expenses, query balances, or record settlements concurrently.

Concurrent operations must not cause incorrect balances, inconsistent expense records, or duplicate settlement processing.
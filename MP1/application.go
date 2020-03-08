package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var balances map[string]uint64

func update_balances(m BankMessage) {
	transactionType, account1, account2, amount := parse_transaction(m.Transaction)
	if transactionType == "DEPOSIT" {
		balances[account1] += amount
	} else if transactionType == "TRANSFER" {
		balances[account1] -= amount
		balances[account2] += amount
	}

}

func parse_transaction(transaction string) (string, string, string, uint64) {
	//return format: transaction type (deposit, withdrawl, etc), account1, account2, amount
	t := strings.Split(transaction, " ")
	transactionType := t[0]
	account1 := t[1]
	var account2 string
	var amount uint64
	var err error
	if t[0] == "DEPOSIT" {
		account2 = ""
		amount, err = strconv.ParseUint(t[2], 10, 64) // arguments: string, base, bitSize
		check(err)
	} else if t[0] == "TRANSFER" {
		account2 = t[3]
		amount, err = strconv.ParseUint(t[4], 10, 64)
		check(err)
	} else {
		panic("transactionType was neither DEPOSIT nor TRANSFER")
	}
	return transactionType, account1, account2, amount
}

func print_balances() {
	balances = make(map[string]uint64)
	for {
		time.Sleep(5 * time.Second)
		fmt.Print("\nBALANCES  ")
		for k, v := range balances {
			fmt.Print("%v:%v ", k, v)
		}
	}
}

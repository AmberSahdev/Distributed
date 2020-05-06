package main

import (
	"fmt"
	"net"
	"runtime"
	"sync"
)

type ClientID int

type clientNode struct {
	conn   net.Conn
	inbox  chan string // channel to receive Messages received from neighbor
	outbox chan string // channel to send Messages to neighbor
}

type AccountLock struct {
	localLock          sync.Mutex
	numReaders         int
	isWriteLocked      bool
	isPendingWriteLock bool
}

type Account struct {
	Balance int
	Lock    AccountLock
}

func (curLock *AccountLock) Lock() {
	setPendingWriteLock := false
	for {
		curLock.localLock.Lock()
		if curLock.isPendingWriteLock == false {
			curLock.isPendingWriteLock = true
			setPendingWriteLock = true
		}
		if setPendingWriteLock && curLock.numReaders == 0 && !curLock.isWriteLocked {
			curLock.numReaders = 1
			curLock.isWriteLocked = true
			curLock.isPendingWriteLock = false
			// got the write lock, stop blocking
			break
		}
		curLock.localLock.Unlock()
		runtime.Gosched()
	}
	curLock.localLock.Unlock()
}

func (curLock *AccountLock) Unlock() {
	curLock.localLock.Lock()
	curLock.numReaders = 0
	curLock.isWriteLocked = false
	curLock.localLock.Unlock()
}
func (curLock *AccountLock) RLock() {
	for {
		curLock.localLock.Lock()
		if !curLock.isWriteLocked && !curLock.isPendingWriteLock {
			curLock.numReaders += 1
			// got the read lock, stop blocking
			break
		}
		curLock.localLock.Unlock()
		runtime.Gosched()
	}
}

func (curLock *AccountLock) PromoteLock() {
	setPendingWriteLock := false
	for {
		curLock.localLock.Lock()
		if curLock.isPendingWriteLock == false {
			curLock.isPendingWriteLock = true
			setPendingWriteLock = true
		}
		if setPendingWriteLock && curLock.numReaders == 1 && !curLock.isWriteLocked {
			curLock.isWriteLocked = true
			curLock.isPendingWriteLock = false
			// got the write lock, stop blocking
			break
		}
		curLock.localLock.Unlock()
		runtime.Gosched()
	}
	curLock.localLock.Unlock()
}

func (curLock *AccountLock) RUnlock() {
	curLock.localLock.Lock()
	if curLock.numReaders == 0 {
		Error.Println("Negative Number Readers!")
	}
	curLock.numReaders -= 1
	curLock.localLock.Unlock()
}

func check(e error) {
	if e != nil {
		fmt.Println("Error Detected:")
		panic(e)
	}
}

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

var curLongestChainLeaf *Block

func blockchain() {
	go handleBlockchainServerVerifies()
	curLongestChainLeaf = <-localVerifiedBlocks
	go startNewMine(curLongestChainLeaf)
	for newValidBlock := range localVerifiedBlocks {
		if newValidBlock.ParentBlockID == curLongestChainLeaf.BlockID {
			// Add this block to longest chain, update pending transactions
			curLongestChainLeafMutex.Lock()
			curLongestChainLeaf = newValidBlock
			curLongestChainLeafMutex.Unlock()
			Info.Println("New Longest Chain Leaf received w/ ID:", hex.EncodeToString(curLongestChainLeaf.BlockID[:]), "current height: ", curLongestChainLeaf.BlockHeight)
			processedTransactionMutex.Lock()
			addTransactionsToProcessedSet(newValidBlock.Transactions)
			processedTransactionMutex.Unlock()
			transactionMutex.Lock()
			deleteDuplicateTransactions(newValidBlock.Transactions)
			transactionMutex.Unlock()
			go startNewMine(curLongestChainLeaf)
		} else if newValidBlock.BlockHeight > curLongestChainLeaf.BlockHeight { //new Longest chain!
			// TODO: Find the fork's common ancestor!
			// remove rolled-back processed Transaction from set
			// add rolled forward transactions to set
			blockMutex.RLock()
			commonAncestorBlock := getForkBlock(newValidBlock, curLongestChainLeaf)
			transactionsToRollback := getTransactionsInSegment(curLongestChainLeaf, commonAncestorBlock) // Not including commonAncestorBlock
			transactionsToCommit := getTransactionsInSegment(newValidBlock, commonAncestorBlock)
			blockMutex.RUnlock()
			processedTransactionMutex.Lock()
			removeTransactionsFromProcessedSet(transactionsToRollback)
			addTransactionsToProcessedSet(transactionsToCommit)
			processedTransactionMutex.Unlock()
			transactionMutex.Lock()
			addRolledBackTransactions(transactionsToRollback)
			deleteDuplicateTransactions(transactionsToCommit)
			transactionMutex.Unlock()
			curLongestChainLeafMutex.Lock()
			curLongestChainLeaf = newValidBlock
			curLongestChainLeafMutex.Unlock()
			Info.Println("New Longest Chain Leaf received w/ ID:", hex.EncodeToString(curLongestChainLeaf.BlockID[:]), "current height:", curLongestChainLeaf.BlockHeight)

		}
	}
}

func startNewMine(parentBlock *Block) {
	// Mine a new Block
	const TransactionWaitPeriod = 100
	for {
		curLongestChainLeafMutex.Lock()
		if parentBlock.BlockID != curLongestChainLeaf.BlockID {
			return // starting mine of new block
		}
		curLongestChainLeafMutex.Unlock()
		transactionMutex.RLock()
		numPendingTransactions := len(transactionList)
		transactionMutex.RUnlock()
		if numPendingTransactions >= MaxTransactionsInBlock {
			break
		}
		time.Sleep(TransactionWaitPeriod * time.Millisecond)
	}
	curLongestChainLeafMutex.Lock()
	if parentBlock.BlockID != curLongestChainLeaf.BlockID {
		return // starting mine of new block
	}
	curLongestChainLeafMutex.Unlock()
	// Have Enough Transactions
	newBlockToMine := new(Block)
	transactionMutex.RLock()
	finalAccounts, newBlockTransactionList := extractValidTransactions(parentBlock)
	transactionMutex.RUnlock()
	*newBlockToMine = Block{
		ParentBlockID:   parentBlock.BlockID,
		Transactions:    newBlockTransactionList,
		AccountBalances: finalAccounts,
		BlockHeight:     parentBlock.BlockHeight + 1,
		BlockID:         BlockID{},
		BlockProof:      BlockPW{},
	}
	newBlockToMine.BlockID = computeBlockID(newBlockToMine)
	curLongestChainLeafMutex.Lock()
	if parentBlock.BlockID != curLongestChainLeaf.BlockID {
		return // starting mine of new block kill this mining operation
	}
	curLongestChainLeafMutex.Unlock()
	tryMineBlock(newBlockToMine)
}

func extractValidTransactions(parentBlock *Block) (map[AccountID]uint64, []TransactionMessage) {
	// TODO: initialize account balances to parent block values
	// operate on
	newAccountBalances := make(map[AccountID]uint64)
	newTransactionList := make([]TransactionMessage, 0)

	newAccountBalances = parentBlock.AccountBalances

	i := 0
	for len(newTransactionList) <= MaxTransactionsInBlock && i < len(transactionList) {
		t := transactionList[i]
		if t.Src == 0 {
			newAccountBalances[t.Dest] += t.Amount
			newTransactionList = append(newTransactionList, *t)
		} else if newAccountBalances[t.Src]-t.Amount >= 0 { // if only the account's balance remainds non negative
			newAccountBalances[t.Src] -= t.Amount
			newAccountBalances[t.Dest] += t.Amount
			newTransactionList = append(newTransactionList, *t)
		}
		i++
	}

	return newAccountBalances, newTransactionList
}

func removeTransactionsFromProcessedSet(oldTransactions []TransactionMessage) {
	for _, transMsg := range oldTransactions {
		delete(processedTransactionSet, transMsg.TransactionID)
	}
}

func addTransactionsToProcessedSet(newTransactions []TransactionMessage) {
	for _, transMsg := range newTransactions {
		processedTransactionSet[transMsg.TransactionID] = empty
	}
}

func handleBlockchainServerVerifies() {
	for newBlockID := range serviceVerifiedBlockIDs {
		blockMutex.Lock()
		blockInf, exists := blockMap[newBlockID]
		if !exists {
			panic("Impossible, Block must exist here")
		}
		parentID := blockList[blockInf.Index].ParentBlockID
		parentInf, exists := blockMap[parentID]
		if !exists {
			panic("Impossible, Block must exist here")
		}
		if parentInf.Verified {
			blockInf.Verified = true
			localVerifiedBlocks <- blockList[blockInf.Index]
			verifyChildDependents(newBlockID)
		} else {
			parentInf.ChildDependents = append(parentInf.ChildDependents, newBlockID)
			Warning.Println("Waiting on Parent of", hex.EncodeToString(newBlockID[:]), "to be verified by the service!")
		}
		blockMutex.Unlock()
	}
}

// recursively verifies children waiting on parent to be verified
func verifyChildDependents(curBlockID BlockID) {
	blockInf, exists := blockMap[curBlockID]
	if !exists {
		panic("Impossible, Block must exist here")
	}
	for _, childBlockID := range blockInf.ChildDependents {
		childBlockInf, exists := blockMap[childBlockID]
		if !exists {
			panic("Impossible, Block must exist here")
		}
		childBlockInf.Verified = true
		localVerifiedBlocks <- blockList[childBlockInf.Index]
		verifyChildDependents(childBlockID)
	}
}

func verifyBlock(curBlock *Block) {
	blockMutex.RLock()
	parentBlockInfo, _ := blockMap[curBlock.ParentBlockID]
	parentBlock := blockList[parentBlockInfo.Index]
	blockMutex.RUnlock()
	if curBlock.BlockID == computeBlockID(curBlock) && curBlock.BlockHeight-1 == parentBlock.BlockHeight {
		if verifyTransactions(curBlock) {
			askVerifyBlock(curBlock)
		} else {
			Warning.Println("Rejected block for invalid blockID hash")
		}
	} else {
		Warning.Println("Rejected block for invalid blockID hash")
	}
}

//TODO: ensures final balances are non-negative and correspond to parent block's balances
func verifyTransactions(curBlock *Block) bool {
	// ensure current block balances are positive
	// Then Do:
	// Step 1, ensure parent exists (it should, otherwise Log Error)
	// Step 2, get parents account state
	// Step 3, ensure after processing transactions you arrive at current block's account balances
	return true
}

func tryMineBlock(curBlock *Block) {
	currentBlockBeingMinedMutex.Lock()
	currentBlockBeingMined = curBlock
	currentBlockBeingMinedMutex.Unlock()
	blockIDstr := hex.EncodeToString(curBlock.BlockID[:])
	Info.Println("Trying to mine Block with ID:", blockIDstr)
	serviceMsg := "SOLVE " + blockIDstr + "\n"
	mp2Service.outbox <- serviceMsg
}

func computeBlockID(curBlock *Block) BlockID {
	dataToHash := make([][]byte, 1)
	dataToHash[0] = curBlock.ParentBlockID[:]
	//Info.Println("dataToHash 1", dataToHash)
	for _, curTrans := range curBlock.Transactions {
		dataToHash = append(dataToHash, curTrans.TransactionID[:])
		// Info.Println("dataToHash 2", dataToHash)
	}
	//Info.Println("dataToHash 2", dataToHash)
	//h := sha256.New()
	//var result [sha256.Size]byte
	//copy(result[:], sha256.Sum256(bytes.Join(dataToHash, nil))[:])
	//Info.Println("dataToHash 3", bytes.Join(dataToHash, nil))
	//Info.Println("dataToHash", sha256.Sum256(bytes.Join(dataToHash, nil)))
	return sha256.Sum256(bytes.Join(dataToHash, nil))
}

func askVerifyBlock(b *Block) {
	hash := hex.EncodeToString(b.BlockID[:])
	proofOfWork := hex.EncodeToString(b.BlockProof[:])

	// removing prefixed 0s
	var i int
	for i = 0; i < len(proofOfWork); i++ {
		if string(proofOfWork[i]) != "0" {
			break
		}
	}
	proofOfWork = proofOfWork[i:]

	serviceMsg := "VERIFY " + hash + " " + proofOfWork + "\n"
	mp2Service.outbox <- serviceMsg
	Info.Println(serviceMsg)
}

func addRolledBackTransactions(Transactions []TransactionMessage) {
	for _, transaction := range Transactions {
		// remove from transactionMap
		addTransaction(transaction, true)
	}
}

func deleteDuplicateTransactions(Transactions []TransactionMessage) {
	// remove from transactionList
	for _, transaction := range Transactions {
		// remove from transactionMap
		ind, exists := transactionMap[transaction.TransactionID]
		if exists {
			transactionList[ind] = nil
			delete(transactionMap, transaction.TransactionID)
		}
	}
	newTransactionList := make([]*TransactionMessage, 0)
	newTransListInd := 0
	for _, val := range transactionList {
		if val != nil {
			transactionMap[val.TransactionID] = newTransListInd
			newTransactionList = append(newTransactionList, val)
			newTransListInd++
		}
	}
	transactionList = newTransactionList
	resetLastSentTransactionIndices()
}

func resetLastSentTransactionIndices() {
	neighborMutex.Lock()
	for _, curNode := range neighborMap {
		curNode.lastSentTransactionIndex = -1
	}
	neighborMutex.Unlock()
}

func getForkBlock(b1 *Block, b2 *Block) *Block {
	var higherBlock *Block // block at higher height
	var lowerBlock *Block  // block at lower height

	if b1.BlockHeight >= b2.BlockHeight {
		higherBlock = b1
		lowerBlock = b2
	} else {
		higherBlock = b2
		lowerBlock = b1
	}

	// Bring both branches to same height
	for higherBlock.BlockHeight != lowerBlock.BlockHeight {
		higherBlock = getParentBlock(higherBlock)
	}

	// now iterate backwards until you find the same blockID
	for higherBlock.BlockID != lowerBlock.BlockID {
		higherBlock = getParentBlock(higherBlock)
		lowerBlock = getParentBlock(higherBlock)
	}
	return higherBlock
}

func getParentBlock(b *Block) *Block {
	parent, exists := blockMap[b.ParentBlockID]
	if !exists {
		panic("getParentBlock")
	}
	return blockList[parent.Index]
}

func getTransactionsInSegment(leaf *Block, fork *Block) []TransactionMessage {
	segmentTransactions := make([]TransactionMessage, 0)
	b := leaf
	for b.BlockID != fork.BlockID {
		for i := len(b.Transactions) - 1; i >= 0; i-- {
			segmentTransactions = append(segmentTransactions, b.Transactions[i])
		}
		b = getParentBlock(b)
	}
	return segmentTransactions
}

func reverseSliceTransactions(s []TransactionMessage) []TransactionMessage {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

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
	for newValidBlock := range localVerifiedBlocks {
		if newValidBlock.ParentBlockID == curLongestChainLeaf.BlockID {
			// Add this block to longest chain, update pending transactions
			curLongestChainLeafMutex.Lock()
			curLongestChainLeaf = newValidBlock
			curLongestChainLeafMutex.Unlock()
			processedTransactionMutex.Lock()
			addTransactionsToProcessedSet(newValidBlock)
			processedTransactionMutex.Unlock()
			transactionMutex.Lock()
			deleteDuplicateTransactions(newValidBlock)
			transactionMutex.Unlock()
			go startNewMine(curLongestChainLeaf)
		} else if newValidBlock.BlockHeight > curLongestChainLeaf.BlockHeight { //new Longest chain!
			panic("Longest Chain Overtaken! Sepuku")
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

func extractValidTransactions(parentBlock *Block) (map[AccountID]uint64, []*TransactionMessage) {
	// TODO: initialize account balances to parent block values
	// operate on
	newAccountBalances := make(map[AccountID]uint64)
	return newAccountBalances, transactionList[:2000]
}

func addTransactionsToProcessedSet(newBlock *Block) {
	for _, transMsg := range newBlock.Transactions {
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
	serviceMsg := "SOLVE " + hex.EncodeToString(curBlock.BlockID[:]) + "\n"
	mp2Service.outbox <- serviceMsg
}

func computeBlockID(curBlock *Block) BlockID {
	dataToHash := make([][]byte, 1)
	dataToHash[0] = curBlock.ParentBlockID[:]
	for _, curTrans := range curBlock.Transactions {
		dataToHash = append(dataToHash, curTrans.TransactionID[:])
	}
	h := sha256.New()
	var result [sha256.Size]byte
	copy(result[:], h.Sum(bytes.Join(dataToHash, nil))[:sha256.Size])
	return result
}

func askVerifyBlock(b *Block) {
	hash := hex.EncodeToString(b.BlockID[:sha256.Size])
	proofOfWork := hex.EncodeToString(b.BlockProof[:sha256.Size])

	serviceMsg := "VERIFY " + hash + " " + proofOfWork + "\n"
	mp2Service.outbox <- serviceMsg
}

func deleteDuplicateTransactions(b *Block) {
	// remove from transactionList
	for _, transaction := range b.Transactions {
		// remove from transactionMap
		ind, exists := transactionMap[transaction.TransactionID]
		if exists {
			transactionList[ind] = nil
			delete(transactionMap, transaction.TransactionID)
		}
	}
	newTransactionList := make([]*TransactionMessage, 0)
	for _, val := range transactionList {
		if val != nil {
			newTransactionList = append(newTransactionList, val)
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

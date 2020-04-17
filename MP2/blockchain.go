package main

import (
	"crypto/sha256"
	"fmt"
)

func blockchain() {
	// ask neighbors about their blockchain status,
	//    everyone sends out their height
	// ask the node with the highest height for their whole blockchain
	// OR the current state of ledger (Not rec)

	// once you have the whole blockchain, check if any duplicate transactionID in transactionList you have accrued while verifying that block hashes are correct
	// if duplicate, delete corresponding transactionID in transactionList
	//    and reset every node's lastSentTransactionIndex to 0
	//

	// CONCERN: you delete a block from your transactionList but then a neighbor you sent that block to now sends it back to you.
	//

	// send block to every neighbor with node.blockIndex < height
	// CONCERN: if you every go back and change the branch of blockchain you're on, you need to reset every node.BlockIndex to the reset height

	// block sending logic
	//   when you receive a new block, resend it to your neighbors
	//   if len(transactionList) > 2000, mine a new block, send it to your neighbors
	//      if you receive a new block while minig for another, stop mining

	// block receiving logic
	// delete duplicate transactionID in transactionList
	// if you receive a block from service
	//     verify hash with service (requirement)
	// if you recieve a block from neighbor
	//    if block.height > localBlockHeight
	//       ask your neighbor for previous blocks until your height is the same
	//          AND blockID (previous block hash) is the same
	//            i.e. you're on the same blockchain branch
	//    propogate new blocks to your neighbors
}

// TODO abolish all Sprintfs
func askProofOfWork(b *Block) {
	h := sha256.New()
	hash := fmt.Sprintf("%x", h.Sum(nil))
	serviceMsg := "SOLVE " + hash + "\n"
	mp2Service.outbox <- serviceMsg
}

func askVerifyBlock(b *Block) {
	// remove the proof of work
	bNew := new(Block)
	*bNew = *b
	bNew.BlockID = BlockID{}

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", bNew)))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	proofOfWork := fmt.Sprintf("%x", b.BlockID) // hex.EncodeToString([]byte(b.BlockID))

	serviceMsg := "VERIFY " + hash + " " + proofOfWork + "\n"
	mp2Service.outbox <- serviceMsg
}

func deleteDuplicateTransactions(b *Block) {
	transactionMutex.Lock()
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
	transactionMutex.Unlock()
}

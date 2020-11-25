package main

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"go.mongodb.org/mongo-driver/bson"
)

func subscribeToHead() {
	logger.Info("starting blockchain head subscription..")
	headers := make(chan *types.Header)
	sub, err := ethClient.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		logger.Fatalf("error while subscribing to blockchain head: %v", err)
	}
	for {
		select {
		case err := <-sub.Err():
			logger.Fatalf("error during blockchain head subscription: %v", err)
		case header := <-headers:
			logger.Infof("new block received, hash: %s", header.Hash().String())
			block, err := ethClient.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				logger.Fatalf("error while retrieving block by hash %s: %v", header.Hash(), err)
			}
			logs, err := ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
				FromBlock: block.Number(),
				ToBlock:   block.Number(),
			})
			if err != nil {
				logger.Fatalf("error while retrieving logs by block number %d: %v", block.Number().Int64(), err)
			}
			// Iterate on all the logs found to create the mongo log
			storeLogs(logs)
		}
	}
}

func syncLogs() {
	collection := mongoDatabase.Collection("logLatestBlockRead")
	blockNumber, err := ethClient.BlockNumber(context.TODO())
	if err != nil {
		logger.Errorf("error while retrieving block number: %v", err)
		return
	}
	if len(Configuration.Contracts) == 0 {
		logger.Infof("no contracts configured, retrieving all the logs.")
		var serverLatestBlockRead ServerLatestBlockRead
		// Check latest block read
		res := collection.FindOne(context.Background(), bson.M{"_id": 0})
		if res.Err() != nil {
			err := res.Decode(&serverLatestBlockRead)
			if err != nil {
				serverLatestBlockRead = ServerLatestBlockRead{BlockNumber: Configuration.Server.FromBlock}
			}
		}
		// Build the filter query
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(Configuration.Server.FromBlock)),
			ToBlock:   big.NewInt(int64(blockNumber)),
		}
		// Retrieve the logs
		logs, err := ethClient.FilterLogs(context.Background(), query)
		if err != nil {
			logger.Errorf("error while filtering logs: %v", err)
			return
		}
		// Iterate on all the logs found to create the mongo log
		storeLogs(logs)
		Configuration.Server.FromBlock = blockNumber
		serverLatestBlockRead.BlockNumber = blockNumber
		res = collection.FindOneAndUpdate(context.Background(), bson.M{"_id": 0}, bson.M{"$set": serverLatestBlockRead})
		if res.Err() != nil {
			serverLatestBlockRead.ID = 0
			_, err = collection.InsertOne(context.Background(), serverLatestBlockRead)
			if err != nil {
				logger.Errorf("error while inserting log latest block read: %v", res.Err())
			}
		}
	} else {
		// Retrieve all the contracts and iterate
		for _, contract := range Configuration.Contracts {
			logger.Infof("retrieving logs for contract %s at block number %d", contract.Address, blockNumber)
			var logLatestBlockRead LogLatestBlockRead
			// Contract address hex to address
			addr := common.HexToAddress(contract.Address)
			// Check latest block read
			res := collection.FindOne(context.Background(), bson.M{"address": contract.Address})
			if res.Err() != nil {
				err := res.Decode(&logLatestBlockRead)
				if err != nil {
					logLatestBlockRead = LogLatestBlockRead{Address: contract.Address, BlockNumber: 0}
				}
			} else {
				logLatestBlockRead = LogLatestBlockRead{Address: contract.Address, BlockNumber: 0}
			}
			// Iterate all the signatures
			for _, signature := range contract.Signatures {
				// Build the filter query
				query := ethereum.FilterQuery{
					FromBlock: big.NewInt(int64(logLatestBlockRead.BlockNumber)),
					ToBlock:   big.NewInt(int64(blockNumber)),
					Addresses: []common.Address{addr},
					Topics:    [][]common.Hash{{crypto.Keccak256Hash([]byte(signature))}},
				}
				// Retrieve the logs
				logs, err := ethClient.FilterLogs(context.Background(), query)
				if err != nil {
					logger.Errorf("error while filtering logs: %v", err)
					return
				}
				// Iterate on all the logs found to create the mongo log
				storeLogs(logs)
				logLatestBlockRead.BlockNumber = blockNumber
				_, err = collection.InsertOne(context.Background(), logLatestBlockRead)
				if err != nil {
					logger.Errorf("error while inserting log latest block read: %v", err)
					continue
				}
			}
			logger.Infof("contract %s logs at block number %d retrieved successfully!", contract.Address, blockNumber)
		}
	}
}

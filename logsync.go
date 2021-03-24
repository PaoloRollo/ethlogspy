package main

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
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
			logger.Errorf("error during blockchain head subscription: %v", err)
		case header := <-headers:
			logger.Infof("new block received, hash: %s", header.Hash().String())
			block, err := ethClient.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				logger.Errorf("error while retrieving block by hash %s: %v", header.Hash(), err)
				continue
			}
			logs, err := ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
				FromBlock: block.Number(),
				ToBlock:   block.Number(),
			})
			if err != nil {
				logger.Errorf("error while retrieving logs by block number %d: %v", block.Number().Int64(), err)
				continue
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
	logger.Infof("retrieving logs using the following query: %+v", query)
	// Retrieve the logs
	logs, err := ethClient.FilterLogs(context.Background(), query)
	if err != nil {
		logger.Errorf("error while filtering logs: %v", err)
		return
	}
	logger.Info("logs retrieved successfully from the node, storing them..")
	// Iterate on all the logs found to create the mongo log
	storeLogs(logs)
	serverLatestBlockRead.BlockNumber = blockNumber
	res = collection.FindOneAndUpdate(context.Background(), bson.M{"_id": 0}, bson.M{"$set": serverLatestBlockRead})
	if res.Err() != nil {
		serverLatestBlockRead.ID = 0
		_, err = collection.InsertOne(context.Background(), serverLatestBlockRead)
		if err != nil {
			logger.Errorf("error while inserting log latest block read: %v", res.Err())
		}
	}
}

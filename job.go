package main

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.mongodb.org/mongo-driver/bson"
)

// RetrieveLogs is the function used to retrieve logs
func retrieveLogs() {
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
		for _, log := range logs {
			topics := []string{}
			for _, topic := range log.Topics {
				topics = append(topics, topic.String())
			}
			mongoLog := Log{
				Removed:          log.Removed,
				LogIndex:         log.Index,
				TransactionIndex: log.TxIndex,
				BlockNumber:      log.BlockNumber,
				Address:          log.Address.Hex(),
				Data:             string(log.Data),
				Topics:           topics,
			}
			collection := mongoDatabase.Collection("logs")
			_, err := collection.InsertOne(context.Background(), mongoLog)
			if err != nil {
				logger.Errorf("error while inserting log: %v", err)
				continue
			}
		}
		Configuration.Server.FromBlock = blockNumber
		serverLatestBlockRead.BlockNumber = blockNumber
		res = collection.FindOneAndUpdate(context.Background(), bson.M{"_id": 0}, serverLatestBlockRead)
		if res.Err() != nil {
			logger.Errorf("error while inserting log latest block read: %v", err)
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
					logger.Errorf("error while decoding log latest block read: %v", err)
					continue
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
				for _, log := range logs {
					topics := []string{}
					for _, topic := range log.Topics {
						topics = append(topics, topic.String())
					}
					mongoLog := Log{
						Removed:          log.Removed,
						LogIndex:         log.Index,
						TransactionIndex: log.TxIndex,
						BlockNumber:      log.BlockNumber,
						Address:          log.Address.Hex(),
						Data:             string(log.Data),
						Topics:           topics,
					}
					collection := mongoDatabase.Collection("logs")
					_, err := collection.InsertOne(context.Background(), mongoLog)
					if err != nil {
						logger.Errorf("error while inserting log: %v", err)
						continue
					}
				}
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

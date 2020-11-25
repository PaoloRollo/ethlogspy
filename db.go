package main

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func storeLogs(logs []types.Log) {
	// Iterate on all the logs found to create the mongo log
	for _, log := range logs {
		logJSON, err := log.MarshalJSON()
		if err != nil {
			logger.Errorf("error while marshaling log: %v", err)
			continue
		}
		logJSONString := string(logJSON)
		logger.Infof("currenty iterating: %s", logJSONString)
		topics := []string{}
		for _, topic := range log.Topics {
			topics = append(topics, topic.String())
		}
		JSONlog := JSONLog{}
		mongoLog := Log{}
		err = json.Unmarshal(logJSON, &JSONlog)
		if err != nil {
			logger.Errorf("error while unmarshaling log: %v", err)
			continue
		}
		mongoLog.Address = JSONlog.Address
		mongoLog.Removed = JSONlog.Removed
		mongoLog.BlockHash = JSONlog.BlockHash
		mongoLog.Data = JSONlog.Data
		mongoLog.Topics = JSONlog.Topics
		logIndex, err := hexutil.DecodeBig(JSONlog.LogIndex)
		if err != nil {
			logger.Errorf("error while decoding log index: %v", err)
			continue
		}
		mongoLog.LogIndex = int(logIndex.Int64())
		transactionIndex, err := hexutil.DecodeBig(JSONlog.TransactionIndex)
		if err != nil {
			logger.Errorf("error while decoding transaction index: %v", err)
			continue
		}
		mongoLog.TransactionIndex = int(transactionIndex.Int64())
		blockNumber, err := hexutil.DecodeUint64(JSONlog.BlockNumber)
		if err != nil {
			logger.Errorf("error while decoding block number: %v", err)
			continue
		}
		mongoLog.BlockNumber = int(blockNumber)
		collection := mongoDatabase.Collection("logs")
		logger.Infof("inserting log in the collection: %v", mongoLog)
		_, err = collection.InsertOne(context.Background(), mongoLog)
		if err != nil {
			logger.Errorf("error while inserting log: %v", err)
			continue
		}
	}
}

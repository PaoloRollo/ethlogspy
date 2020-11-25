package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
)

var logger *logrus.Logger

func setuplogger() {
	logger = &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &logrus.TextFormatter{DisableColors: false, FullTimestamp: true},
		Level:     logrus.InfoLevel,
	}
}

func validatePath(path string) (*string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("'%s' is not a valid directory", path)
	}
	return &path, nil
}

func getFilter(params []LogRequestFilter) (bson.M, error) {
	var bsonFilter bson.M
	if len(params) > 0 {
		filter := params[0]
		logger.Infof("using filter %+v", filter)
		if filter.Address != "" {
			bsonFilter["address"] = filter.Address
		}
		if filter.FromBlock != "" {
			switch filter.FromBlock.(type) {
			case string:
				if filter.FromBlock == "earliest" {
					bsonFilter["fromBlock"] = bson.M{"$ge": 0}
				} else if filter.FromBlock == "latest" {
					blockNumber, err := ethClient.BlockNumber(context.TODO())
					if err != nil {
						return bsonFilter, err
					}
					bsonFilter["fromBlock"] = bson.M{"$ge": blockNumber}
				}
			default:
				bsonFilter["fromBlock"] = bson.M{"$ge": filter.FromBlock}
			}
		}
		if filter.ToBlock != "" {
			switch filter.ToBlock.(type) {
			case string:
				if filter.ToBlock == "earliest" {
					bsonFilter["toBlock"] = bson.M{"$le": 0}
				} else if filter.ToBlock == "latest" {
					blockNumber, err := ethClient.BlockNumber(context.TODO())
					if err != nil {
						logger.Errorf("error while retrieving block number: %v", err)
						return bsonFilter, err
					}
					bsonFilter["toBlock"] = bson.M{"$le": blockNumber}
				}
			default:
				if filter.ToBlock != nil {
					bsonFilter["toBlock"] = bson.M{"$le": filter.ToBlock}
				}
			}
		}
		if filter.Topics != nil && len(filter.Topics) > 0 {
			bsonFilter["topics"] = bson.M{"$in": filter.Topics}
		}
	}
	return bsonFilter, nil
}

func getLogs(req LogRequest, ctx *fasthttp.RequestCtx) error {
	var logs []Log
	// Build the redis key
	redisKeyBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}
	redisKey := string(redisKeyBytes)
	// Retrieve the value from redis, if it exists
	val, err := rdb.Get(redisCtx, redisKey).Result()
	if err == nil || err == redis.Nil {
		// An error has occurred or the key was not set, retrieve it from couchdb
		bsonFilter, err := getFilter(req.Params)
		if err != nil {
			return err
		}
		// Create find query context
		findCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Get logs mongodb collection
		collection := mongoDatabase.Collection("logs")
		res, err := collection.Find(findCtx, bsonFilter)
		if err != nil {
			logger.Errorf("error while finding logs on database: %v", err)
			return err
		}
		defer res.Close(findCtx)
		for res.Next(findCtx) {
			var log Log
			if err = res.Decode(&log); err != nil {
				logger.Errorf("error while retrieving logs: %v", err)
				return err
			}
			logs = append(logs, log)
		}
		// Marshal the logs for redis
		marshaledLogs, err := json.Marshal(logs)
		if err != nil {
			logger.Errorf("error while marshaling logs: %v", err)
			return err
		}
		// Set the value in cache
		rdb.SetEX(redisCtx, redisKey, string(marshaledLogs), 30*time.Second)
		// Encode the value read from mongo
		if err := json.NewEncoder(ctx).Encode(LogResponse{ID: req.ID, JSONRPC: req.JSONRPC, Result: logs}); err != nil {
			return err
		}
		return err
	}
	json.Unmarshal([]byte(val), &logs)
	// Encode the value read from the cached val
	if err := json.NewEncoder(ctx).Encode(LogResponse{ID: req.ID, JSONRPC: req.JSONRPC, Result: logs}); err != nil {
		return err
	}
	return nil
}

func getWsLogs(req LogRequest) (LogResponse, error) {
	var logs []Log
	// Build the redis key
	redisKeyBytes, err := json.Marshal(req)
	if err != nil {
		return LogResponse{}, err
	}
	redisKey := string(redisKeyBytes)
	// Retrieve the value from redis, if it exists
	val, err := rdb.Get(redisCtx, redisKey).Result()
	if err == nil || err == redis.Nil {
		// An error has occurred or the key was not set, retrieve it from couchdb
		bsonFilter, err := getFilter(req.Params)
		if err != nil {
			return LogResponse{}, err
		}
		// Create find query context
		findCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Get logs mongodb collection
		collection := mongoDatabase.Collection("logs")
		res, err := collection.Find(findCtx, bsonFilter)
		if err != nil {
			logger.Errorf("error while finding logs on database: %v", err)
			return LogResponse{}, err
		}
		defer res.Close(findCtx)
		for res.Next(findCtx) {
			var log Log
			if err = res.Decode(&log); err != nil {
				logger.Errorf("error while retrieving logs: %v", err)
				return LogResponse{}, err
			}
			logs = append(logs, log)
		}
		// Marshal the logs for redis
		marshaledLogs, err := json.Marshal(logs)
		if err != nil {
			logger.Errorf("error while marshaling logs: %v", err)
			return LogResponse{}, err
		}
		// Set the value in cache
		rdb.SetEX(redisCtx, redisKey, string(marshaledLogs), 30*time.Second)
		// Return the log response
		return LogResponse{ID: req.ID, JSONRPC: req.JSONRPC, Result: logs}, err
	}
	json.Unmarshal([]byte(val), &logs)
	return LogResponse{ID: req.ID, JSONRPC: req.JSONRPC, Result: logs}, nil
}

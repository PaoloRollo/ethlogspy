package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
	"github.com/valyala/fasthttp"
	proxy "github.com/yeqown/fasthttp-reverse-proxy"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ethClient          *ethclient.Client
	proxyServer        *proxy.ReverseProxy
	rdb                *redis.Client
	mongoClient        *mongo.Client
	mongoDatabase      *mongo.Database
	redisCtx           = context.Background()
	strContentType     = []byte("Content-Type")
	strApplicationJSON = []byte("application/json")
)

// LogRequest struct
type LogRequest struct {
	JSONRPC string             `json:"jsonrpc"`
	Method  string             `json:"method"`
	Params  []LogRequestFilter `json:"params"`
	ID      int                `json:"id"`
}

// LogLatestBlockRead struct
type LogLatestBlockRead struct {
	Address     string `json:"address"`
	BlockNumber int    `json:"blockNumber"`
}

// LogRequestFilter struct
type LogRequestFilter struct {
	FromBlock interface{} `json:"fromBlock"`
	ToBlock   interface{} `json:"toBlock"`
	Address   string      `json:"address"`
	Topics    []string    `json:"topics"`
}

// Log Struct
type Log struct {
	Removed          bool     `json:"removed"`
	LogIndex         uint     `json:"log_index"`
	TransactionIndex uint     `json:"transaction_index"`
	BlockNumber      uint64   `json:"block_number"`
	BlockHash        string   `json:"block_hash"`
	Address          string   `json:"address"`
	Data             string   `json:"data"`
	Topics           []string `json:"topics"`
}

// LogResponse struct
type LogResponse struct {
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
	Result  []Log  `json:"result"`
}

func retrieveLogs() {
	// Retrieve all the contracts and iterate
	for _, contract := range Configuration.Contracts {
		// Contract address hex to address
		addr := common.HexToAddress(contract.Address)
		// Iterate all the signatures
		for _, signature := range contract.Signatures {
			// Build the filter query
			query := ethereum.FilterQuery{
				Addresses: []common.Address{addr},
				Topics:    [][]common.Hash{{crypto.Keccak256Hash([]byte(signature))}},
			}
			// Retrieve the logs
			logs, err := ethClient.FilterLogs(context.Background(), query)
			if err != nil {
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
					continue
				}
			}
		}
	}
}

// ProxyHandler is the main core
func ProxyHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) == "POST" {
		var req LogRequest
		json.Unmarshal(ctx.PostBody(), &req)
		if strings.ToLower(req.Method) == "eth_getlogs" && req.ID == 1 {
			var logs []Log
			// Set the content type response header
			ctx.Response.Header.SetCanonical(strContentType, strApplicationJSON)
			// Build the redis key
			redisKeyBytes, err := json.Marshal(req)
			if err != nil {
				// Error while marshaling request
				proxyServer.ServeHTTP(ctx)
				return
			}
			redisKey := string(redisKeyBytes)
			// Retrieve the value from redis, if it exists
			val, err := rdb.Get(redisCtx, redisKey).Result()
			if err == nil || err == redis.Nil {
				// An error has occurred or the key was not set, retrieve it from couchdb
				bsonFilter := bson.M{}
				if len(req.Params) > 0 {
					filter := req.Params[0]
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
									// Error while retrieving blockNumber, forward call to node
									proxyServer.ServeHTTP(ctx)
									return
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
									// Error while retrieving blockNumber, forward call to node
									proxyServer.ServeHTTP(ctx)
									return
								}
								bsonFilter["toBlock"] = bson.M{"$le": blockNumber}
							}
						default:
							bsonFilter["toBlock"] = bson.M{"$le": filter.ToBlock}
						}
					}
					if filter.Topics != nil && len(filter.Topics) > 0 {
						bsonFilter["topics"] = bson.M{"$in": filter.Topics}
					}
				}
				// Create find query context
				findCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				// Get logs mongodb collection
				collection := mongoDatabase.Collection("logs")
				res, err := collection.Find(findCtx, bson.M{})
				if err != nil {
					// Error while finding on database, forward call to node
					proxyServer.ServeHTTP(ctx)
					return
				}
				err = res.Decode(&logs)
				if err != nil {
					// Error while decoding request, forward call to node
					proxyServer.ServeHTTP(ctx)
					return
				}
				// Marshal the logs for redis
				marshaledLogs, err := json.Marshal(logs)
				if err != nil {
					// Error while marshaling logs, forward call to node
					proxyServer.ServeHTTP(ctx)
					return
				}
				// Set the value in cache
				rdb.SetEX(redisCtx, redisKey, string(marshaledLogs), 30*time.Second)
				// Set the 200 status code
				ctx.Response.SetStatusCode(200)
				// Encode the value read from mongo
				if err := json.NewEncoder(ctx).Encode(LogResponse{ID: req.ID, JSONRPC: req.JSONRPC, Result: logs}); err != nil {
					ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
				}
				return
			}
			json.Unmarshal([]byte(val), &logs)
			// Set the 200 status code
			ctx.Response.SetStatusCode(200)
			// Encode the value read from the cached val
			if err := json.NewEncoder(ctx).Encode(LogResponse{ID: req.ID, JSONRPC: req.JSONRPC, Result: logs}); err != nil {
				ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
			}
			return
		}
	}
	// Proxy to the node
	proxyServer.ServeHTTP(ctx)
}

func main() {
	var err error
	// Setup the logger
	SetupLogger()
	// Parse the config path or use the default value
	configPath := flag.String("config", "/usr/local/ethlogspy/configs/", "path to config file")
	flag.Parse()
	// Validates the path
	validatedPath, err := ValidatePath(*configPath)
	// If the path is invalid log the error and exit
	if err != nil {
		Logger.Fatalf("failed to validate path: %v", err)
	}
	Logger.Info("retrieving configuration..")
	// Retrieve the config from the config path
	GetConfig(*validatedPath)
	ethClient, err = ethclient.Dial(fmt.Sprintf("http://%s:%d", Configuration.Node.Host, Configuration.Node.Port))
	if err != nil {
		panic(err)
	}
	// Create proxy server
	proxyServer = proxy.NewReverseProxy(fmt.Sprintf("%s:%d", Configuration.Node.Host, Configuration.Node.Port))
	// Connect to redis
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", Configuration.Redis.Host, Configuration.Redis.Port),
		Password: Configuration.Redis.Password,
		DB:       Configuration.Redis.DB,
	})
	// Create mongo client context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Create mongo client
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(Configuration.Mongo.Connection))
	if err != nil {
		panic(err)
	}
	// Call defer mongo client connection
	defer func() {
		if err = mongoClient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	// Retrieve the database
	mongoDatabase = mongoClient.Database(Configuration.Mongo.DbName)
	// Create cron
	c := cron.New()
	// Add the retrieve logs function every 30 seconds
	c.AddFunc("0/30 * * * * ?", retrieveLogs)
	// Start the cron
	c.Start()
	if err := fasthttp.ListenAndServe(fmt.Sprintf(":%d", Configuration.Server.Port), ProxyHandler); err != nil {
		// Stop the cron
		c.Stop()
		Logger.Fatal(err)
	}
}

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fasthttp/websocket"
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
	wsProxyServer      *proxy.WSReverseProxy
	rdb                *redis.Client
	mongoClient        *mongo.Client
	mongoDatabase      *mongo.Database
	redisCtx           = context.Background()
	strContentType     = []byte("Content-Type")
	strApplicationJSON = []byte("application/json")
	upgrader           = websocket.FastHTTPUpgrader{
		WriteBufferSize: 1024,
		ReadBufferSize:  1024,
	}
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
	Address     string `json:"address" bson:"address,omitempty"`
	BlockNumber uint64 `json:"blockNumber" bson:"blockNumber,omitempty"`
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
	Removed          bool     `json:"removed" bson:"removed,omitempty"`
	LogIndex         uint     `json:"log_index" bson:"log_index,omitempty"`
	TransactionIndex uint     `json:"transaction_index" bson:"transaction_index,omitempty"`
	BlockNumber      uint64   `json:"block_number" bson:"block_number,omitempty"`
	BlockHash        string   `json:"block_hash" bson:"block_hash,omitempty"`
	Address          string   `json:"address" bson:"address,omitempty"`
	Data             string   `json:"data" bson:"data,omitempty"`
	Topics           []string `json:"topics" bson:"topics,omitempty"`
}

// LogResponse struct
type LogResponse struct {
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
	Result  []Log  `json:"result"`
}

func retrieveLogs() {
	collection := mongoDatabase.Collection("logLatestBlockRead")
	blockNumber, err := ethClient.BlockNumber(context.TODO())
	if err != nil {
		Logger.Errorf("error while retrieving block number: %v", err)
		return
	}
	// Retrieve all the contracts and iterate
	for _, contract := range Configuration.Contracts {
		Logger.Infof("retrieving logs for contract %s at block number %d", contract.Address, blockNumber)
		var logLatestBlockRead LogLatestBlockRead
		// Contract address hex to address
		addr := common.HexToAddress(contract.Address)
		// Check latest block read
		res := collection.FindOne(context.Background(), bson.M{"address": contract.Address})
		if res.Err() != nil {
			err := res.Decode(&logLatestBlockRead)
			if err != nil {
				Logger.Errorf("error while decoding log latest block read: %v", err)
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
				Logger.Errorf("error while filtering logs: %v", err)
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
					Logger.Errorf("error while inserting log: %v", err)
					continue
				}
			}
			logLatestBlockRead.BlockNumber = blockNumber
			_, err = collection.InsertOne(context.Background(), logLatestBlockRead)
			if err != nil {
				Logger.Errorf("error while inserting log latest block read: %v", err)
				continue
			}
		}
		Logger.Infof("contract %s logs at block number %d retrieved successfully!", contract.Address, blockNumber)
	}
}

// ProxyHandler is the main core
func ProxyHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) == "POST" {
		var req LogRequest
		json.Unmarshal(ctx.PostBody(), &req)
		if strings.ToLower(req.Method) == "eth_getlogs" && req.ID == 1 {
			err := GetLogs(req, ctx)
			if err != nil {
				Logger.Errorf("error while retrieving logs: %v", err)
			}
		}
	}
	// Serve the context
	proxyServer.ServeHTTP(ctx)
}

// WebsocketProxyHandler is the websocket main core
func WebsocketProxyHandler(ctx *fasthttp.RequestCtx) {
	err := upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
		// Defer connection close
		defer ws.Close()
		for {
			var req LogRequest
			_, message, err := ws.ReadMessage()
			if err != nil {
				Logger.Errorf("error while reading message: %v", err)
				break
			}
			err = json.Unmarshal(message, &req)
			if err != nil {
				Logger.Errorf("error while marshaling ws body: %v", err)
				break
			}
			if strings.ToLower(req.Method) == "eth_getlogs" {
				logResponse, err := GetWsLogs(req)
				if err != nil {
					Logger.Errorf("error retrieving logs: %v", err)
					break
				}
				ws.WriteJSON(logResponse)
			} else {
				// Serve all the requests with the proxy
				wsProxyServer.ServeHTTP(ctx)
			}
		}
	})
	if err != nil {
		Logger.Errorf("error during request context upgrade: %v", err)
		return
	}
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
	Logger.Info("configuration retrieved successfully..")
	// Initializing ETH client
	Logger.Info("initializing eth client..")
	ethClient, err = ethclient.Dial(fmt.Sprintf("http://%s:%d", Configuration.Node.Host, Configuration.Node.Port))
	if err != nil {
		panic(err)
	}
	Logger.Info("eth client initialized successfully..")
	// Create proxy server
	Logger.Info("creating http and ws proxy servers..")
	wsProxyServer = proxy.NewWSReverseProxy(fmt.Sprintf("%s:%d", Configuration.Node.Host, Configuration.Node.Port), "/ws")
	proxyServer = proxy.NewReverseProxy(fmt.Sprintf("%s:%d", Configuration.Node.Host, Configuration.Node.Port))
	Logger.Info("proxy servers created successfully..")
	// Connect to redis
	Logger.Info("connecting to redis..")
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", Configuration.Redis.Host, Configuration.Redis.Port),
		Password: Configuration.Redis.Password,
		DB:       Configuration.Redis.DB,
	})
	Logger.Info("connected successfully to redis..")
	// Create mongo client context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Create mongo client
	Logger.Info("connecting to mongodb..")
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
	Logger.Info("connected successfully to mongodb..")
	// Retrieve the database
	mongoDatabase = mongoClient.Database(Configuration.Mongo.DbName)
	// Create cron
	Logger.Info("creating and starting cron jobs every 30 seconds..")
	c := cron.New()
	// Add the retrieve logs function every 30 seconds
	c.AddFunc("0/30 * * * * ?", retrieveLogs)
	// Start the cron
	c.Start()
	Logger.Info("starting ethlogspy server..")
	if err := fasthttp.ListenAndServe(fmt.Sprintf(":%d", Configuration.Server.Port), ProxyHandler); err != nil {
		// Stop the cron
		c.Stop()
		Logger.Fatal(err)
	}
}

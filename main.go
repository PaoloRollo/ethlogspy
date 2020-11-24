package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fasthttp/websocket"
	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
	"github.com/valyala/fasthttp"
	proxy "github.com/yeqown/fasthttp-reverse-proxy"
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

// ServerLatestBlockRead struct
type ServerLatestBlockRead struct {
	ID          int    `json:"id" bson:"_id"`
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

// ProxyHandler is the main core
func ProxyHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) == "POST" {
		var req LogRequest
		json.Unmarshal(ctx.PostBody(), &req)
		if strings.ToLower(req.Method) == "eth_getlogs" && req.ID == 1 {
			err := getLogs(req, ctx)
			if err != nil {
				logger.Errorf("error while retrieving logs: %v", err)
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
				logger.Errorf("error while reading message: %v", err)
				break
			}
			err = json.Unmarshal(message, &req)
			if err != nil {
				logger.Errorf("error while marshaling ws body: %v", err)
				break
			}
			if strings.ToLower(req.Method) == "eth_getlogs" {
				logResponse, err := getWsLogs(req)
				if err != nil {
					logger.Errorf("error retrieving logs: %v", err)
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
		logger.Errorf("error during request context upgrade: %v", err)
		return
	}
}

func main() {
	var err error
	// Setup the logger
	setuplogger()
	// Parse the config path or use the default value
	configPath := flag.String("config", "/usr/local/ethlogspy/configs/", "path to config file")
	flag.Parse()
	// Validates the path
	validatedPath, err := validatePath(*configPath)
	// If the path is invalid log the error and exit
	if err != nil {
		logger.Fatalf("failed to validate path: %v", err)
	}
	logger.Info("retrieving configuration..")
	// Retrieve the config from the config path
	GetConfig(*validatedPath)
	logger.Info("configuration retrieved successfully..")
	// Initializing ETH client
	logger.Info("initializing eth client..")
	ethClient, err = ethclient.Dial(fmt.Sprintf("http://%s:%d", Configuration.Node.Host, Configuration.Node.Port))
	if err != nil {
		panic(err)
	}
	logger.Info("eth client initialized successfully..")
	// Create proxy server
	logger.Info("creating http and ws proxy servers..")
	wsProxyServer = proxy.NewWSReverseProxy(fmt.Sprintf("%s:%d", Configuration.Node.Host, Configuration.Node.Port), "/ws")
	proxyServer = proxy.NewReverseProxy(fmt.Sprintf("%s:%d", Configuration.Node.Host, Configuration.Node.Port))
	logger.Info("proxy servers created successfully..")
	// Connect to redis
	logger.Info("connecting to redis..")
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", Configuration.Redis.Host, Configuration.Redis.Port),
		Password: Configuration.Redis.Password,
		DB:       Configuration.Redis.DB,
	})
	logger.Info("connected successfully to redis..")
	// Create mongo client context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Create mongo client
	logger.Info("connecting to mongodb..")
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
	logger.Info("connected successfully to mongodb..")
	// Retrieve the database
	mongoDatabase = mongoClient.Database(Configuration.Mongo.DbName)
	// Create cron
	logger.Info("creating and starting cron jobs every 30 seconds..")
	c := cron.New()
	// Add the retrieve logs function every 30 seconds
	c.AddFunc("0/30 * * * * ?", retrieveLogs)
	// Start the cron
	c.Start()
	logger.Info("starting ethlogspy server..")
	if err := fasthttp.ListenAndServe(fmt.Sprintf(":%d", Configuration.Server.Port), ProxyHandler); err != nil {
		// Stop the cron
		c.Stop()
		logger.Fatal(err)
	}
}

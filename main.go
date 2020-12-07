package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	cors "github.com/AdhityaRamadhanus/fasthttpcors"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fasthttp/websocket"
	"github.com/go-redis/redis/v8"
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

// ProxyHandler is the main core
func ProxyHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) == "POST" {
		var req LogRequest
		json.Unmarshal(ctx.PostBody(), &req)
		logger.Infof("new request incoming for method: %s", req.Method)
		if strings.ToLower(req.Method) == "eth_getlogs" {
			logger.Infof("spying on eth_getLogs.. shh...")
			err := getLogs(req, ctx)
			if err != nil {
				logger.Errorf("error while retrieving logs: %v", err)
			} else {
				return
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
	// Parse the intel mode flag
	intelMode := flag.Bool("intel", false, "activate intel mode")
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
	ethClient, err = ethclient.Dial(fmt.Sprintf("ws://%s:%d", Configuration.Node.Host, Configuration.Node.Port))
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
	// Check if logs have been already saved
	res := mongoDatabase.Collection("logs").FindOne(context.Background(), bson.M{})
	if res.Err() != nil {
		start := time.Now()
		logger.Info("syncing mongodb with ethereum logs..")
		syncLogs()
		elapsed := time.Since(start)
		logger.Info("logs sync successful, elapsed: ", elapsed)
	}
	// Start subscribing to logs
	go subscribeToHead()
	// Create handler
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		if strings.Contains(string(ctx.Path()), "/ws") {
			WebsocketProxyHandler(ctx)
		} else {
			ProxyHandler(ctx)
		}
	}
	// Add cors middleware
	withCors := cors.NewCorsHandler(cors.Options{
		AllowedOrigins: []string{Configuration.Server.CorsOrigin},
	})
	logger.Info("starting ethlogspy server..")
	if err := fasthttp.ListenAndServe(":8080", withCors.CorsMiddleware(requestHandler)); err != nil {
		logger.Fatal(err)
	}
}

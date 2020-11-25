package main

// Config struct is used to define the application configuration object
type Config struct {
	Mongo struct {
		Connection string `yaml:"connection"`
		DbName     string `yaml:"db_name"`
	}
	Node struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	}
	Redis struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	}
	Server struct {
		FromBlock uint64 `yaml:"from_block"`
	}
}

// JSONLog Struct
type JSONLog struct {
	Removed          bool     `json:"removed"`
	LogIndex         string   `json:"logIndex"`
	TransactionIndex string   `json:"transactionIndex"`
	BlockNumber      string   `json:"blockNumber"`
	BlockHash        string   `json:"blockHash"`
	Address          string   `json:"address"`
	Data             string   `json:"data"`
	Topics           []string `json:"topics"`
}

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
	LogIndex         int      `json:"logIndex" bson:"logIndex,omitempty"`
	TransactionIndex int      `json:"transactionIndex" bson:"transactionIndex,omitempty"`
	BlockNumber      int      `json:"blockNumber" bson:"blockNumber,omitempty"`
	BlockHash        string   `json:"blockHash" bson:"blockHash,omitempty"`
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

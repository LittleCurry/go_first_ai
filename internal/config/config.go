package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server ServerConfig
	LLM    LLMConfig
	Vector VectorConfig
	RAG    RAGConfig
	Agent  AgentConfig
	Redis  RedisConfig
	MySQL  MySQLConfig
}

type MySQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type ServerConfig struct {
	Port  string
	Debug bool
}

type LLMConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int
	Temperature float64
}

type VectorConfig struct {
	Type     string // "chroma" or "qdrant"
	ChromaDB ChromaConfig
}

type ChromaConfig struct {
	PersistDir string
}

type RAGConfig struct {
	TopK                int
	SimilarityThreshold float64
}

type AgentConfig struct {
	MaxIterations int
	Timeout       int
}

var AppConfig *Config

func LoadConfig() error {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system env")
	}

	AppConfig = &Config{
		Server: ServerConfig{
			Port:  getEnv("API_PORT", "8000"),
			Debug: getEnvAsBool("DEBUG", true),
		},
		LLM: LLMConfig{
			APIKey:      getEnv("DEEPSEEK_API_KEY", ""),
			BaseURL:     getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
			Model:       getEnv("LLM_MODEL", "deepseek-chat"),
			MaxTokens:   getEnvAsInt("LLM_MAX_TOKENS", 2048),
			Temperature: getEnvAsFloat("LLM_TEMPERATURE", 0.7),
		},
		Vector: VectorConfig{
			Type: getEnv("VECTOR_DB_TYPE", "chroma"),
			ChromaDB: ChromaConfig{
				PersistDir: getEnv("CHROMA_PERSIST_DIR", "./data/chroma"),
			},
		},
		RAG: RAGConfig{
			TopK:                getEnvAsInt("RAG_TOP_K", 5),
			SimilarityThreshold: getEnvAsFloat("RAG_SIMILARITY_THRESHOLD", 0.7),
		},
		Agent: AgentConfig{
			MaxIterations: getEnvAsInt("AGENT_MAX_ITERATIONS", 3),
			Timeout:       getEnvAsInt("AGENT_TIMEOUT", 30),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		MySQL: MySQLConfig{
			Host:     getEnv("MYSQL_HOST", "127.0.0.1"),
			Port:     getEnv("MYSQL_PORT", "3306"),
			User:     getEnv("MYSQL_USER", "root"),
			Password: getEnv("MYSQL_PASSWORD", ""),
			Database: getEnv("MYSQL_DATABASE", "ai_customer_service"),
		},
	}

	if AppConfig.LLM.APIKey == "" {
		log.Fatal("DEEPSEEK_API_KEY is required in .env file")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

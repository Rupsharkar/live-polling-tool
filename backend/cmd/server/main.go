package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"live-polling-tool/backend/handlers"
	"live-polling-tool/backend/middleware"
	"live-polling-tool/backend/repository"
	"live-polling-tool/backend/routes"
	"live-polling-tool/backend/services"
)

func main() {
	godotenv.Load()

	port := getenv("PORT", "8080")
	frontendURL := getenv("FRONTEND_URL", "http://localhost:5174")
	mongoURI := mustGetenv("MONGO_URI")
	mongoDB := getenv("MONGO_DB", "livepoll")
	redisURL := getenv("REDIS_URL", "redis://localhost:6379")
	jwtSecret := mustGetenv("JWT_SECRET")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MONGO CONNECT ERROR: %v", err)
	}

	if err = mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("MONGO PING ERROR: %v", err)
	}

	log.Println("MongoDB connected successfully")

	db := mongoClient.Database(mongoDB)

	repo, err := repository.New(db)
	if err != nil {
		log.Fatalf("REPOSITORY ERROR: %v", err)
	}

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("REDIS URL ERROR: %v", err)
	}

	log.Printf("Redis address: %s", redisOpts.Addr)

	rdb := redis.NewClient(redisOpts)

	log.Println("Redis client initialized")

	redisService := services.NewRedisService(rdb)
	auth := middleware.NewAuth(jwtSecret)
	handler := handlers.New(repo, redisService, auth, jwtSecret)

router := gin.Default()

router.Use(func(c *gin.Context) {
    c.Header("Access-Control-Allow-Origin", "https://live-polling-tool-alpha.vercel.app")
    c.Header("Access-Control-Allow-Credentials", "true")
    c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
    c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

    if c.Request.Method == "OPTIONS" {
        c.AbortWithStatus(204)
        return
    }

    c.Next()
})

routes.Register(router, handler, auth, redisService)

	log.Printf("server listening on :%s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func mustGetenv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s is required", key)
	}
	return value
}
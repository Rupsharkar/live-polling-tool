package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	port := getenv("PORT", "8080")
	frontendURL := getenv("FRONTEND_URL", "http://localhost:5173")
	mongoURI := mustGetenv("MONGO_URI")
	mongoDB := getenv("MONGO_DB", "livepoll")
	redisURL := getenv("REDIS_URL", "redis://localhost:6379")
	jwtSecret := mustGetenv("JWT_SECRET")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	if err = mongoClient.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}

	db := mongoClient.Database(mongoDB)

	repo, err := repository.New(db)
	if err != nil {
		log.Fatal(err)
	}

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(redisOpts)

	if err = rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}

	redisService := services.NewRedisService(rdb)
	auth := middleware.NewAuth(jwtSecret)
	handler := handlers.New(repo, redisService, auth, jwtSecret)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{frontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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

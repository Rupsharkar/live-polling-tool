package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"live-polling-tool/backend/middleware"
	"live-polling-tool/backend/repository"
	"live-polling-tool/backend/services"
)

type Handler struct {
	repo      *repository.Repository
	redis     *services.RedisService
	auth      *middleware.Auth
	jwtSecret string
}

func New(repo *repository.Repository, redis *services.RedisService, auth *middleware.Auth, secret string) *Handler {
	return &Handler{repo: repo, redis: redis, auth: auth, jwtSecret: secret}
}

func (h *Handler) Signup(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !validEmail(in.Email) || len(in.Password) < 8 || len(in.Password) > 72 {
		c.JSON(400, gin.H{"error": "use a valid email and an 8-72 character password"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "could not create account"})
		return
	}
	id, err := h.repo.CreateUser(c, in.Email, string(hash))
	if err == repository.ErrDuplicate {
		c.JSON(409, gin.H{"error": "email already registered"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "could not create account"})
		return
	}
	token := h.makeToken(id)
	c.JSON(201, gin.H{"token": token, "email": in.Email})
}

func (h *Handler) Login(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	user, err := h.repo.FindUser(c, email)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid email or password"})
		return
	}
	hash, _ := user["passwordHash"].(string)
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		c.JSON(401, gin.H{"error": "invalid email or password"})
		return
	}
	id, _ := user["_id"].(primitive.ObjectID)
	c.JSON(200, gin.H{"token": h.makeToken(id), "email": email})
}

func (h *Handler) CreatePoll(c *gin.Context) {
	var in struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}
	in.Question = strings.TrimSpace(in.Question)
	if len([]rune(in.Question)) < 5 || len([]rune(in.Question)) > 200 || len(in.Options) < 2 || len(in.Options) > 5 {
		c.JSON(400, gin.H{"error": "question must be 5-200 characters and poll must have 2-5 options"})
		return
	}
	seen := map[string]bool{}
	for i, opt := range in.Options {
		in.Options[i] = strings.TrimSpace(opt)
		key := strings.ToLower(in.Options[i])
		if len([]rune(in.Options[i])) < 1 || len([]rune(in.Options[i])) > 80 || seen[key] {
			c.JSON(400, gin.H{"error": "options must be unique and 1-80 characters"})
			return
		}
		seen[key] = true
	}
	owner := c.MustGet("userID").(primitive.ObjectID)
	id, err := h.repo.CreatePoll(c, owner, in.Question, in.Options)
	if err != nil {
		c.JSON(500, gin.H{"error": "could not create poll"})
		return
	}
	counts := make([]int64, len(in.Options))
	_ = h.redis.SetCounts(c, id.Hex(), counts)
	h.redis.Expire(c, id.Hex())
	c.JSON(201, gin.H{"id": id.Hex(), "question": in.Question, "options": in.Options, "open": true})
}

func (h *Handler) GetPoll(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid poll id"})
		return
	}
	p, err := h.repo.GetPoll(c, id)
	if err == repository.ErrNotFound {
		c.JSON(404, gin.H{"error": "poll not found"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "could not load poll"})
		return
	}

	options, _ := p["options"].(primitive.A)
	counts, err := h.repo.Counts(c, id, len(options))
	if err != nil {
		c.JSON(500, gin.H{"error": "could not load results"})
		return
	}
	_ = h.redis.SetCounts(c, id.Hex(), counts)
	c.JSON(200, gin.H{"id": id.Hex(), "question": p["question"], "options": options, "open": p["open"], "counts": counts})
}

func (h *Handler) Vote(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid poll id"})
		return
	}
	var in struct {
		Option  int    `json:"option"`
		VoterID string `json:"voterId"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}
	if _, err := uuid.Parse(in.VoterID); err != nil {
		c.JSON(400, gin.H{"error": "invalid voter id"})
		return
	}

	p, err := h.repo.GetPoll(c, id)
	if err == repository.ErrNotFound {
		c.JSON(404, gin.H{"error": "poll not found"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "could not load poll"})
		return
	}
	options, _ := p["options"].(primitive.A)
	open, _ := p["open"].(bool)
	if !open {
		c.JSON(409, gin.H{"error": "this poll is closed"})
		return
	}
	if in.Option < 0 || in.Option >= len(options) {
		c.JSON(400, gin.H{"error": "invalid option"})
		return
	}

	if err := h.repo.CreateVote(c, id, in.Option, in.VoterID); err == repository.ErrDuplicate {
		c.JSON(409, gin.H{"error": "you have already voted"})
		return
	} else if err != nil {
		c.JSON(500, gin.H{"error": "could not record vote"})
		return
	}

	counts, err := h.redis.Increment(c, id.Hex(), in.Option)
	if err != nil {
		counts, _ = h.repo.Counts(c, id, len(options))
		_ = h.redis.SetCounts(c, id.Hex(), counts)
	}
	update := services.Update{Option: in.Option, Counts: counts}
	_ = h.redis.Publish(c, id.Hex(), update)
	c.JSON(200, gin.H{"counts": counts})
}

func (h *Handler) ClosePoll(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid poll id"})
		return
	}
	owner := c.MustGet("userID").(primitive.ObjectID)
	if err := h.repo.ClosePoll(c, id, owner); err == repository.ErrNotFound {
		c.JSON(404, gin.H{"error": "poll not found or not owned by you"})
		return
	} else if err != nil {
		c.JSON(500, gin.H{"error": "could not close poll"})
		return
	}
	c.JSON(200, gin.H{"message": "poll closed"})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) WebSocket(c *gin.Context) {
	pollID := c.Param("id")
	if _, err := primitive.ObjectIDFromHex(pollID); err != nil {
		c.JSON(400, gin.H{"error": "invalid poll id"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	pubsub := h.redis.Subscribe(pollID)
	defer pubsub.Close()
	_, _ = pubsub.Receive(context.Background())

	poll, err := h.repo.GetPoll(c, mustObjectID(pollID))
	if err == nil {
		options, _ := poll["options"].(primitive.A)
		counts, _ := h.repo.Counts(c, mustObjectID(pollID), len(options))
		_ = conn.WriteJSON(gin.H{"type": "results", "counts": counts})
	}

	ch := pubsub.Channel()
	for msg := range ch {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
			return
		}
	}
}

func (h *Handler) makeToken(id primitive.ObjectID) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": id.Hex(), "exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(h.jwtSecret))
	return signed
}

func validEmail(s string) bool {
	return regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`).MatchString(s)
}
func mustObjectID(s string) primitive.ObjectID { id, _ := primitive.ObjectIDFromHex(s); return id }

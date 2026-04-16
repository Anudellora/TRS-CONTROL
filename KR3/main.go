package main

import (
	"crypto/subtle"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	secretKey    = []byte("super_secret_key_12345")
	fakeUsersDB  = make(map[string]*UserInDB)
	fakeUsersMu  sync.RWMutex
	rateLimiter  = newRateLimiter()
)

// ==================== Rate Limiter (Задание 6.5) ====================

type rateLimiterStore struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

func newRateLimiter() *rateLimiterStore {
	return &rateLimiterStore{requests: make(map[string][]time.Time)}
}

func (rl *rateLimiterStore) allow(key string, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	var valid []time.Time
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= limit {
		rl.requests[key] = valid
		return false
	}

	rl.requests[key] = append(valid, now)
	return true
}

// ==================== JWT (Задание 6.4) ====================

func generateJWT(username string) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(30 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func parseJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// ==================== Middleware ====================

func jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Missing or invalid token"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := parseJWT(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid or expired token"})
			c.Abort()
			return
		}

		username, _ := claims["sub"].(string)
		c.Set("username", username)
		c.Next()
	}
}

func roleRequired(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetString("username")
		fakeUsersMu.RLock()
		user, exists := fakeUsersDB[username]
		fakeUsersMu.RUnlock()

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "User not found"})
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if user.Role == role {
				c.Set("role", user.Role)
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"detail": "Access denied"})
		c.Abort()
	}
}

// ==================== Задание 6.3: Документация ====================

func getMode() string {
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "DEV"
	}
	return mode
}

func docsAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		docsUser := os.Getenv("DOCS_USER")
		docsPass := os.Getenv("DOCS_PASSWORD")
		if docsUser == "" || docsPass == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": "Docs credentials not configured"})
			c.Abort()
			return
		}

		user, pass, ok := c.Request.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(user), []byte(docsUser)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte(docsPass)) != 1 {
			c.Header("WWW-Authenticate", `Basic realm="Docs"`)
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ==================== Main ====================

func main() {
	initDB()
	defer db.Close()

	mode := getMode()
	router := gin.Default()

	// --- Задание 6.3: Управление документацией ---
	if mode == "DEV" {
		docsGroup := router.Group("/", docsAuthMiddleware())
		docsGroup.GET("/docs", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "API Documentation (DEV mode)"})
		})
		docsGroup.GET("/openapi.json", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"openapi": "3.0.0", "info": gin.H{"title": "KR3 API", "version": "1.0.0"}})
		})
	} else if mode == "PROD" {
		for _, path := range []string{"/docs", "/openapi.json", "/redoc"} {
			p := path
			router.GET(p, func(c *gin.Context) {
				c.JSON(http.StatusNotFound, gin.H{"detail": "Not Found"})
			})
		}
	}

	// ==================== Задание 6.1: Базовая аутентификация ====================
	router.GET("/basic-login", func(c *gin.Context) {
		user, pass, ok := c.Request.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(user), []byte("admin")) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte("secret")) != 1 {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Unauthorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "You got my secret, welcome"})
	})

	// ==================== Задание 6.2 + 6.5: Регистрация с хешированием ====================
	router.POST("/register", func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !rateLimiter.allow("register:"+clientIP, 1, time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"detail": "Too many requests"})
			return
		}

		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}

		fakeUsersMu.Lock()
		defer fakeUsersMu.Unlock()

		for existingName := range fakeUsersDB {
			if subtle.ConstantTimeCompare([]byte(existingName), []byte(req.Username)) == 1 {
				c.JSON(http.StatusConflict, gin.H{"detail": "User already exists"})
				return
			}
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to hash password"})
			return
		}

		role := req.Role
		if role == "" {
			role = "user"
		}
		if role != "admin" && role != "user" && role != "guest" {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid role. Allowed: admin, user, guest"})
			return
		}

		fakeUsersDB[req.Username] = &UserInDB{
			Username:       req.Username,
			HashedPassword: string(hashed),
			Role:           role,
		}

		c.JSON(http.StatusCreated, gin.H{"message": "New user created"})
	})

	// ==================== Задание 6.4 + 6.5: JWT логин ====================
	router.POST("/login", func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !rateLimiter.allow("login:"+clientIP, 5, time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"detail": "Too many requests"})
			return
		}

		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}

		fakeUsersMu.RLock()
		var foundUser *UserInDB
		for name, u := range fakeUsersDB {
			if subtle.ConstantTimeCompare([]byte(name), []byte(req.Username)) == 1 {
				foundUser = u
				break
			}
		}
		fakeUsersMu.RUnlock()

		if foundUser == nil {
			c.JSON(http.StatusNotFound, gin.H{"detail": "User not found"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(foundUser.HashedPassword), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Authorization failed"})
			return
		}

		token, err := generateJWT(foundUser.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, TokenResponse{AccessToken: token, TokenType: "bearer"})
	})

	// ==================== Задание 6.4: Защищённый ресурс ====================
	protected := router.Group("/", jwtAuthMiddleware())
	{
		protected.GET("/protected_resource", func(c *gin.Context) {
			username := c.GetString("username")
			c.JSON(http.StatusOK, gin.H{"message": "Access granted", "user": username})
		})

		// ==================== Задание 7.1: RBAC ====================
		// Админ: полный CRUD
		adminGroup := protected.Group("/admin", roleRequired("admin"))
		{
			adminGroup.POST("/resource", func(c *gin.Context) {
				c.JSON(http.StatusCreated, gin.H{"message": "Resource created by admin"})
			})
			adminGroup.GET("/resource", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Resource read by admin"})
			})
			adminGroup.PUT("/resource", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Resource updated by admin"})
			})
			adminGroup.DELETE("/resource", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Resource deleted by admin"})
			})
		}

		// Пользователь: чтение и обновление
		userGroup := protected.Group("/user", roleRequired("admin", "user"))
		{
			userGroup.GET("/resource", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Resource read by user"})
			})
			userGroup.PUT("/resource", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Resource updated by user"})
			})
		}

		// Гость: только чтение
		guestGroup := protected.Group("/guest", roleRequired("admin", "user", "guest"))
		{
			guestGroup.GET("/resource", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Resource read by guest"})
			})
		}
	}

	// ==================== Задание 8.1: PostgreSQL регистрация ====================
	router.POST("/db/register", func(c *gin.Context) {
		var req User
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}

		_, err := db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"detail": "Registration failed: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User registered successfully!"})
	})

	// ==================== Задание 8.2: CRUD Todo ====================
	router.POST("/todos", func(c *gin.Context) {
		var todo Todo
		if err := c.ShouldBindJSON(&todo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}

		var id int
		err := db.QueryRow("INSERT INTO todos (title, description, completed) VALUES ($1, $2, $3) RETURNING id",
			todo.Title, todo.Description, false).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			return
		}

		todo.ID = id
		todo.Completed = false
		c.JSON(http.StatusCreated, todo)
	})

	router.GET("/todos/:id", func(c *gin.Context) {
		id := c.Param("id")
		var todo Todo
		err := db.QueryRow("SELECT id, title, description, completed FROM todos WHERE id = $1", id).
			Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Completed)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Todo not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, todo)
	})

	router.PUT("/todos/:id", func(c *gin.Context) {
		id := c.Param("id")

		var update TodoUpdate
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}

		result, err := db.Exec("UPDATE todos SET title = $1, description = $2, completed = $3 WHERE id = $4",
			update.Title, update.Description, update.Completed, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Todo not found"})
			return
		}

		var todo Todo
		db.QueryRow("SELECT id, title, description, completed FROM todos WHERE id = $1", id).
			Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Completed)
		c.JSON(http.StatusOK, todo)
	})

	router.DELETE("/todos/:id", func(c *gin.Context) {
		id := c.Param("id")

		result, err := db.Exec("DELETE FROM todos WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Todo not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Todo deleted successfully"})
	})

	fmt.Println("Starting server on :8080...")
	fmt.Printf("Mode: %s\n", mode)
	router.Run(":8080")
}

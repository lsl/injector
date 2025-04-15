// File is commented out for now due to main function conflicts
// To use this file, remove the main functions from other example files

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/lsl/injector"
)

// User represents a user in our system
type User struct {
	ID   int
	Name string
}

// UserRepo handles user data access
type UserRepo struct {
	users map[int]*User
}

// NewUserRepo creates a new user repository with some sample data
func NewUserRepo() *UserRepo {
	return &UserRepo{
		users: map[int]*User{
			1: {ID: 1, Name: "Alice"},
			2: {ID: 2, Name: "Bob"},
			3: {ID: 3, Name: "Charlie"},
		},
	}
}

// GetByID retrieves a user by ID, returning (user, found)
func (r *UserRepo) GetByID(id int) (*User, bool) {
	user, found := r.users[id]
	return user, found
}

// AppLogger is a simple wrapper around log.Logger
type AppLogger struct {
	*log.Logger
}

// NewLogger creates a new logger
func NewLogger() *AppLogger {
	return &AppLogger{log.New(os.Stdout, "[INJECTOR] ", log.LstdFlags)}
}

// AuthMiddleware extracts a user ID from the request and injects the user into the context
func AuthMiddleware(userRepo *UserRepo, logger *AppLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from query parameter
			userIDStr := r.URL.Query().Get("user_id")
			if userIDStr == "" {
				logger.Println("No user_id provided, proceeding as guest")
				next.ServeHTTP(w, r)
				return
			}

			// Parse user ID
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				logger.Printf("Invalid user_id: %s", userIDStr)
				http.Error(w, "Invalid user ID", http.StatusBadRequest)
				return
			}

			// Look up user
			user, found := userRepo.GetByID(userID)
			if !found {
				logger.Printf("User not found: %d", userID)
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			// Inject user into context
			logger.Printf("User found: %s (ID: %d)", user.Name, user.ID)
			ctx := injector.WithValue(r.Context(), user)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// Simple handlers demonstrating different injection scenarios

// HomeHandler is a basic handler with no injection
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Injector example!\n\n")
	fmt.Fprintf(w, "Try these endpoints:\n")
	fmt.Fprintf(w, "- /hello - Basic handler (no parameters needed)\n")
	fmt.Fprintf(w, "- /me?user_id=1 - Shows user info (requires user_id)\n")
	fmt.Fprintf(w, "- /greet?user_id=1 - Greets the user (requires user_id)\n")
	fmt.Fprintf(w, "- /admin - Admin page (uses repository directly)\n")
}

// LogHandler demonstrates a simple injected logger
func LogHandler(w http.ResponseWriter, r *http.Request, logger *AppLogger) {
	logger.Println("Hello handler called")
	fmt.Fprintf(w, "Hello, World!")
}

// UserInfoHandler demonstrates injecting both a user and a logger
func UserInfoHandler(w http.ResponseWriter, r *http.Request, user *User, logger *AppLogger) {
	logger.Printf("Showing info for user: %s", user.Name)
	fmt.Fprintf(w, "User: %s (ID: %d)", user.Name, user.ID)
}

// GreetingHandler demonstrates injecting a user
func GreetingHandler(w http.ResponseWriter, r *http.Request, user *User) {
	fmt.Fprintf(w, "Hello, %s!", user.Name)
}

// AdminHandler demonstrates injecting a repository
func AdminHandler(w http.ResponseWriter, r *http.Request, userRepo *UserRepo, logger *AppLogger) {
	logger.Println("Admin page accessed")
	fmt.Fprintf(w, "All Users:\n\n")

	for id, user := range userRepo.users {
		fmt.Fprintf(w, "- User %d: %s\n", id, user.Name)
	}
}

func main() {
	// Create our dependencies
	logger := NewLogger()
	userRepo := NewUserRepo()

	// Debug log to see what type is getting registered
	fmt.Printf("Logger type: %T\n", logger)

	// Register dependencies for injection
	injector.Register(userRepo)

	// Explicitly register AppLogger with the exact type needed by handlers
	injector.RegisterInjector(func(r *http.Request) *AppLogger {
		return logger
	})

	// Register user injection from context
	injector.RegisterInjector(func(r *http.Request) *User {
		user, ok := injector.Try[*User](r.Context())
		if !ok {
			// This will cause handlers that require User to fail
			// when no user is in the context - this shouldn't happen
			// so long as the middleware is setup to error on
			// missing user / user_id
			panic("No user found in request context")
		}
		return user
	})

	// Create router and set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/hello", injector.Injected(LogHandler))
	mux.HandleFunc("/me", injector.Injected(UserInfoHandler))
	mux.HandleFunc("/greet", injector.Injected(GreetingHandler))
	mux.HandleFunc("/admin", injector.Injected(AdminHandler))

	// Apply middleware
	handler := AuthMiddleware(userRepo, logger)(mux)

	// Start the server
	serverAddr := ":8080"
	logger.Printf("Server starting on http://localhost%s", serverAddr)
	logger.Fatal(http.ListenAndServe(serverAddr, handler))
}

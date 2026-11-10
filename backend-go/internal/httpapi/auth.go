package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type adminStore interface {
	QueryOne(context.Context, string, ...any) (map[string]any, error)
}

type Claims struct {
	Username string `json:"-"`
	jwt.RegisteredClaims
}

type Auth struct {
	Store   adminStore
	Secret  []byte
	Limiter *RateLimiter
	MaxAge  time.Duration
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&request) != nil {
		writeUnreadableBody(w)
		return
	}
	// Java validates the DTO before touching either rate limit, so a malformed
	// login must not consume the caller's quota.
	if fieldErrors := loginValidationErrors(request.Username, request.Password); len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	ip := ClientIP(r)
	// Per IP: every attempt counts (Java MAX_ATTEMPTS_PER_IP).
	if !a.Limiter.Allow("login-ip:"+ip, 20, 15*time.Minute) {
		writeError(w, http.StatusTooManyRequests, "Too many login attempts. Please try again later.")
		return
	}
	// The limiter key is trimmed and lowercased, but the account lookup is not:
	// Java's AdminUserDetailsService uses findByUsername, which is case- and
	// whitespace-sensitive, so "Admin" must not authenticate "admin".
	normalized := strings.ToLower(strings.TrimSpace(request.Username))
	if utf8.RuneCountInString(normalized) > 128 {
		digest := sha256.Sum256([]byte(normalized))
		normalized = "sha256:" + hex.EncodeToString(digest[:])
	}
	key := "login-user:" + ip + "|" + normalized
	// Per username + IP: only failures count, so a successful login must peek
	// rather than consume (Java uses isLimited + record/reset).
	if a.Limiter.IsLimited(key, 5, 15*time.Minute) {
		writeError(w, http.StatusTooManyRequests, "Too many login attempts. Please try again later.")
		return
	}
	row, err := a.Store.QueryOne(r.Context(), `SELECT username, email, password FROM admins WHERE username = $1`, request.Username)
	if err != nil {
		a.Limiter.Record(key, 15*time.Minute)
		writeError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	passwordHash, _ := row["password"].(string)
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(request.Password)) != nil {
		a.Limiter.Record(key, 15*time.Minute)
		writeError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   row["username"].(string),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.MaxAge)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.Secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Login failed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":    signed,
		"username": row["username"],
		"email":    row["email"],
	})
	// Java resets the per-username counter after a successful login.
	a.Limiter.Reset(key)
}

// loginValidationErrors mirrors LoginRequest's @NotBlank constraints.
func loginValidationErrors(username, password string) map[string]string {
	fieldErrors := map[string]string{}
	if strings.TrimSpace(username) == "" {
		fieldErrors["username"] = "Username is required"
	}
	if password == "" {
		fieldErrors["password"] = "Password is required"
	}
	return fieldErrors
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeEmpty(w, http.StatusUnauthorized)
			return
		}
		claims := &Claims{}
		parsed, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), claims, func(token *jwt.Token) (any, error) {
			// JJWT signs with HS256/384/512 depending on the secret length, so a
			// token Java issued must verify here too. Any HMAC method proves the
			// same shared secret, and "none" stays rejected.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return a.Secret, nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
		if err != nil || !parsed.Valid || strings.TrimSpace(claims.Subject) == "" {
			writeEmpty(w, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), adminContextKey{}, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(authenticatedUsername(r.Context())) == "" {
			writeEmpty(w, http.StatusUnauthorized)
			return
		}
		// Java re-loads the account on every request and requires ROLE_ADMIN,
		// so a subject that is not (or no longer) an admin must not pass here
		// either, even though the JWT itself is valid.
		if !a.isAdmin(r.Context(), authenticatedUsername(r.Context())) {
			writeEmpty(w, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isAdmin mirrors Java's AdminUserDetailsService: every account in the admins
// table carries ROLE_ADMIN, so membership of that table is the role check.
func (a *Auth) isAdmin(ctx context.Context, username string) bool {
	if a == nil || a.Store == nil {
		return false
	}
	row, err := a.Store.QueryOne(ctx, `SELECT EXISTS(SELECT 1 FROM admins WHERE username=$1) AS found`, username)
	if err != nil {
		return false
	}
	found, _ := row["found"].(bool)
	return found
}

type adminContextKey struct{}

func authenticatedUsername(ctx context.Context) string {
	value, _ := ctx.Value(adminContextKey{}).(string)
	return value
}

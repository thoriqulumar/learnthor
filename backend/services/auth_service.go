package services

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
)

// AuthService handles registration, login, and JWT issuance.
type AuthService struct {
	db *gorm.DB
}

// NewAuthService constructs an AuthService.
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

// jwtSecret reads JWT_SECRET from the environment. Panics at startup if unset.
func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		panic("JWT_SECRET environment variable is not set")
	}
	return []byte(s)
}

// ─────────────────────────────────────────────────────────────────────────────
// Register
// ─────────────────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// Register creates a new student account and returns a signed JWT.
func (s *AuthService) Register(req RegisterRequest) (*AuthResponse, error) {
	// Check duplicate email / username.
	var count int64
	s.db.Model(&models.User{}).Where("email = ? OR username = ?", req.Email, req.Username).Count(&count)
	if count > 0 {
		return nil, ErrBadRequest
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         models.UserRoleStudent,
		IsActive:     true,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	token, err := issueToken(user.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: user}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Login
// ─────────────────────────────────────────────────────────────────────────────

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login validates credentials and returns a signed JWT.
func (s *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBadRequest // don't reveal "user not found"
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrBadRequest
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrBadRequest
	}

	token, err := issueToken(user.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: user}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Preferences (PreferredGloss etc.)
// ─────────────────────────────────────────────────────────────────────────────

// UpdatePreferencesRequest carries the user-editable preference fields.
type UpdatePreferencesRequest struct {
	PreferredGloss string `json:"preferred_gloss,omitempty"`
}

// PreferencesResponse returns the user's current preference set.
type PreferencesResponse struct {
	PreferredGloss string `json:"preferred_gloss"`
}

// validGloss is the set of accepted preferred_gloss values.
var validGloss = map[string]bool{"en": true, "id": true, "both": true}

// GetPreferences returns the user's preferences.
func (s *AuthService) GetPreferences(userID uuid.UUID) (*PreferencesResponse, error) {
	var user models.User
	if err := s.db.Select("preferred_gloss").First(&user, "id = ?", userID).Error; err != nil {
		return nil, ErrNotFound
	}
	return &PreferencesResponse{PreferredGloss: user.PreferredGloss}, nil
}

// UpdatePreferences updates the user's preference fields. Only the supplied
// fields are touched; omitted fields keep their current value.
func (s *AuthService) UpdatePreferences(userID uuid.UUID, req UpdatePreferencesRequest) (*PreferencesResponse, error) {
	updates := map[string]any{}
	if req.PreferredGloss != "" {
		if !validGloss[req.PreferredGloss] {
			return nil, ErrBadRequest
		}
		updates["preferred_gloss"] = req.PreferredGloss
	}
	if len(updates) == 0 {
		return s.GetPreferences(userID)
	}
	if err := s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetPreferences(userID)
}

// ─────────────────────────────────────────────────────────────────────────────
// JWT helpers
// ─────────────────────────────────────────────────────────────────────────────

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// issueToken signs a JWT valid for 72 hours.
func issueToken(userID uuid.UUID) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret())
}

// ParseToken validates a JWT string and returns the embedded claims.
func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrBadRequest
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrBadRequest
	}
	return claims, nil
}

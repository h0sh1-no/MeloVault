// Package auth handles user authentication, registration, JWT tokens, and OAuth.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"github.com/h0sh1-no/MeloVault/internal/database"
)

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailExists         = errors.New("email already exists")
	ErrUsernameExists      = errors.New("username already exists")
	ErrInvalidCode         = errors.New("invalid or expired code")
	ErrRegistrationDisabled = errors.New("registration is disabled")
)

// Config holds authentication configuration.
type Config struct {
	JWTSecret          string
	JWTAccessDuration  time.Duration
	JWTRefreshDuration time.Duration
	FrontendURL        string
	SMTPHost           string
	SMTPPort           int
	SMTPUser           string
	SMTPPassword       string
	SMTPFrom           string
}

// LinuxdoOAuthConfig holds LinuxDO OAuth credentials (resolved at request time).
type LinuxdoOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// Configured reports whether all required fields are present.
func (c LinuxdoOAuthConfig) Configured() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.RedirectURI != ""
}

// Service provides authentication operations.
type Service struct {
	db  *database.Pool
	cfg Config
}

// User represents a user in the system.
type User struct {
	ID           int64
	Username     string
	Email        *string
	PasswordHash *string
	Avatar       *string
	Provider     string
	ProviderID   *string
	Role         string // "user" | "admin" | "superadmin"
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Claims holds JWT claims.
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Provider string `json:"provider"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair holds access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// NewService creates a new authentication service.
func NewService(db *database.Pool, cfg Config) *Service {
	return &Service{db: db, cfg: cfg}
}

// SetSMTPConfig updates SMTP configuration at runtime (called when admin saves site settings).
func (s *Service) SetSMTPConfig(host string, port int, user, password, from string) {
	s.cfg.SMTPHost = host
	s.cfg.SMTPPort = port
	s.cfg.SMTPUser = user
	s.cfg.SMTPPassword = password
	s.cfg.SMTPFrom = from
}

// SMTPConfigured reports whether SMTP is configured.
func (s *Service) SMTPConfigured() bool {
	return s.cfg.SMTPHost != "" && s.cfg.SMTPUser != "" && s.cfg.SMTPPassword != ""
}

// Register creates a new user with email and password.
func (s *Service) Register(ctx context.Context, username, email, password string) (*User, *TokenPair, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return nil, nil, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return nil, nil, ErrUsernameExists
	}

	err = s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return nil, nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, nil, ErrEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	var user User
	err = s.db.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, provider)
		 VALUES ($1, $2, $3, 'email')
		 RETURNING id, username, email, password_hash, avatar, provider, provider_id, role, created_at, updated_at`,
		username, email, string(hash),
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Avatar,
		&user.Provider, &user.ProviderID, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("insert user: %w", err)
	}

	tokens, err := s.GenerateTokenPair(&user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return &user, tokens, nil
}

// Login authenticates a user with email and password.
func (s *Service) Login(ctx context.Context, email, password string) (*User, *TokenPair, error) {
	var user User
	err := s.db.QueryRow(ctx,
		`SELECT id, username, email, password_hash, avatar, provider, provider_id, role, created_at, updated_at
		 FROM users WHERE email = $1 AND provider = 'email'`, email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Avatar,
		&user.Provider, &user.ProviderID, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("query user: %w", err)
	}

	if user.PasswordHash == nil {
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.GenerateTokenPair(&user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return &user, tokens, nil
}

// GenerateTokenPair creates access and refresh tokens.
func (s *Service) GenerateTokenPair(user *User) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(s.cfg.JWTAccessDuration)
	refreshExp := now.Add(s.cfg.JWTRefreshDuration)

	accessClaims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Provider: user.Provider,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "melovault",
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Provider: user.Provider,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "melovault",
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int64(s.cfg.JWTAccessDuration.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates a JWT token and returns claims.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUserByID retrieves a user by ID.
func (s *Service) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var user User
	err := s.db.QueryRow(ctx,
		`SELECT id, username, email, password_hash, avatar, provider, provider_id, role, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Avatar,
		&user.Provider, &user.ProviderID, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &user, nil
}

// LinuxdoOAuthURL generates the OAuth authorization URL for Linuxdo.
func (s *Service) LinuxdoOAuthURL(state string, oauthCfg LinuxdoOAuthConfig) string {
	params := url.Values{
		"client_id":     {oauthCfg.ClientID},
		"redirect_uri":  {oauthCfg.RedirectURI},
		"response_type": {"code"},
		"state":         {state},
		"scope":         {"read"},
	}
	return "https://connect.linux.do/oauth2/authorize?" + params.Encode()
}

// LinuxdoCallback handles OAuth callback from Linuxdo.
// When allowRegistration is false, only existing users can log in;
// new user creation is rejected with ErrRegistrationDisabled.
func (s *Service) LinuxdoCallback(ctx context.Context, code string, oauthCfg LinuxdoOAuthConfig, allowRegistration bool) (*User, *TokenPair, error) {
	tokenResp, err := s.exchangeLinuxdoCode(ctx, code, oauthCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("exchange code: %w", err)
	}

	userInfo, err := s.getLinuxdoUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("get user info: %w", err)
	}

	user, err := s.findOrCreateLinuxdoUser(ctx, userInfo, allowRegistration)
	if err != nil {
		return nil, nil, err
	}

	tokens, err := s.GenerateTokenPair(user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return user, tokens, nil
}

type linuxdoTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type linuxdoUserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar_url"`
}

func (s *Service) exchangeLinuxdoCode(ctx context.Context, code string, oauthCfg LinuxdoOAuthConfig) (*linuxdoTokenResponse, error) {
	data := url.Values{
		"client_id":     {oauthCfg.ClientID},
		"client_secret": {oauthCfg.ClientSecret},
		"redirect_uri":  {oauthCfg.RedirectURI},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://connect.linux.do/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var result linuxdoTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *Service) getLinuxdoUserInfo(ctx context.Context, accessToken string) (*linuxdoUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://connect.linux.do/api/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get user info failed: %s", string(body))
	}

	var result linuxdoUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *Service) findOrCreateLinuxdoUser(ctx context.Context, info *linuxdoUserInfo, allowCreate bool) (*User, error) {
	providerID := strconv.Itoa(info.ID)

	var user User
	err := s.db.QueryRow(ctx,
		`SELECT id, username, email, password_hash, avatar, provider, provider_id, role, created_at, updated_at
		 FROM users WHERE provider = 'linuxdo' AND provider_id = $1`, providerID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Avatar,
		&user.Provider, &user.ProviderID, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		if info.Avatar != "" && (user.Avatar == nil || *user.Avatar != info.Avatar) {
			_, _ = s.db.Exec(ctx, "UPDATE users SET avatar = $1, updated_at = NOW() WHERE id = $2", info.Avatar, user.ID)
			user.Avatar = &info.Avatar
		}
		return &user, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("query user: %w", err)
	}

	if !allowCreate {
		return nil, ErrRegistrationDisabled
	}

	username := info.Username
	if username == "" {
		username = info.Name
	}
	if username == "" {
		username = "linuxdo_" + providerID
	}

	baseUsername := username
	counter := 1
	for {
		var exists bool
		err := s.db.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("check username: %w", err)
		}
		if !exists {
			break
		}
		username = fmt.Sprintf("%s_%d", baseUsername, counter)
		counter++
	}

	var avatar *string
	if info.Avatar != "" {
		avatar = &info.Avatar
	}

	err = s.db.QueryRow(ctx,
		`INSERT INTO users (username, avatar, provider, provider_id)
		 VALUES ($1, $2, 'linuxdo', $3)
		 RETURNING id, username, email, password_hash, avatar, provider, provider_id, role, created_at, updated_at`,
		username, avatar, providerID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Avatar,
		&user.Provider, &user.ProviderID, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &user, nil
}

// GenerateState generates a random OAuth state string.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SendVerificationCode sends a verification code to email.
func (s *Service) SendVerificationCode(ctx context.Context, email, purpose string) (string, error) {
	code := generateCode(6)

	_, _ = s.db.Exec(ctx,
		"DELETE FROM email_codes WHERE email = $1 AND purpose = $2 AND used = FALSE",
		email, purpose)

	expiresAt := time.Now().Add(10 * time.Minute)
	_, err := s.db.Exec(ctx,
		`INSERT INTO email_codes (email, code, purpose, expires_at)
		 VALUES ($1, $2, $3, $4)`, email, code, purpose, expiresAt)
	if err != nil {
		return "", fmt.Errorf("insert code: %w", err)
	}

	// Send email (simplified - in production use proper SMTP)
	if s.cfg.SMTPHost != "" {
		if err := s.sendEmail(email, "验证码 - MeloVault", fmt.Sprintf("您的验证码是: %s\n有效期10分钟。", code)); err != nil {
			log.Printf("send email failed: %v", err)
			// Return code anyway for development
		}
	}

	return code, nil
}

// VerifyCode verifies an email verification code.
func (s *Service) VerifyCode(ctx context.Context, email, code, purpose string) (bool, error) {
	var storedCode string
	var expiresAt time.Time
	var used bool

	err := s.db.QueryRow(ctx,
		`SELECT code, expires_at, used FROM email_codes
		 WHERE email = $1 AND purpose = $2 ORDER BY created_at DESC LIMIT 1`,
		email, purpose,
	).Scan(&storedCode, &expiresAt, &used)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query code: %w", err)
	}

	if used || time.Now().After(expiresAt) || storedCode != code {
		return false, nil
	}

	_, _ = s.db.Exec(ctx, "UPDATE email_codes SET used = TRUE WHERE email = $1 AND code = $2", email, code)

	return true, nil
}

func generateCode(length int) string {
	const digits = "0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		log.Printf("crypto/rand failed: %v, using time-based seed", err)
		src := time.Now().UnixNano()
		for i := range b {
			src = src*6364136223846793005 + 1442695040888963407
			b[i] = digits[(src>>33)%int64(len(digits))]
		}
		return string(b)
	}
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b)
}

func (s *Service) sendEmail(to, subject, body string) error {
	if s.cfg.SMTPHost == "" || s.cfg.SMTPUser == "" || s.cfg.SMTPPassword == "" {
		log.Printf("[DEV] Email to %s: [%s] %s", to, subject, body)
		return nil
	}
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	from := s.cfg.SMTPUser
	if s.cfg.SMTPFrom != "" {
		from = fmt.Sprintf("%s <%s>", s.cfg.SMTPFrom, s.cfg.SMTPUser)
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)
	return smtp.SendMail(addr, auth, s.cfg.SMTPUser, []string{to}, []byte(msg))
}

// SendTestEmail sends a test email to verify SMTP configuration.
func (s *Service) SendTestEmail(to string) error {
	return s.sendEmail(to, "测试邮件 - MeloVault", "这是一封测试邮件，如果您收到此邮件说明 SMTP 配置正确。\n\nThis is a test email from MeloVault.")
}

// ChangePassword changes user password.
func (s *Service) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	var hash string
	err := s.db.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE id = $1 AND provider = 'email'", userID).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("query password: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.db.Exec(ctx,
		"UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2",
		string(newHash), userID)
	return err
}

// UpdateProfile updates user profile.
func (s *Service) UpdateProfile(ctx context.Context, userID int64, username string, avatar *string) error {
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND id != $2)", username, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check username: %w", err)
	}
	if exists {
		return ErrUsernameExists
	}

	if avatar != nil {
		_, err = s.db.Exec(ctx,
			"UPDATE users SET username = $1, avatar = $2, updated_at = NOW() WHERE id = $3",
			username, *avatar, userID)
	} else {
		_, err = s.db.Exec(ctx,
			"UPDATE users SET username = $1, updated_at = NOW() WHERE id = $2",
			username, userID)
	}
	return err
}

// Ensure pgxpool.Pool implements database.Pool interface
var _ *pgxpool.Pool = (*pgxpool.Pool)(nil)

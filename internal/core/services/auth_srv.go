package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo ports.UserRepository
}

func NewAuthService(userRepo ports.UserRepository) ports.AuthService {
	return &authService{userRepo: userRepo}
}

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("ADVERTENCIA: JWT_SECRET no configurado. Defina JWT_SECRET en variables de entorno.")
		secret = "dev-only-unsafe-secret-must-change-in-production"
	}
	if len(secret) < 32 {
		log.Println("ADVERTENCIA: JWT_SECRET debe tener al menos 32 caracteres.")
	}
	jwtSecret = []byte(secret)
}

func getJWTSecret() []byte {
	return jwtSecret
}

func (s *authService) generateAccessToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "access",
		"exp":     time.Now().Add(1 * time.Hour).Unix(), // 1 hora
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func (s *authService) generateRefreshToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 días
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func (s *authService) Login(email, password string) (*domain.AuthResponse, error) {
	// Buscar usuario por email
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("credenciales inválidas")
	}

	// Verificar contraseña
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("credenciales inválidas")
	}

	// Generar tokens
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, errors.New("error generando token")
	}

	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, errors.New("error generando refresh token")
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

func (s *authService) RefreshToken(refreshToken string) (*domain.AuthResponse, error) {
	// Validar el refresh token
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de firma inválido")
		}
		return getJWTSecret(), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("refresh token inválido o expirado")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("token inválido")
	}

	// Verificar que sea un refresh token
	if claims["type"] != "refresh" {
		return nil, errors.New("tipo de token inválido")
	}

	// Obtener user_id
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return nil, errors.New("token inválido")
	}
	userID := uint(userIDFloat)

	// Verificar que el usuario sigue activo
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("usuario no encontrado")
	}

	// Generar nuevos tokens
	newAccessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, errors.New("error generando token")
	}

	newRefreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, errors.New("error generando refresh token")
	}

	return &domain.AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User:         *user,
	}, nil
}

func (s *authService) ValidateToken(tokenStr string) (uint, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de firma inválido")
		}
		return getJWTSecret(), nil
	})

	if err != nil || !token.Valid {
		return 0, errors.New("token inválido o expirado")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("token inválido")
	}

	if claims["type"] != "access" {
		return 0, errors.New("tipo de token inválido")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("token inválido: user_id ausente")
	}
	return uint(userIDFloat), nil
}

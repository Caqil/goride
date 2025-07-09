package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JWTClaims struct {
	UserID   primitive.ObjectID `json:"user_id"`
	UserType string             `json:"user_type"`
	Email    string             `json:"email"`
	Phone    string             `json:"phone"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func GenerateTokenPair(userID primitive.ObjectID, userType, email, phone, secretKey string) (*TokenPair, error) {
	// Access Token
	accessClaims := &JWTClaims{
		UserID:   userID,
		UserType: userType,
		Email:    email,
		Phone:    phone,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWTAccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    AppName,
			Subject:   userID.Hex(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(secretKey))
	if err != nil {
		return nil, err
	}

	// Refresh Token
	refreshClaims := &JWTClaims{
		UserID:   userID,
		UserType: userType,
		Email:    email,
		Phone:    phone,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWTRefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    AppName,
			Subject:   userID.Hex(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(secretKey))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(JWTAccessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func ValidateToken(tokenString, secretKey string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func RefreshAccessToken(refreshTokenString, secretKey string) (*TokenPair, error) {
	claims, err := ValidateToken(refreshTokenString, secretKey)
	if err != nil {
		return nil, err
	}

	return GenerateTokenPair(claims.UserID, claims.UserType, claims.Email, claims.Phone, secretKey)
}

func ExtractUserIDFromToken(tokenString, secretKey string) (primitive.ObjectID, error) {
	claims, err := ValidateToken(tokenString, secretKey)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return claims.UserID, nil
}

package oauth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AppleOAuthProvider struct {
	clientID    string
	teamID      string
	keyID       string
	privateKey  *rsa.PrivateKey
	redirectURL string
}

func NewAppleOAuthProvider(clientID, teamID, keyID, keyFile, redirectURL string) (*AppleOAuthProvider, error) {
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &AppleOAuthProvider{
		clientID:    clientID,
		teamID:      teamID,
		keyID:       keyID,
		privateKey:  privateKey,
		redirectURL: redirectURL,
	}, nil
}

func (a *AppleOAuthProvider) GetAuthURL(state string, scopes []string) string {
	if len(scopes) == 0 {
		scopes = []string{"name", "email"}
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", a.clientID)
	params.Set("redirect_uri", a.redirectURL)
	params.Set("scope", strings.Join(scopes, " "))
	params.Set("state", state)
	params.Set("response_mode", "form_post")

	return "https://appleid.apple.com/auth/authorize?" + params.Encode()
}

func (a *AppleOAuthProvider) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	clientSecret, err := a.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", a.redirectURL)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://appleid.apple.com/auth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple API error: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		IDToken      string `json:"id_token"`
	}

	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}

func (a *AppleOAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Apple doesn't provide a user info endpoint
	// User information is typically in the ID token from the auth response
	return nil, fmt.Errorf("GetUserInfo not supported by Apple OAuth - use ID token from auth response")
}

func (a *AppleOAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	clientSecret, err := a.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://appleid.apple.com/auth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple API error: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}

func (a *AppleOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	clientSecret, err := a.generateClientSecret()
	if err != nil {
		return fmt.Errorf("failed to generate client secret: %w", err)
	}

	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("client_secret", clientSecret)
	data.Set("token", token)
	data.Set("token_type_hint", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://appleid.apple.com/auth/revoke", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Apple API error: %s", string(body))
	}

	return nil
}

func (a *AppleOAuthProvider) generateClientSecret() (string, error) {
	claims := jwt.MapClaims{
		"iss": a.teamID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * 24 * 30 * 6).Unix(),
		"aud": "https://appleid.apple.com",
		"sub": a.clientID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = a.keyID

	return token.SignedString(a.privateKey)
}

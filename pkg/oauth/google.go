package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleOAuthProvider struct {
	config *oauth2.Config
}

func NewGoogleOAuthProvider(clientID, clientSecret, redirectURL string) *GoogleOAuthProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	return &GoogleOAuthProvider{
		config: config,
	}
}

func (g *GoogleOAuthProvider) GetAuthURL(state string, scopes []string) string {
	if len(scopes) > 0 {
		g.config.Scopes = scopes
	}

	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

func (g *GoogleOAuthProvider) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return &TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.Expiry.Unix(),
		Scope:        strings.Join(g.config.Scopes, " "),
	}, nil
}

func (g *GoogleOAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	url := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API error: %s", string(body))
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Locale        string `json:"locale"`
	}

	err = json.Unmarshal(body, &googleUser)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &UserInfo{
		ID:            googleUser.ID,
		Email:         googleUser.Email,
		EmailVerified: googleUser.VerifiedEmail,
		FirstName:     googleUser.GivenName,
		LastName:      googleUser.FamilyName,
		Name:          googleUser.Name,
		Picture:       googleUser.Picture,
		Locale:        googleUser.Locale,
		Provider:      "google",
	}, nil
}

func (g *GoogleOAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := g.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		ExpiresIn:    newToken.Expiry.Unix(),
	}, nil
}

func (g *GoogleOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	revokeURL := "https://oauth2.googleapis.com/revoke?token=" + token

	resp, err := http.Post(revokeURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Google API error: %s", string(body))
	}

	return nil
}

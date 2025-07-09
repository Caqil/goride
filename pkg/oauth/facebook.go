package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

type FacebookOAuthProvider struct {
	config *oauth2.Config
}

func NewFacebookOAuthProvider(appID, appSecret, redirectURL string) *FacebookOAuthProvider {
	config := &oauth2.Config{
		ClientID:     appID,
		ClientSecret: appSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"email", "public_profile"},
		Endpoint:     facebook.Endpoint,
	}

	return &FacebookOAuthProvider{
		config: config,
	}
}

func (f *FacebookOAuthProvider) GetAuthURL(state string, scopes []string) string {
	if len(scopes) > 0 {
		f.config.Scopes = scopes
	}

	return f.config.AuthCodeURL(state)
}

func (f *FacebookOAuthProvider) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	token, err := f.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return &TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.Expiry.Unix(),
		Scope:        strings.Join(f.config.Scopes, " "),
	}, nil
}

func (f *FacebookOAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	fields := "id,email,first_name,last_name,name,picture,locale"
	url := fmt.Sprintf("https://graph.facebook.com/me?fields=%s&access_token=%s", fields, accessToken)

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
		return nil, fmt.Errorf("Facebook API error: %s", string(body))
	}

	var fbUser struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Name      string `json:"name"`
		Picture   struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
		Locale string `json:"locale"`
	}

	err = json.Unmarshal(body, &fbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &UserInfo{
		ID:            fbUser.ID,
		Email:         fbUser.Email,
		EmailVerified: true, // Facebook emails are verified by default
		FirstName:     fbUser.FirstName,
		LastName:      fbUser.LastName,
		Name:          fbUser.Name,
		Picture:       fbUser.Picture.Data.URL,
		Locale:        fbUser.Locale,
		Provider:      "facebook",
	}, nil
}

func (f *FacebookOAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	// Facebook doesn't use refresh tokens in the traditional sense
	// Long-lived tokens can be exchanged for new long-lived tokens
	return nil, fmt.Errorf("refresh token not supported by Facebook")
}

func (f *FacebookOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	url := fmt.Sprintf("https://graph.facebook.com/me/permissions?access_token=%s", token)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Facebook API error: %s", string(body))
	}

	return nil
}

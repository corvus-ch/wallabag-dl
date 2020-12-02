package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type HttpClient interface {
	PostForm(url string, data url.Values) (resp *http.Response, err error)
}

type CredentialStore interface {
	Username() string
	Password() string
}

type Client struct {
	httpClient   HttpClient
	baseURL      string
	clientId     string
	clientSecret string
	credentials  CredentialStore

	token *token
}

type token struct {
	AccessToken    string    `json:"access_token"`
	ExpirationTime time.Time `json:"expiration_time"`
	TokenType      string    `json:"token_type"`
	Scope          string    `json:"scope"`
	RefreshToken   string    `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

func New(httpClient HttpClient, baseURL, clientId, clientSecret string, credentials CredentialStore) *Client {
	c := &Client{
		httpClient:   httpClient,
		baseURL:      baseURL,
		clientId:     clientId,
		clientSecret: clientSecret,
		credentials:  credentials,
		token:        &token{},
	}

	tokenFile, err := os.Open("token.json")
	if tokenFile != nil {
		defer tokenFile.Close()
	}
	if err == nil {
		json.NewDecoder(tokenFile).Decode(c.token)
	}

	return c
}

func (c *Client) AuthHeader() (string, error) {
	var refreshErr error
	if c.token.ExpirationTime.Before(time.Now()) {
		refreshErr = c.RefreshToken()
	}

	if refreshErr != nil || c.token.TokenType == "" || c.token.AccessToken == "" {
		if err := c.GetToken(); err != nil {
			return "", err
		}
	}

	return strings.Title(c.token.TokenType) + " " + c.token.AccessToken, nil
}

func (c *Client) GetToken() error {
	resp, err := c.httpClient.PostForm(c.tokenURL(), url.Values{
		"grant_type":    {"password"},
		"client_id":     {c.clientId},
		"client_secret": {c.clientSecret},
		"username":      {c.credentials.Username()},
		"password":      {c.credentials.Password()},
	})
	if err != nil {
		return err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("getToken: bad token from server: %v", resp.StatusCode)
	}

	if err := c.parseTokenResponse(resp.Body); err != nil {
		return err
	}

	return c.saveToken()
}

func (c *Client) RefreshToken() error {
	resp, err := c.httpClient.PostForm(c.tokenURL(), url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.clientId},
		"client_secret": {c.clientSecret},
		"refresh_token": {c.token.RefreshToken},
	})
	if err != nil {
		return err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refreshToken: bad token from server: %v", resp.StatusCode)
	}

	if err := c.parseTokenResponse(resp.Body); err != nil {
		return err
	}

	return c.saveToken()
}

func (c *Client) tokenURL() string {
	return c.baseURL + "/oauth/v2/token"
}

func (c *Client) parseTokenResponse(reader io.Reader) error {
	var response tokenResponse
	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		return err
	}

	c.token.ExpirationTime = time.Now().Add(time.Duration(response.ExpiresIn) * time.Second)
	c.token.AccessToken = response.AccessToken
	c.token.TokenType = response.TokenType
	c.token.RefreshToken = response.RefreshToken
	c.token.Scope = response.Scope

	return nil
}

func (c *Client) saveToken() error {
	tokenFile, err := os.Create("token.json")
	if err != nil {
		return err
	}

	return json.NewEncoder(tokenFile).Encode(c.token)
}
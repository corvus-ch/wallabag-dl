package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
)

type HttpClient interface {
	PostForm(url string, data url.Values) (resp *http.Response, err error)
	Do(req *http.Request) (*http.Response, error)
}

type CredentialStore interface {
	Username() string
	Password() string
}

type Client struct {
	log          logr.Logger
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

func New(log logr.Logger, httpClient HttpClient, baseURL, clientId, clientSecret string, credentials CredentialStore) *Client {
	c := &Client{
		log:          log,
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

func (c *Client) Get(url string, data interface{}) error {
	resp, err := c.Request(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(data)
}

func (c *Client) Patch(url string, data map[string]interface{}) error {
	buf := &bytes.Buffer{}

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		return err
	}

	_, err := c.Request(http.MethodPatch, url, buf)

	return err
}

func (c *Client) Request(method, url string, body io.Reader) (*http.Response, error) {
	authHeader, err := c.AuthHeader()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", "application/json")

	return c.httpClient.Do(req)
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

package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/opsrelay/agent/internal/config"
	"github.com/opsrelay/agent/internal/core"
)

type Client struct {
	serverURL string
	http      *http.Client
}

type EnrollRequest struct {
	Token           string      `json:"token"`
	Name            string      `json:"name"`
	OS              core.OSInfo `json:"os"`
	AgentVersion    string      `json:"agent_version"`
	ProtocolVersion string      `json:"protocol_version"`
	Capabilities    []string    `json:"capabilities"`
}

type enrollResponse struct {
	OK   bool `json:"ok"`
	Data struct {
		DeviceID        string `json:"device_id"`
		DeviceSecret    string `json:"device_secret"`
		ProtocolVersion string `json:"protocol_version"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(serverURL string) *Client {
	return &Client{serverURL: strings.TrimRight(serverURL, "/"), http: http.DefaultClient}
}

func (client *Client) Enroll(ctx context.Context, request EnrollRequest) (config.Config, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return config.Config{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.serverURL+"/v1/agent/enroll", bytes.NewReader(raw))
	if err != nil {
		return config.Config{}, err
	}
	httpRequest.Header.Set("content-type", "application/json")
	response, err := client.http.Do(httpRequest)
	if err != nil {
		return config.Config{}, err
	}
	defer response.Body.Close()

	var decoded enrollResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return config.Config{}, err
	}
	if !decoded.OK {
		return config.Config{}, fmt.Errorf("%s: %s", decoded.Error.Code, decoded.Error.Message)
	}
	return config.Config{
		DeviceID:        decoded.Data.DeviceID,
		DeviceSecret:    decoded.Data.DeviceSecret,
		ProtocolVersion: decoded.Data.ProtocolVersion,
		AllowReboot:     false,
	}, nil
}

func (client *Client) Connect(ctx context.Context, deviceID string, deviceSecret string) (*websocket.Conn, *http.Response, error) {
	connectURL, err := url.Parse(client.serverURL)
	if err != nil {
		return nil, nil, err
	}
	if connectURL.Scheme == "https" {
		connectURL.Scheme = "wss"
	} else {
		connectURL.Scheme = "ws"
	}
	connectURL.Path = "/v1/agent/connect"
	query := connectURL.Query()
	query.Set("device_id", deviceID)
	query.Set("device_secret", deviceSecret)
	connectURL.RawQuery = query.Encode()
	return websocket.DefaultDialer.DialContext(ctx, connectURL.String(), nil)
}

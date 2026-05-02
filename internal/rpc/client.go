package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type JSONRPCRequest struct {
	Version string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  []any       `json:"params"`
}

type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

type Aria2Client struct {
	instanceID string
	baseURL    string
	secret     string
	httpClient *http.Client
}

func NewAria2Client(instanceID string, port int, secret string) *Aria2Client {
	return &Aria2Client{
		instanceID: instanceID,
		baseURL:    fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", port),
		secret:     secret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Aria2Client) Call(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	allParams := []any{fmt.Sprintf("token:%s", c.secret)}
	allParams = append(allParams, params...)

	reqBody := JSONRPCRequest{
		Version: "2.0",
		ID:      "1",
		Method:  method,
		Params:  allParams,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, respBody)
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}

func (c *Aria2Client) MultiCall(ctx context.Context, calls [][]any) ([]json.RawMessage, error) {
	params := []any{fmt.Sprintf("token:%s", c.secret)}
	methods := make([]any, len(calls))
	for i, call := range calls {
		methods[i] = map[string]any{
			"methodName": call[0],
			"params":     call[1:],
		}
	}
	params = append(params, methods...)

	result, err := c.Call(ctx, "system.multicall", params...)
	if err != nil {
		return nil, err
	}

	var results []json.RawMessage
	if err := json.Unmarshal(result, &results); err != nil {
		return nil, fmt.Errorf("unmarshal multicall result: %w", err)
	}
	return results, nil
}

func (c *Aria2Client) InstanceID() string { return c.instanceID }
func (c *Aria2Client) Secret() string     { return c.secret }
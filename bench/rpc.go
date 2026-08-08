package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
)

// Hand rolled JSON-RPC so the HTTP transport is fully under our control.
// The default Go transport allows 2 idle conns per host, which silently
// serialises concurrent senders behind TLS handshakes and would corrupt
// every concurrency number in this benchmark.
type Client struct {
	url string
	hc  *http.Client
	id  int64
}

func NewClient(url string, maxConns int) *Client {
	tr := &http.Transport{
		MaxIdleConns:        maxConns * 2,
		MaxIdleConnsPerHost: maxConns * 2,
		MaxConnsPerHost:     maxConns * 2,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	return &Client{url: url, hc: &http.Client{Transport: tr, Timeout: 30 * time.Second}}
}

type rpcErr struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (e *rpcErr) Error() string { return fmt.Sprintf("rpc %d: %s", e.Code, e.Message) }

type rpcResp struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcErr         `json:"error"`
}

func (c *Client) call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	if params == nil {
		params = []interface{}{}
	}
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out rpcResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode %q: %w", truncate(string(raw), 200), err)
	}
	if out.Error != nil {
		return nil, out.Error
	}
	return out.Result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Warm establishes the TLS connection so the first timed send is not
// distorted by a handshake.
func (c *Client) Warm(ctx context.Context) error {
	_, err := c.call(ctx, "getHealth")
	return err
}

func (c *Client) LatestBlockhash(ctx context.Context, commitment string) (solana.Hash, uint64, error) {
	res, err := c.call(ctx, "getLatestBlockhash", map[string]string{"commitment": commitment})
	if err != nil {
		return solana.Hash{}, 0, err
	}
	var v struct {
		Value struct {
			Blockhash            string `json:"blockhash"`
			LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
		} `json:"value"`
	}
	if err := json.Unmarshal(res, &v); err != nil {
		return solana.Hash{}, 0, err
	}
	h, err := solana.HashFromBase58(v.Value.Blockhash)
	return h, v.Value.LastValidBlockHeight, err
}

func (c *Client) BlockHeight(ctx context.Context) (uint64, error) {
	res, err := c.call(ctx, "getBlockHeight")
	if err != nil {
		return 0, err
	}
	var h uint64
	err = json.Unmarshal(res, &h)
	return h, err
}

func (c *Client) Slot(ctx context.Context) (uint64, error) {
	res, err := c.call(ctx, "getSlot")
	if err != nil {
		return 0, err
	}
	var s uint64
	err = json.Unmarshal(res, &s)
	return s, err
}

func (c *Client) SendTx(ctx context.Context, tx *solana.Transaction, skipPreflight bool) (string, error) {
	b, err := tx.MarshalBinary()
	if err != nil {
		return "", err
	}
	res, err := c.call(ctx, "sendTransaction",
		base64.StdEncoding.EncodeToString(b),
		map[string]interface{}{
			"encoding":            "base64",
			"skipPreflight":       skipPreflight,
			"preflightCommitment": "processed",
			"maxRetries":          0,
		})
	if err != nil {
		return "", err
	}
	var sig string
	err = json.Unmarshal(res, &sig)
	return sig, err
}

type AccountInfo struct {
	Owner solana.PublicKey
	Data  []byte
}

func (c *Client) AccountInfo(ctx context.Context, pk solana.PublicKey) (*AccountInfo, error) {
	res, err := c.call(ctx, "getAccountInfo", pk.String(),
		map[string]string{"encoding": "base64", "commitment": "processed"})
	if err != nil {
		return nil, err
	}
	var v struct {
		Value *struct {
			Owner string   `json:"owner"`
			Data  []string `json:"data"`
		} `json:"value"`
	}
	if err := json.Unmarshal(res, &v); err != nil {
		return nil, err
	}
	if v.Value == nil {
		return nil, nil
	}
	data, err := base64.StdEncoding.DecodeString(v.Value.Data[0])
	if err != nil {
		return nil, err
	}
	return &AccountInfo{Owner: solana.MustPublicKeyFromBase58(v.Value.Owner), Data: data}, nil
}

// CounterValue reads the u64 at offset 8, after the Anchor discriminator.
func (c *Client) CounterValue(ctx context.Context, pk solana.PublicKey) (uint64, error) {
	ai, err := c.AccountInfo(ctx, pk)
	if err != nil {
		return 0, err
	}
	if ai == nil {
		return 0, fmt.Errorf("account %s not found", pk)
	}
	if len(ai.Data) < 16 {
		return 0, fmt.Errorf("account %s too short (%d bytes)", pk, len(ai.Data))
	}
	return binary.LittleEndian.Uint64(ai.Data[8:16]), nil
}

func (c *Client) SignatureLanded(ctx context.Context, sig string) (bool, string, error) {
	res, err := c.call(ctx, "getSignatureStatuses", []string{sig},
		map[string]bool{"searchTransactionHistory": true})
	if err != nil {
		return false, "", err
	}
	var v struct {
		Value []*struct {
			ConfirmationStatus string          `json:"confirmationStatus"`
			Err                json.RawMessage `json:"err"`
		} `json:"value"`
	}
	if err := json.Unmarshal(res, &v); err != nil {
		return false, "", err
	}
	if len(v.Value) == 0 || v.Value[0] == nil {
		return false, "", nil
	}
	if len(v.Value[0].Err) > 0 && string(v.Value[0].Err) != "null" {
		return false, string(v.Value[0].Err), nil
	}
	return true, v.Value[0].ConfirmationStatus, nil
}

func (c *Client) Balance(ctx context.Context, pk solana.PublicKey) (uint64, error) {
	res, err := c.call(ctx, "getBalance", pk.String())
	if err != nil {
		return 0, err
	}
	var v struct {
		Value uint64 `json:"value"`
	}
	err = json.Unmarshal(res, &v)
	return v.Value, err
}

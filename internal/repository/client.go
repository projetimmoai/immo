// Package repository est la couche d'accès aux données : il parle à l'API
// REST (PostgREST) de Supabase avec la clé service_role, et convertit entre
// les types du domaine (internal/domain) et le JSON échangé avec l'API.
package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client est le client d'accès à l'API REST (PostgREST) de Supabase.
type Client struct {
	baseURL        string // ex: https://xxxx.supabase.co/rest/v1
	serviceRoleKey string
	httpClient     *http.Client
}

// NewClientFromEnv construit un Client à partir des variables d'environnement
// SUPABASE_URL et SUPABASE_SERVICE_ROLE_KEY.
func NewClientFromEnv() (*Client, error) {
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return nil, fmt.Errorf("repository: variable d'environnement SUPABASE_URL manquante")
	}
	key := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if key == "" {
		return nil, fmt.Errorf("repository: variable d'environnement SUPABASE_SERVICE_ROLE_KEY manquante")
	}
	return NewClient(supabaseURL, key), nil
}

// NewClient construit un Client à partir d'une URL de projet Supabase et
// d'une clé service_role fournies explicitement.
func NewClient(supabaseURL, serviceRoleKey string) *Client {
	return &Client{
		baseURL:        strings.TrimRight(supabaseURL, "/") + "/rest/v1",
		serviceRoleKey: serviceRoleKey,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError représente une erreur retournée par l'API REST (PostgREST).
type APIError struct {
	StatusCode int
	Message    string `json:"message"`
	Details    string `json:"details"`
	Hint       string `json:"hint"`
	Code       string `json:"code"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("repository: réponse %d de l'API (code=%s): %s", e.StatusCode, e.Code, e.Message)
}

// do exécute une requête HTTP vers l'API REST et décode la réponse JSON dans
// out (si non nil). body, si non nil, est encodé en JSON comme corps de la
// requête. prefer, si non vide, positionne l'en-tête PostgREST "Prefer"
// (ex: "return=representation" pour récupérer la ligne créée/modifiée).
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	return c.doWithPrefer(ctx, method, path, body, "", out)
}

func (c *Client) doWithPrefer(ctx context.Context, method, path string, body any, prefer string, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("repository: encodage JSON de la requête %s %s: %w", method, path, err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("repository: construction de la requête %s %s: %w", method, path, err)
	}
	req.Header.Set("apikey", c.serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("repository: appel %s %s: %w", method, path, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("repository: fermeture du corps de la réponse %s %s: %v", method, path, cerr)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("repository: lecture de la réponse %s %s: %w", method, path, err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		// Best-effort : le corps d'erreur est normalement du JSON PostgREST,
		// mais peut être autre chose (ex: page d'erreur d'une passerelle réseau).
		// Dans ce cas on retombe sur le corps brut ci-dessous.
		if len(respBody) > 0 {
			_ = json.Unmarshal(respBody, apiErr)
		}
		if apiErr.Message == "" {
			apiErr.Message = string(respBody)
		}
		return apiErr
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("repository: décodage JSON de la réponse %s %s: %w", method, path, err)
		}
	}
	return nil
}

// CallRPC appelle une fonction Postgres exposée en RPC par PostgREST
// (endpoint POST /rest/v1/rpc/<name>), utilisée pour les traitements
// nécessitant une atomicité multi-écritures que l'API REST ne peut pas
// garantir appel par appel.
func (c *Client) CallRPC(ctx context.Context, name string, args, out any) error {
	if err := c.do(ctx, http.MethodPost, "/rpc/"+name, args, out); err != nil {
		return fmt.Errorf("repository: appel RPC %s: %w", name, err)
	}
	return nil
}

// escapeFilterValue échappe une valeur pour l'utiliser dans un filtre PostgREST
// de type "colonne=eq.valeur" en query string.
func escapeFilterValue(v string) string {
	return url.QueryEscape(v)
}

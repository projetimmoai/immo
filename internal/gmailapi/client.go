// Package gmailapi est la couche d'accès à l'API Gmail v1 : authentification
// OAuth2 "application installée" (le compte surveillé est un Gmail
// personnel, un compte de service seul n'y a pas accès — contrairement à
// internal/drive), puis appels REST avec un client net/http fait main —
// même esprit que internal/repository (Supabase) et internal/drive.
package gmailapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// gmailScope est la portée demandée : lecture seule de la boîte mail. Cette
// étape du projet analyse les e-mails et décide comment les router, elle ne
// les modifie pas (pas d'envoi, pas de marquage) — élargir la portée
// (ex: gmail.modify) si un futur besoin l'exige (ex: marquer comme traité).
const gmailScope = "https://www.googleapis.com/auth/gmail.readonly"

// Client est le client d'accès à l'API REST Gmail v1, pour la boîte mail du
// titulaire du jeton OAuth2 ("me" dans les chemins de l'API).
type Client struct {
	baseURL     string
	tokenSource oauth2.TokenSource
	httpClient  *http.Client
}

// NewOAuthConfigFromEnv construit la configuration OAuth2 (identifiant/secret
// de l'application) à partir de la variable d'environnement
// GOOGLE_OAUTH_CLIENT_JSON (fichier "ID client OAuth" de type "Application
// de bureau", téléchargé depuis Google Cloud Console). Exportée pour être
// réutilisée par cmd/gmail-auth (autorisation ponctuelle).
func NewOAuthConfigFromEnv() (*oauth2.Config, error) {
	clientJSON := os.Getenv("GOOGLE_OAUTH_CLIENT_JSON")
	if clientJSON == "" {
		return nil, fmt.Errorf("gmailapi: variable d'environnement GOOGLE_OAUTH_CLIENT_JSON manquante")
	}
	cfg, err := google.ConfigFromJSON([]byte(clientJSON), gmailScope)
	if err != nil {
		return nil, fmt.Errorf("gmailapi: lecture du client OAuth2: %w", err)
	}
	return cfg, nil
}

// NewClientFromEnv construit un Client à partir de GOOGLE_OAUTH_CLIENT_JSON
// (identifiant/secret de l'application) et GOOGLE_OAUTH_TOKEN_JSON (jeton
// obtenu une fois pour toutes via `go run ./cmd/gmail-auth`, contenant le
// refresh_token qui permet de renouveler l'accès indéfiniment sans nouvelle
// intervention humaine).
func NewClientFromEnv(ctx context.Context) (*Client, error) {
	cfg, err := NewOAuthConfigFromEnv()
	if err != nil {
		return nil, err
	}
	tokenJSON := os.Getenv("GOOGLE_OAUTH_TOKEN_JSON")
	if tokenJSON == "" {
		return nil, fmt.Errorf("gmailapi: variable d'environnement GOOGLE_OAUTH_TOKEN_JSON manquante (cf. `go run ./cmd/gmail-auth` pour l'obtenir)")
	}
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &tok); err != nil {
		return nil, fmt.Errorf("gmailapi: lecture du jeton OAuth2 (GOOGLE_OAUTH_TOKEN_JSON): %w", err)
	}
	return NewClient(cfg.TokenSource(ctx, &tok)), nil
}

// NewClient construit un Client à partir d'une source de jetons OAuth2
// fournie explicitement (utile pour les tests).
func NewClient(tokenSource oauth2.TokenSource) *Client {
	return &Client{
		baseURL:     "https://gmail.googleapis.com/gmail/v1/users/me",
		tokenSource: tokenSource,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError représente une erreur retournée par l'API REST Gmail v1.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gmailapi: réponse %d de l'API: %s", e.StatusCode, e.Body)
}

func (c *Client) do(ctx context.Context, method, path string, out any) error {
	tok, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("gmailapi: obtention du jeton d'accès: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("gmailapi: construction de la requête %s %s: %w", method, path, err)
	}
	tok.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gmailapi: appel %s %s: %w", method, path, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("gmailapi: fermeture du corps de la réponse %s %s: %v", method, path, cerr)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gmailapi: lecture de la réponse %s %s: %w", method, path, err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("gmailapi: décodage JSON de la réponse %s %s: %w", method, path, err)
		}
	}
	return nil
}

type messageListResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
	NextPageToken string `json:"nextPageToken"`
}

// ListMessageIDs liste les ids de messages correspondant à une requête de
// recherche Gmail (syntaxe Gmail, ex: "is:unread", "after:2026/01/01"), en
// paginant si nécessaire.
func (c *Client) ListMessageIDs(ctx context.Context, query string) ([]string, error) {
	var ids []string
	pageToken := ""
	for {
		path := "/messages?q=" + url.QueryEscape(query)
		if pageToken != "" {
			path += "&pageToken=" + url.QueryEscape(pageToken)
		}
		var resp messageListResponse
		if err := c.do(ctx, http.MethodGet, path, &resp); err != nil {
			return nil, fmt.Errorf("gmailapi: listage des messages (q=%q): %w", query, err)
		}
		for _, m := range resp.Messages {
			ids = append(ids, m.ID)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return ids, nil
}

// GetMessage récupère un message par son id et le normalise (en-têtes
// extraits, corps texte/HTML décodés, pièces jointes listées) — voir
// Message et parseMessage (message.go).
func (c *Client) GetMessage(ctx context.Context, id string) (*Message, error) {
	path := "/messages/" + url.PathEscape(id) + "?format=full"
	var raw gmailMessage
	if err := c.do(ctx, http.MethodGet, path, &raw); err != nil {
		return nil, fmt.Errorf("gmailapi: récupération du message %s: %w", id, err)
	}
	return parseMessage(raw)
}

type attachmentBody struct {
	Data string `json:"data"`
	Size int64  `json:"size"`
}

// GetAttachment télécharge le contenu d'une pièce jointe (identifiée par son
// AttachmentID, cf. Attachment) d'un message.
func (c *Client) GetAttachment(ctx context.Context, messageID, attachmentID string) ([]byte, error) {
	path := "/messages/" + url.PathEscape(messageID) + "/attachments/" + url.PathEscape(attachmentID)
	var raw attachmentBody
	if err := c.do(ctx, http.MethodGet, path, &raw); err != nil {
		return nil, fmt.Errorf("gmailapi: récupération de la pièce jointe %s du message %s: %w", attachmentID, messageID, err)
	}
	return decodeBase64URL(raw.Data)
}

// Package storage est la couche d'accès au stockage de fichiers Supabase
// Storage — remplace internal/drive (Google Drive) : les fichiers vivent
// dans le même projet Supabase que la base de données, avec la même clé
// service_role, sans système externe ni identifiants séparés.
//
// Un seul bucket (Bucket) sert à tous les documents (contrats, sinistres,
// incidents, pièces jointes d'e-mails...), avec une convention de chemin
// par catégorie plutôt qu'un bucket par catégorie, ex :
// "COP3 - Residence Horizon/contrats/facture-ascenseur.pdf". Ce package ne
// connaît pas cette convention (pas plus que internal/repository ne connaît
// le sens métier des tables qu'il manipule) : c'est à l'appelant de
// construire le chemin.
//
// Contrairement à Drive, aucun "dossier" n'a besoin d'être créé à l'avance
// : un chemin existe implicitement dès qu'un fichier y est déposé. Ça
// élimine le problème de compensation à deux phases qu'avait
// service.CoproprieteService.CreateCopropriete avec Drive.
package storage

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

// Bucket est le bucket Supabase Storage utilisé pour tous les documents.
const Bucket = "documents"

// Client est le client d'accès à l'API REST Supabase Storage.
type Client struct {
	baseURL        string // ex: https://xxxx.supabase.co/storage/v1
	serviceRoleKey string
	httpClient     *http.Client
}

// NewClientFromEnv construit un Client à partir des variables d'environnement
// SUPABASE_URL et SUPABASE_SERVICE_ROLE_KEY — les mêmes que internal/repository,
// l'API Storage faisant partie du même projet Supabase.
func NewClientFromEnv() (*Client, error) {
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return nil, fmt.Errorf("storage: variable d'environnement SUPABASE_URL manquante")
	}
	key := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if key == "" {
		return nil, fmt.Errorf("storage: variable d'environnement SUPABASE_SERVICE_ROLE_KEY manquante")
	}
	return NewClient(supabaseURL, key), nil
}

// NewClient construit un Client à partir d'une URL de projet Supabase et
// d'une clé service_role fournies explicitement.
func NewClient(supabaseURL, serviceRoleKey string) *Client {
	return &Client{
		baseURL:        strings.TrimRight(supabaseURL, "/") + "/storage/v1",
		serviceRoleKey: serviceRoleKey,
		httpClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

// APIError représente une erreur retournée par l'API REST Supabase Storage.
type APIError struct {
	StatusCode int
	Message    string `json:"message"`
	ErrType    string `json:"error"`
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.ErrType
	}
	return fmt.Sprintf("storage: réponse %d de l'API: %s", e.StatusCode, msg)
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, contentType string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("storage: construction de la requête %s %s: %w", method, path, err)
	}
	req.Header.Set("apikey", c.serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("storage: appel %s %s: %w", method, path, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("storage: fermeture du corps de la réponse %s %s: %v", method, path, cerr)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("storage: lecture de la réponse %s %s: %w", method, path, err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if len(respBody) > 0 {
			_ = json.Unmarshal(respBody, apiErr)
		}
		if apiErr.Message == "" && apiErr.ErrType == "" {
			apiErr.Message = string(respBody)
		}
		return apiErr
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("storage: décodage JSON de la réponse %s %s: %w", method, path, err)
		}
	}
	return nil
}

// Upload dépose contenu à chemin dans Bucket, avec le type MIME donné.
// Écrase un fichier existant au même chemin (upsert) plutôt que d'échouer
// — idempotent, sûr à rejouer.
func (c *Client) Upload(ctx context.Context, chemin string, contenu io.Reader, contentType string) error {
	path := "/object/" + Bucket + "/" + encodeChemin(chemin)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, contenu)
	if err != nil {
		return fmt.Errorf("storage: construction de la requête d'upload %s: %w", chemin, err)
	}
	req.Header.Set("apikey", c.serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("storage: upload de %s: %w", chemin, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("storage: fermeture du corps de la réponse upload %s: %v", chemin, cerr)
		}
	}()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("storage: lecture de la réponse d'upload %s: %w", chemin, err)
	}
	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if len(respBody) > 0 {
			_ = json.Unmarshal(respBody, apiErr)
		}
		if apiErr.Message == "" && apiErr.ErrType == "" {
			apiErr.Message = string(respBody)
		}
		return fmt.Errorf("storage: upload de %s: %w", chemin, apiErr)
	}
	return nil
}

// signedURLResponse est la réponse de POST /object/sign/{bucket}/{chemin}.
type signedURLResponse struct {
	SignedURL string `json:"signedURL"`
}

// SignedURL retourne une URL temporaire (valable expiration) pour
// télécharger le fichier à chemin, sans rendre le bucket public.
func (c *Client) SignedURL(ctx context.Context, chemin string, expiration time.Duration) (string, error) {
	body, err := json.Marshal(map[string]any{"expiresIn": int(expiration.Seconds())})
	if err != nil {
		return "", fmt.Errorf("storage: encodage JSON de la requête d'URL signée pour %s: %w", chemin, err)
	}
	var resp signedURLResponse
	path := "/object/sign/" + Bucket + "/" + encodeChemin(chemin)
	if err := c.do(ctx, http.MethodPost, path, bytes.NewReader(body), "application/json", &resp); err != nil {
		return "", fmt.Errorf("storage: génération de l'URL signée pour %s: %w", chemin, err)
	}
	if resp.SignedURL == "" {
		return "", fmt.Errorf("storage: génération de l'URL signée pour %s: réponse sans signedURL", chemin)
	}
	base := strings.TrimSuffix(c.baseURL, "/storage/v1")
	return base + "/storage/v1" + resp.SignedURL, nil
}

// Delete supprime un ou plusieurs fichiers (par chemin exact) de Bucket.
func (c *Client) Delete(ctx context.Context, chemins ...string) error {
	if len(chemins) == 0 {
		return nil
	}
	body, err := json.Marshal(map[string]any{"prefixes": chemins})
	if err != nil {
		return fmt.Errorf("storage: encodage JSON de la requête de suppression: %w", err)
	}
	if err := c.do(ctx, http.MethodDelete, "/object/"+Bucket, bytes.NewReader(body), "application/json", nil); err != nil {
		return fmt.Errorf("storage: suppression de %v: %w", chemins, err)
	}
	return nil
}

// Object décrit un fichier retourné par List.
type Object struct {
	Nom string `json:"name"`
	ID  string `json:"id"`
}

// List liste les fichiers sous le préfixe donné (ex: la référence d'une
// copropriété, pour retrouver tous ses documents).
func (c *Client) List(ctx context.Context, prefixe string) ([]Object, error) {
	body, err := json.Marshal(map[string]any{"prefix": prefixe, "limit": 1000})
	if err != nil {
		return nil, fmt.Errorf("storage: encodage JSON de la requête de listage: %w", err)
	}
	var objets []Object
	if err := c.do(ctx, http.MethodPost, "/object/list/"+Bucket, bytes.NewReader(body), "application/json", &objets); err != nil {
		return nil, fmt.Errorf("storage: listage sous le préfixe %q: %w", prefixe, err)
	}
	return objets, nil
}

// encodeChemin échappe un chemin d'objet segment par segment (pas
// url.PathEscape sur le chemin entier, qui encoderait aussi les "/" de
// séparation).
func encodeChemin(chemin string) string {
	segments := strings.Split(chemin, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

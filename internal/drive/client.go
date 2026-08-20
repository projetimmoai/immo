// Package drive est la couche d'accès à Google Drive : authentification via
// un compte de service Google, puis appels à l'API REST Drive v3 avec un
// client net/http fait main — même esprit que internal/repository pour
// Supabase. Seule l'obtention du jeton d'accès s'appuie sur golang.org/x/oauth2
// (signature JWT du compte de service), le reste est du REST simple.
package drive

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

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// driveScope est la portée d'accès demandée au compte de service. Il faut la
// portée complète "drive" (et non la portée restreinte "drive.file") car le
// dossier racine est partagé avec le compte de service par un compte humain,
// pas créé par le compte de service lui-même — "drive.file" ne donnerait
// accès qu'aux fichiers que le compte de service aurait créés.
const driveScope = "https://www.googleapis.com/auth/drive"

// CoproprieteSousDossiers sont les sous-dossiers créés sous chaque dossier de
// copropriété, un par catégorie de document. Exporté pour être réutilisé par
// internal/check (vérification de cohérence base/Drive), qui doit connaître
// la même liste sans la dupliquer.
var CoproprieteSousDossiers = []string{
	"contrats",
	"sinistres",
	"incidents",
	"emails",
	"assemblees_generales",
}

// Client est le client d'accès à l'API REST Drive v3.
type Client struct {
	baseURL      string
	rootFolderID string
	tokenSource  oauth2.TokenSource
	httpClient   *http.Client
}

// NewClientFromEnv construit un Client à partir des variables d'environnement
// GOOGLE_SERVICE_ACCOUNT_JSON (contenu JSON complet de la clé de compte de
// service Google) et DRIVE_ROOT_FOLDER_ID (id du dossier Drive racine, ex :
// "NE PAS MODIFIER - fichiers", déjà partagé avec ce compte de service).
func NewClientFromEnv(ctx context.Context) (*Client, error) {
	keyJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
	if keyJSON == "" {
		return nil, fmt.Errorf("drive: variable d'environnement GOOGLE_SERVICE_ACCOUNT_JSON manquante")
	}
	rootFolderID := os.Getenv("DRIVE_ROOT_FOLDER_ID")
	if rootFolderID == "" {
		return nil, fmt.Errorf("drive: variable d'environnement DRIVE_ROOT_FOLDER_ID manquante")
	}
	cfg, err := google.JWTConfigFromJSON([]byte(keyJSON), driveScope)
	if err != nil {
		return nil, fmt.Errorf("drive: lecture de la clé de compte de service: %w", err)
	}
	return NewClient(cfg.TokenSource(ctx), rootFolderID), nil
}

// NewClient construit un Client à partir d'une source de jetons OAuth2 et de
// l'id du dossier Drive racine, fournis explicitement (utile pour les tests).
func NewClient(tokenSource oauth2.TokenSource, rootFolderID string) *Client {
	return &Client{
		baseURL:      "https://www.googleapis.com/drive/v3",
		rootFolderID: rootFolderID,
		tokenSource:  tokenSource,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// RootFolderID retourne l'id du dossier racine (configuré, jamais généré).
func (c *Client) RootFolderID() string { return c.rootFolderID }

// APIError représente une erreur retournée par l'API REST Drive v3.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("drive: réponse %d de l'API: %s", e.StatusCode, e.Body)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	tok, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("drive: obtention du jeton d'accès: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("drive: encodage JSON de la requête %s %s: %w", method, path, err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("drive: construction de la requête %s %s: %w", method, path, err)
	}
	tok.SetAuthHeader(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("drive: appel %s %s: %w", method, path, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("drive: fermeture du corps de la réponse %s %s: %v", method, path, cerr)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("drive: lecture de la réponse %s %s: %w", method, path, err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("drive: décodage JSON de la réponse %s %s: %w", method, path, err)
		}
	}
	return nil
}

type fileMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type fileListResponse struct {
	Files         []fileMeta `json:"files"`
	NextPageToken string     `json:"nextPageToken"`
}

// Folder représente un dossier Drive (id + nom), exposé aux autres packages
// (ex : internal/check pour la vérification de cohérence).
type Folder struct {
	ID   string
	Name string
}

// ListChildFolders liste tous les sous-dossiers directs (non supprimés) d'un
// dossier Drive, en paginant si nécessaire.
func (c *Client) ListChildFolders(ctx context.Context, parentID string) ([]Folder, error) {
	q := fmt.Sprintf(
		"'%s' in parents and mimeType = 'application/vnd.google-apps.folder' and trashed = false",
		escapeQueryValue(parentID),
	)
	var folders []Folder
	pageToken := ""
	for {
		path := "/files?q=" + url.QueryEscape(q) + "&fields=" + url.QueryEscape("nextPageToken,files(id,name)") + "&pageSize=1000"
		if pageToken != "" {
			path += "&pageToken=" + url.QueryEscape(pageToken)
		}
		var resp fileListResponse
		if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, fmt.Errorf("drive: listage des sous-dossiers de %s: %w", parentID, err)
		}
		for _, f := range resp.Files {
			folders = append(folders, Folder(f))
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return folders, nil
}

// findFolder cherche un dossier par nom exact sous un parent donné. Retourne
// ("", false, nil) si aucun dossier ne correspond.
func (c *Client) findFolder(ctx context.Context, name, parentID string) (string, bool, error) {
	q := fmt.Sprintf(
		"name = '%s' and '%s' in parents and mimeType = 'application/vnd.google-apps.folder' and trashed = false",
		escapeQueryValue(name), escapeQueryValue(parentID),
	)
	path := "/files?q=" + url.QueryEscape(q) + "&fields=" + url.QueryEscape("files(id,name)")
	var resp fileListResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return "", false, fmt.Errorf("drive: recherche du dossier %q (parent=%s): %w", name, parentID, err)
	}
	switch len(resp.Files) {
	case 0:
		return "", false, nil
	case 1:
		return resp.Files[0].ID, true, nil
	default:
		return "", false, fmt.Errorf("drive: %d dossiers nommés %q trouvés sous le parent %s (1 attendu au plus) : lever l'ambiguïté manuellement", len(resp.Files), name, parentID)
	}
}

func (c *Client) createFolder(ctx context.Context, name, parentID string) (string, error) {
	payload := map[string]any{
		"name":     name,
		"mimeType": "application/vnd.google-apps.folder",
		"parents":  []string{parentID},
	}
	var created fileMeta
	if err := c.do(ctx, http.MethodPost, "/files?fields=id,name", payload, &created); err != nil {
		return "", fmt.Errorf("drive: création du dossier %q (parent=%s): %w", name, parentID, err)
	}
	return created.ID, nil
}

// EnsureFolder retourne l'id du dossier `name` sous `parentID`, en le créant
// s'il n'existe pas déjà. Idempotent : sûr à rejouer après un échec partiel
// (ex : la création d'un sous-dossier a échoué et on relance l'opération).
func (c *Client) EnsureFolder(ctx context.Context, name, parentID string) (string, error) {
	id, found, err := c.findFolder(ctx, name, parentID)
	if err != nil {
		return "", err
	}
	if found {
		return id, nil
	}
	return c.createFolder(ctx, name, parentID)
}

// CreateCoproprieteFolders crée (ou retrouve, si déjà existante) l'arborescence
// Drive d'une copropriété sous le dossier racine : un dossier
// "<reference> - <nom>" contenant un sous-dossier par catégorie de document
// (contrats, sinistres, incidents, emails, assemblees_generales). Retourne
// l'id du dossier de la copropriété.
func (c *Client) CreateCoproprieteFolders(ctx context.Context, reference, nom string) (string, error) {
	folderName := reference + " - " + nom
	rootID, err := c.EnsureFolder(ctx, folderName, c.rootFolderID)
	if err != nil {
		return "", fmt.Errorf("drive: dossier racine de la copropriété %q: %w", folderName, err)
	}
	for _, cat := range CoproprieteSousDossiers {
		if _, err := c.EnsureFolder(ctx, cat, rootID); err != nil {
			return rootID, fmt.Errorf("drive: sous-dossier %q de %q: %w", cat, folderName, err)
		}
	}
	return rootID, nil
}

// TrashFile déplace un fichier ou dossier dans la corbeille Drive
// (récupérable), plutôt que de le supprimer définitivement.
func (c *Client) TrashFile(ctx context.Context, id string) error {
	payload := map[string]any{"trashed": true}
	if err := c.do(ctx, http.MethodPatch, "/files/"+url.PathEscape(id), payload, nil); err != nil {
		return fmt.Errorf("drive: mise à la corbeille du fichier %s: %w", id, err)
	}
	return nil
}

// escapeQueryValue échappe une valeur pour l'utiliser dans une chaîne entre
// quotes simples d'une requête Drive "q=" (cf. doc Drive API : \\ et \').
func escapeQueryValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return v
}

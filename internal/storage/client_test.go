package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// newTestClient construit un Client à partir des variables d'environnement,
// et fait passer le test en "skip" (pas en échec) si elles sont absentes —
// ce sont des tests d'intégration qui ont besoin d'un vrai accès à Supabase
// Storage.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("SUPABASE_URL") == "" || os.Getenv("SUPABASE_SERVICE_ROLE_KEY") == "" {
		t.Skip("SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY non définies : test d'intégration ignoré")
	}
	c, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("NewClientFromEnv: %v", err)
	}
	return c
}

func TestUploadSignedURLListDelete(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	chemin := fmt.Sprintf("TEST/upload-%d.txt", time.Now().UnixNano())
	contenu := "contenu de test internal/storage"

	if err := c.Upload(ctx, chemin, strings.NewReader(contenu), "text/plain"); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	t.Cleanup(func() {
		if err := c.Delete(context.Background(), chemin); err != nil {
			t.Errorf("nettoyage Delete %s: %v", chemin, err)
		}
	})

	// Rejouer l'upload doit remplacer le fichier (upsert) plutôt qu'échouer.
	if err := c.Upload(ctx, chemin, strings.NewReader(contenu), "text/plain"); err != nil {
		t.Fatalf("Upload (rejeu, upsert): %v", err)
	}

	objets, err := c.List(ctx, "TEST/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	trouve := false
	for _, o := range objets {
		if strings.HasSuffix(chemin, o.Nom) {
			trouve = true
		}
	}
	if !trouve {
		t.Errorf("List(%q) = %+v, attendu le fichier uploadé", "TEST/", objets)
	}

	signedURL, err := c.SignedURL(ctx, chemin, time.Minute)
	if err != nil {
		t.Fatalf("SignedURL: %v", err)
	}
	if !strings.HasPrefix(signedURL, "http") {
		t.Fatalf("SignedURL = %q, attendu une URL complète", signedURL)
	}

	resp, err := http.Get(signedURL)
	if err != nil {
		t.Fatalf("téléchargement via SignedURL: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("téléchargement via SignedURL: statut %d", resp.StatusCode)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("lecture du contenu téléchargé: %v", err)
	}
	if string(got) != contenu {
		t.Errorf("contenu téléchargé = %q, attendu %q", got, contenu)
	}
}

package drive

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// newTestClient construit un Client à partir des variables d'environnement,
// et fait passer le test en "skip" (pas en échec) si elles sont absentes —
// ce sont des tests d'intégration qui ont besoin d'un vrai accès Drive.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON") == "" || os.Getenv("DRIVE_ROOT_FOLDER_ID") == "" {
		t.Skip("GOOGLE_SERVICE_ACCOUNT_JSON / DRIVE_ROOT_FOLDER_ID non définies : test d'intégration ignoré")
	}
	c, err := NewClientFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewClientFromEnv: %v", err)
	}
	return c
}

func TestCreateCoproprieteFoldersIdempotent(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	reference := fmt.Sprintf("TEST%d", time.Now().UnixNano())
	nom := "Copropriete de test"

	folderID, err := c.CreateCoproprieteFolders(ctx, reference, nom)
	if err != nil {
		t.Fatalf("CreateCoproprieteFolders: %v", err)
	}
	t.Cleanup(func() {
		if err := c.TrashFile(context.Background(), folderID); err != nil {
			t.Errorf("nettoyage TrashFile id=%s: %v", folderID, err)
		}
	})
	if folderID == "" {
		t.Fatal("CreateCoproprieteFolders: id de dossier vide")
	}

	// Rejouer l'opération doit retrouver le même dossier plutôt que d'en
	// créer un second (idempotence, cf. doc de CreateCoproprieteFolders).
	folderID2, err := c.CreateCoproprieteFolders(ctx, reference, nom)
	if err != nil {
		t.Fatalf("CreateCoproprieteFolders (rejeu): %v", err)
	}
	if folderID2 != folderID {
		t.Errorf("CreateCoproprieteFolders (rejeu) a créé un nouveau dossier: %s != %s", folderID2, folderID)
	}
}

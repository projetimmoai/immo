package gmailapi

import (
	"context"
	"os"
	"testing"
)

// newTestClient construit un Client à partir des variables d'environnement,
// et fait passer le test en "skip" (pas en échec) si elles sont absentes —
// ce sont des tests d'intégration qui ont besoin d'un vrai accès Gmail
// (cf. `go run ./cmd/gmail-auth` pour obtenir GOOGLE_OAUTH_TOKEN_JSON).
func newTestClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("GOOGLE_OAUTH_CLIENT_JSON") == "" || os.Getenv("GOOGLE_OAUTH_TOKEN_JSON") == "" {
		t.Skip("GOOGLE_OAUTH_CLIENT_JSON / GOOGLE_OAUTH_TOKEN_JSON non définies : test d'intégration ignoré")
	}
	c, err := NewClientFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewClientFromEnv: %v", err)
	}
	return c
}

func TestListMessageIDsRequeteSansResultat(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	// Requête qui ne devrait jamais correspondre à un message réel :
	// vérifie juste que l'appel réussit et gère une liste vide.
	ids, err := c.ListMessageIDs(ctx, "subject:\"chaine-de-test-qui-ne-devrait-jamais-exister-xyz\"")
	if err != nil {
		t.Fatalf("ListMessageIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ListMessageIDs: attendu aucun résultat, obtenu %v", ids)
	}
}

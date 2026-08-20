package claudeapi

import (
	"context"
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// newTestClient construit un Client à partir des variables d'environnement,
// et fait passer le test en "skip" (pas en échec) si elles sont absentes —
// ce sont des tests d'intégration qui ont besoin d'un vrai accès à l'API
// Claude.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY non définie : test d'intégration ignoré")
	}
	c, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("NewClientFromEnv: %v", err)
	}
	return c
}

// TestClientRequeteMinimale vérifie que la plomberie (clé API, modèle,
// SDK) fonctionne bout en bout avec une requête minimale.
func TestClientRequeteMinimale(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	resp, err := c.anthropic.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     Model,
		MaxTokens: 16,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("Réponds uniquement par le mot \"ok\".")),
		},
	})
	if err != nil {
		t.Fatalf("Messages.New: %v", err)
	}
	if len(resp.Content) == 0 {
		t.Fatal("Messages.New: réponse sans contenu")
	}
}

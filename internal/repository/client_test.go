package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// newTestClient construit un Client à partir des variables d'environnement,
// et fait passer le test en "skip" (pas en échec) si elles sont absentes —
// ce sont des tests d'intégration qui ont besoin d'un vrai accès Supabase.
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

func TestLookupReferenceID(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		lookup      func(context.Context, string) (int64, error)
		description string
	}{
		{"action", c.ActionID, domain.ActionSinistre},
		{"email_statut_traitement", c.EmailStatutTraitementID, domain.EmailStatutNouveau},
		{"incident_statut", c.IncidentStatutID, domain.IncidentStatutNouveau},
		{"niveau_urgence", c.NiveauUrgenceID, domain.NiveauUrgenceFaible},
		{"sinistre_statut", c.SinistreStatutID, domain.SinistreStatutDeclare},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := tt.lookup(ctx, tt.description)
			if err != nil {
				t.Fatalf("lookup %s(%q): %v", tt.name, tt.description, err)
			}
			if id <= 0 {
				t.Fatalf("lookup %s(%q): ID invalide: %d", tt.name, tt.description, id)
			}
		})
	}

	t.Run("description inconnue", func(t *testing.T) {
		if _, err := c.ActionID(ctx, "ne-devrait-pas-exister"); err == nil {
			t.Fatal("attendu une erreur pour une description inconnue, obtenu nil")
		}
	})
}

func TestSousActionIDInconnue(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	if _, err := c.SousActionID(ctx, 0, "ne-devrait-pas-exister"); err == nil {
		t.Fatal("attendu une erreur pour une sous_action inconnue, obtenu nil")
	}
}

func TestInsertEmailAndFind(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	statutID, err := c.EmailStatutTraitementID(ctx, domain.EmailStatutNouveau)
	if err != nil {
		t.Fatalf("EmailStatutTraitementID: %v", err)
	}

	messageID := fmt.Sprintf("test-integration-%d@immo.local", time.Now().UnixNano())
	objet := "Test d'intégration internal/repository"
	input := &domain.Email{
		MessageID:          &messageID,
		DateReception:      time.Now().UTC().Truncate(time.Second),
		ExpediteurEmail:    "test-integration@example.com",
		Objet:              &objet,
		StatutTraitementID: statutID,
	}

	created, err := c.InsertEmail(ctx, input)
	if err != nil {
		t.Fatalf("InsertEmail: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteEmail(context.Background(), created.ID); err != nil {
			t.Errorf("nettoyage DeleteEmail id=%d: %v", created.ID, err)
		}
	})

	if created.ID <= 0 {
		t.Fatalf("InsertEmail: ID invalide: %d", created.ID)
	}
	if created.ExpediteurEmail != input.ExpediteurEmail {
		t.Errorf("ExpediteurEmail = %q, attendu %q", created.ExpediteurEmail, input.ExpediteurEmail)
	}

	found, err := c.FindEmailByMessageID(ctx, messageID)
	if err != nil {
		t.Fatalf("FindEmailByMessageID: %v", err)
	}
	if found == nil {
		t.Fatal("FindEmailByMessageID: attendu un résultat, obtenu nil")
	}
	if found.ID != created.ID {
		t.Errorf("FindEmailByMessageID: ID = %d, attendu %d", found.ID, created.ID)
	}
}

func TestFindPersonneByEmailInconnue(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	p, err := c.FindPersonneByEmail(ctx, "adresse-qui-ne-devrait-pas-exister@example.invalid")
	if err != nil {
		t.Fatalf("FindPersonneByEmail: %v", err)
	}
	if p != nil {
		t.Fatalf("FindPersonneByEmail: attendu nil, obtenu %+v", p)
	}
}

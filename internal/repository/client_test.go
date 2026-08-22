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
		{"ticket_source_statut_traitement", c.TicketSourceStatutTraitementID, domain.TicketSourceStatutNouveau},
		{"ticket_source_type", c.TicketSourceTypeID, domain.TicketSourceTypeEmail},
		{"ticket_statut", c.TicketStatutID, domain.TicketStatutNouveau},
		{"niveau_urgence", c.NiveauUrgenceID, domain.NiveauUrgenceFaible},
		{"sinistre_resultat", c.SinistreResultatID, domain.SinistreResultatIndemnise},
		{"log_type", c.LogTypeID, domain.LogTypeCoproprieteNonIdentifiee},
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

	if _, err := c.SousActionID(ctx, 0, nil, "ne-devrait-pas-exister"); err == nil {
		t.Fatal("attendu une erreur pour une sous_action de premier niveau inconnue, obtenu nil")
	}

	parentID := int64(0)
	if _, err := c.SousActionID(ctx, 0, &parentID, "ne-devrait-pas-exister"); err == nil {
		t.Fatal("attendu une erreur pour une sous_action imbriquée inconnue, obtenu nil")
	}
}

func TestInsertEmailAndFind(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	statutID, err := c.TicketSourceStatutTraitementID(ctx, domain.TicketSourceStatutNouveau)
	if err != nil {
		t.Fatalf("TicketSourceStatutTraitementID: %v", err)
	}

	messageID := fmt.Sprintf("test-integration-%d@immo.local", time.Now().UnixNano())
	objet := "Test d'intégration internal/repository"
	sourceInput := &domain.TicketSource{
		DateReception:      time.Now().UTC().Truncate(time.Second),
		StatutTraitementID: statutID,
	}
	emailInput := &domain.Email{
		MessageID:       &messageID,
		ExpediteurEmail: "test-integration@example.com",
		Objet:           &objet,
	}

	createdSource, createdEmail, err := c.InsertEmail(ctx, sourceInput, emailInput)
	if err != nil {
		t.Fatalf("InsertEmail: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteTicketSource(context.Background(), createdSource.ID); err != nil {
			t.Errorf("nettoyage DeleteTicketSource id=%d: %v", createdSource.ID, err)
		}
	})

	if createdSource.ID <= 0 {
		t.Fatalf("InsertEmail: TicketSource.ID invalide: %d", createdSource.ID)
	}
	if createdEmail.ExpediteurEmail != emailInput.ExpediteurEmail {
		t.Errorf("ExpediteurEmail = %q, attendu %q", createdEmail.ExpediteurEmail, emailInput.ExpediteurEmail)
	}
	if createdEmail.TicketSourceID != createdSource.ID {
		t.Errorf("Email.TicketSourceID = %d, attendu %d (celui de la TicketSource créée)", createdEmail.TicketSourceID, createdSource.ID)
	}

	found, err := c.FindEmailByMessageID(ctx, messageID)
	if err != nil {
		t.Fatalf("FindEmailByMessageID: %v", err)
	}
	if found == nil {
		t.Fatal("FindEmailByMessageID: attendu un résultat, obtenu nil")
	}
	if found.TicketSourceID != createdSource.ID {
		t.Errorf("FindEmailByMessageID: TicketSourceID = %d, attendu %d", found.TicketSourceID, createdSource.ID)
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

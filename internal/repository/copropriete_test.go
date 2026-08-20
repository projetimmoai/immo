package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

func TestInsertCoproprieteAndDelete(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	nom := "Test d'intégration internal/repository - copropriete"
	created, err := c.InsertCopropriete(ctx, &domain.Copropriete{Nom: &nom})
	if err != nil {
		t.Fatalf("InsertCopropriete: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteCopropriete(context.Background(), created.ID); err != nil {
			t.Errorf("nettoyage DeleteCopropriete id=%d: %v", created.ID, err)
		}
	})

	if created.ID <= 0 {
		t.Fatalf("InsertCopropriete: ID invalide: %d", created.ID)
	}
	if created.Nom == nil || *created.Nom != nom {
		t.Errorf("Nom = %v, attendu %q", created.Nom, nom)
	}
	if !strings.HasPrefix(created.Reference, "COP") {
		t.Errorf("Reference = %q, attendu un préfixe \"COP\" (générée par la base)", created.Reference)
	}
}

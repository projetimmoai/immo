package repository

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

func TestInsertConseilSyndicalMandatAndDelete(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	statutID, err := c.ConseilSyndicalMandatStatutID(ctx, domain.ConseilSyndicalMandatStatutMembre)
	if err != nil {
		t.Fatalf("ConseilSyndicalMandatStatutID: %v", err)
	}

	created, err := c.InsertConseilSyndicalMandat(ctx, &domain.ConseilSyndicalMandat{StatutID: &statutID})
	if err != nil {
		t.Fatalf("InsertConseilSyndicalMandat: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteConseilSyndicalMandat(context.Background(), created.ID); err != nil {
			t.Errorf("nettoyage DeleteConseilSyndicalMandat id=%d: %v", created.ID, err)
		}
	})

	if created.ID <= 0 {
		t.Fatalf("InsertConseilSyndicalMandat: ID invalide: %d", created.ID)
	}
	if created.StatutID == nil || *created.StatutID != statutID {
		t.Errorf("StatutID = %v, attendu %d", created.StatutID, statutID)
	}
}

func TestInsertConseilSyndicalPresidentAndDelete(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	eluEnAG := true
	created, err := c.InsertConseilSyndicalPresident(ctx, &domain.ConseilSyndicalPresident{EluEnAG: &eluEnAG})
	if err != nil {
		t.Fatalf("InsertConseilSyndicalPresident: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteConseilSyndicalPresident(context.Background(), created.ID); err != nil {
			t.Errorf("nettoyage DeleteConseilSyndicalPresident id=%d: %v", created.ID, err)
		}
	})

	if created.ID <= 0 {
		t.Fatalf("InsertConseilSyndicalPresident: ID invalide: %d", created.ID)
	}
	if created.EluEnAG == nil || !*created.EluEnAG {
		t.Errorf("EluEnAG = %v, attendu true", created.EluEnAG)
	}
}

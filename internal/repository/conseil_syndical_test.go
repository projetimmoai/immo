package repository

import (
	"context"
	"testing"
	"time"

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

// TestInsertConseilSyndicalMandatAvecDates vérifie que date_debut/date_fin
// (colonnes SQL "date") sont correctement décodées une fois non nulles —
// cf. TestInsertCoproprieteAvecDates pour le détail du bug latent couvert.
func TestInsertConseilSyndicalMandatAvecDates(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	statutID, err := c.ConseilSyndicalMandatStatutID(ctx, domain.ConseilSyndicalMandatStatutMembre)
	if err != nil {
		t.Fatalf("ConseilSyndicalMandatStatutID: %v", err)
	}
	debut := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	fin := time.Date(2028, 6, 1, 0, 0, 0, 0, time.UTC)

	created, err := c.InsertConseilSyndicalMandat(ctx, &domain.ConseilSyndicalMandat{StatutID: &statutID, DateDebut: &debut, DateFin: &fin})
	if err != nil {
		t.Fatalf("InsertConseilSyndicalMandat: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteConseilSyndicalMandat(context.Background(), created.ID); err != nil {
			t.Errorf("nettoyage DeleteConseilSyndicalMandat id=%d: %v", created.ID, err)
		}
	})

	if created.DateDebut == nil || !created.DateDebut.Equal(debut) {
		t.Errorf("DateDebut = %v, attendu %v", created.DateDebut, debut)
	}
	if created.DateFin == nil || !created.DateFin.Equal(fin) {
		t.Errorf("DateFin = %v, attendu %v", created.DateFin, fin)
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

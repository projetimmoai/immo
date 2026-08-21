package repository

import (
	"context"
	"strings"
	"testing"
	"time"

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

// TestInsertCoproprieteAvecDates vérifie que les colonnes SQL "date" (par
// opposition à "timestamptz") sont correctement décodées une fois non
// nulles — un bug latent (*time.Time attend du RFC 3339, PostgREST envoie
// "YYYY-MM-DD" pour une colonne "date") ne se déclenchait jamais tant
// qu'aucune n'était renseignée.
func TestInsertCoproprieteAvecDates(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	nom := "Test d'intégration internal/repository - copropriete avec dates"
	exerciceDebut := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	exerciceFin := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	created, err := c.InsertCopropriete(ctx, &domain.Copropriete{
		Nom:           &nom,
		ExerciceDebut: &exerciceDebut,
		ExerciceFin:   &exerciceFin,
	})
	if err != nil {
		t.Fatalf("InsertCopropriete: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteCopropriete(context.Background(), created.ID); err != nil {
			t.Errorf("nettoyage DeleteCopropriete id=%d: %v", created.ID, err)
		}
	})

	if created.ExerciceDebut == nil || !created.ExerciceDebut.Equal(exerciceDebut) {
		t.Errorf("ExerciceDebut = %v, attendu %v", created.ExerciceDebut, exerciceDebut)
	}
	if created.ExerciceFin == nil || !created.ExerciceFin.Equal(exerciceFin) {
		t.Errorf("ExerciceFin = %v, attendu %v", created.ExerciceFin, exerciceFin)
	}
}

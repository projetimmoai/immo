package claudeapi

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

// TestDecideCoproprieteCasNonAmbigu vérifie, avec un vrai appel à l'API,
// que la sortie structurée forcée (tool_choice) fonctionne bout en bout :
// avec un e-mail qui nomme explicitement l'une des deux coproprietés
// candidates, Claude doit la retrouver avec une confiance élevée.
func TestDecideCoproprieteCasNonAmbigu(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	candidats := []domain.CandidatCopropriete{
		{CoproprieteID: 1, CoproprieteReference: "COP1", CoproprieteNom: strPtr("Residence Horizon"), Roles: []domain.Role{domain.RoleGestionnaire}},
		{CoproprieteID: 2, CoproprieteReference: "COP2", CoproprieteNom: strPtr("Les Herbiers"), Roles: []domain.Role{domain.RoleGestionnaire}},
	}

	decision, err := c.DecideCopropriete(
		ctx, candidats,
		"Panne d'ascenseur à la Residence Horizon",
		"Bonjour, l'ascenseur de la Residence Horizon est en panne depuis ce matin, merci d'intervenir rapidement.",
	)
	if err != nil {
		t.Fatalf("DecideCopropriete: %v", err)
	}
	if decision.CoproprieteID == nil {
		t.Fatalf("CoproprieteID = nil, attendu 1 (Residence Horizon nommée explicitement) — decision=%+v", decision)
	}
	if *decision.CoproprieteID != 1 {
		t.Errorf("CoproprieteID = %d, attendu 1 (Residence Horizon) — decision=%+v", *decision.CoproprieteID, decision)
	}
	if decision.Confiance < 0 || decision.Confiance > 1 {
		t.Errorf("Confiance = %v, attendu entre 0 et 1", decision.Confiance)
	}
	if decision.Raison == "" {
		t.Error("Raison vide, attendu une explication")
	}
}

func strPtr(s string) *string { return &s }

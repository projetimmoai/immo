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

// TestDecideCoproprieteRoleCoproprietaire vérifie, avec un vrai appel à
// l'API, que la valeur d'enum renommée "coproprietaire" (ex "client") est
// bien acceptée par le schéma de l'outil et correctement reconnue en
// retour : un e-mail qui parle sans ambiguïté de charges de copropriété
// doit être attribué au rôle coproprietaire.
func TestDecideCoproprieteRoleCoproprietaire(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	candidats := []domain.CandidatCopropriete{
		{CoproprieteID: 1, CoproprieteReference: "COP1", CoproprieteNom: strPtr("Residence Horizon"), Roles: []domain.Role{domain.RoleCoproprietaire}},
		{CoproprieteID: 2, CoproprieteReference: "COP2", CoproprieteNom: strPtr("Les Herbiers"), Roles: []domain.Role{domain.RoleGestionnaire}},
	}

	decision, err := c.DecideCopropriete(
		ctx, candidats,
		"Question sur mes charges de copropriété",
		"Bonjour, je suis propriétaire d'un lot à la Residence Horizon et je m'interroge sur le montant de mes charges de copropriété de ce trimestre. Merci de me répondre.",
	)
	if err != nil {
		t.Fatalf("DecideCopropriete: %v", err)
	}
	if decision.Role == nil || *decision.Role != domain.RoleCoproprietaire {
		t.Fatalf("Role = %v, attendu %q — decision=%+v", decision.Role, domain.RoleCoproprietaire, decision)
	}
	if decision.CoproprieteID == nil || *decision.CoproprieteID != 1 {
		t.Errorf("CoproprieteID = %v, attendu 1 — decision=%+v", decision.CoproprieteID, decision)
	}
}

// TestDecideCoproprieteRolePrestataire vérifie, avec un vrai appel à
// l'API, que la valeur d'enum renommée "prestataire" (ex "fournisseur")
// est bien acceptée par le schéma de l'outil et correctement reconnue en
// retour.
func TestDecideCoproprieteRolePrestataire(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	candidats := []domain.CandidatCopropriete{
		{CoproprieteID: 1, CoproprieteReference: "COP1", CoproprieteNom: strPtr("Residence Horizon"), Roles: []domain.Role{domain.RolePrestataire}},
		{CoproprieteID: 2, CoproprieteReference: "COP2", CoproprieteNom: strPtr("Les Herbiers"), Roles: []domain.Role{domain.RoleGestionnaire}},
	}

	decision, err := c.DecideCopropriete(
		ctx, candidats,
		"Question sur le contrat d'entretien - Residence Horizon",
		"Bonjour, nous intervenons en tant que prestataire d'entretien pour la Residence Horizon, et nous avons une question sur le renouvellement du contrat de maintenance. Merci de nous recontacter.",
	)
	if err != nil {
		t.Fatalf("DecideCopropriete: %v", err)
	}
	if decision.Role == nil || *decision.Role != domain.RolePrestataire {
		t.Fatalf("Role = %v, attendu %q — decision=%+v", decision.Role, domain.RolePrestataire, decision)
	}
	if decision.CoproprieteID == nil || *decision.CoproprieteID != 1 {
		t.Errorf("CoproprieteID = %v, attendu 1 — decision=%+v", decision.CoproprieteID, decision)
	}
}

func strPtr(s string) *string { return &s }

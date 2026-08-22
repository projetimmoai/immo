package repository

import (
	"context"
	"testing"
)

func TestListRolesParPersonneInconnue(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	roles, err := c.ListRolesParPersonne(ctx, 0)
	if err != nil {
		t.Fatalf("ListRolesParPersonne: %v", err)
	}
	if len(roles) != 0 {
		t.Fatalf("ListRolesParPersonne: attendu aucun rôle pour personne_id=0, obtenu %+v", roles)
	}
}

// TestListRolesParPersonneOccupantEtCoproprietaire vérifie, contre la vraie
// vue personne_role, qu'une personne à la fois occupante et propriétaire de
// son lot (PER1, données de seed) ressort bien avec les deux rôles scopés
// sur la même copropriete.
func TestListRolesParPersonneOccupantEtCoproprietaire(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	roles, err := c.ListRolesParPersonne(ctx, 1)
	if err != nil {
		t.Fatalf("ListRolesParPersonne: %v", err)
	}

	var occupant, coproprietaire bool
	for _, r := range roles {
		if r.CoproprieteID == nil {
			t.Errorf("ListRolesParPersonne: rôle %q sans copropriete_id, inattendu pour PER1", r.Role)
			continue
		}
		switch r.Role {
		case "occupant":
			occupant = true
		case "coproprietaire":
			coproprietaire = true
		}
		if r.CoproprieteReference == nil || *r.CoproprieteReference == "" {
			t.Errorf("ListRolesParPersonne: CoproprieteReference vide pour le rôle %q", r.Role)
		}
	}
	if !occupant || !coproprietaire {
		t.Fatalf("ListRolesParPersonne(1) = %+v, attendu occupant ET coproprietaire", roles)
	}
}

// TestListRolesParPersonneGestionnaireNonScope vérifie qu'un rôle
// intrinsèque (gestionnaire, PER4 dans les données de seed) ressort avec
// CoproprieteID nil — non scopé à une copropriete en particulier.
func TestListRolesParPersonneGestionnaireNonScope(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	roles, err := c.ListRolesParPersonne(ctx, 4)
	if err != nil {
		t.Fatalf("ListRolesParPersonne: %v", err)
	}

	var trouve bool
	for _, r := range roles {
		if r.Role != "gestionnaire" {
			continue
		}
		trouve = true
		if r.CoproprieteID != nil {
			t.Errorf("ListRolesParPersonne(4): gestionnaire avec CoproprieteID=%v, attendu nil (rôle non scopé)", *r.CoproprieteID)
		}
	}
	if !trouve {
		t.Fatalf("ListRolesParPersonne(4) = %+v, attendu le rôle gestionnaire", roles)
	}
}

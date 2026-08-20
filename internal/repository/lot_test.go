package repository

import (
	"context"
	"testing"
)

func TestListLotsParPersonneInconnue(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	lots, err := c.ListLotsParPersonne(ctx, 0)
	if err != nil {
		t.Fatalf("ListLotsParPersonne: %v", err)
	}
	if len(lots) != 0 {
		t.Fatalf("ListLotsParPersonne: attendu aucun lot pour personne_id=0, obtenu %+v", lots)
	}
}

func TestListContratsParFournisseurInconnu(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	contrats, err := c.ListContratsParFournisseur(ctx, 0)
	if err != nil {
		t.Fatalf("ListContratsParFournisseur: %v", err)
	}
	if len(contrats) != 0 {
		t.Fatalf("ListContratsParFournisseur: attendu aucun contrat pour entreprise_id=0, obtenu %+v", contrats)
	}
}

func TestFindPersonnePhysiqueByPersonneIDInconnue(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	pp, err := c.FindPersonnePhysiqueByPersonneID(ctx, 0)
	if err != nil {
		t.Fatalf("FindPersonnePhysiqueByPersonneID: %v", err)
	}
	if pp != nil {
		t.Fatalf("FindPersonnePhysiqueByPersonneID: attendu nil, obtenu %+v", pp)
	}
}

func TestFindPersonneMoraleByPersonneIDInconnue(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	pm, err := c.FindPersonneMoraleByPersonneID(ctx, 0)
	if err != nil {
		t.Fatalf("FindPersonneMoraleByPersonneID: %v", err)
	}
	if pm != nil {
		t.Fatalf("FindPersonneMoraleByPersonneID: attendu nil, obtenu %+v", pm)
	}
}

func TestListCoproprietesParGestionnaireInconnu(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	coproprietes, err := c.ListCoproprietesParGestionnaire(ctx, 0)
	if err != nil {
		t.Fatalf("ListCoproprietesParGestionnaire: %v", err)
	}
	if len(coproprietes) != 0 {
		t.Fatalf("ListCoproprietesParGestionnaire: attendu aucune copropriete pour personne_id=0, obtenu %+v", coproprietes)
	}
}

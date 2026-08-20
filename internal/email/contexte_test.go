package email

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

type fakeRepo struct {
	personnes         map[string]*domain.Personne // par email
	personnesPhysique map[int64]*domain.PersonnePhysique
	personnesMorale   map[int64]*domain.PersonneMorale
	lots              map[int64][]repository.LotAssocie
	contrats          map[int64][]*domain.Contrat
}

func (f *fakeRepo) FindPersonneByEmail(_ context.Context, email string) (*domain.Personne, error) {
	return f.personnes[email], nil
}

func (f *fakeRepo) FindPersonnePhysiqueByPersonneID(_ context.Context, personneID int64) (*domain.PersonnePhysique, error) {
	return f.personnesPhysique[personneID], nil
}

func (f *fakeRepo) FindPersonneMoraleByPersonneID(_ context.Context, personneID int64) (*domain.PersonneMorale, error) {
	return f.personnesMorale[personneID], nil
}

func (f *fakeRepo) ListLotsParPersonne(_ context.Context, personneID int64) ([]repository.LotAssocie, error) {
	return f.lots[personneID], nil
}

func (f *fakeRepo) ListContratsParFournisseur(_ context.Context, entrepriseID int64) ([]*domain.Contrat, error) {
	return f.contrats[entrepriseID], nil
}

func TestEnrichirExpediteurInconnu(t *testing.T) {
	repo := &fakeRepo{personnes: map[string]*domain.Personne{}}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "inconnu@example.com")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if ctx.Connu {
		t.Fatalf("attendu Connu=false, obtenu %+v", ctx)
	}
}

func TestEnrichirExpediteurPersonnePhysiqueAvecLots(t *testing.T) {
	vrai := true
	nom := "Dupont"
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"jean@example.com": {ID: 1, EstPhysique: &vrai, Reference: "PER1"},
		},
		personnesPhysique: map[int64]*domain.PersonnePhysique{
			1: {ID: 10, Nom: &nom},
		},
		lots: map[int64][]repository.LotAssocie{
			1: {{LotID: 100, LotReference: "LOT1", CoproprieteReference: "COP1", EstProprietaire: &vrai}},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "jean@example.com")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if !ctx.Connu {
		t.Fatal("attendu Connu=true")
	}
	if ctx.PersonnePhysique == nil || ctx.PersonnePhysique.ID != 10 {
		t.Errorf("PersonnePhysique = %+v", ctx.PersonnePhysique)
	}
	if len(ctx.Lots) != 1 || ctx.Lots[0].LotReference != "LOT1" {
		t.Errorf("Lots = %+v", ctx.Lots)
	}
}

func TestEnrichirExpediteurPersonneMoraleFournisseurAvecContrats(t *testing.T) {
	faux := false
	vrai := true
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"contact@fournisseur.example": {ID: 2, EstPhysique: &faux, Reference: "PER2"},
		},
		personnesMorale: map[int64]*domain.PersonneMorale{
			2: {ID: 20, EstFournisseur: &vrai},
		},
		contrats: map[int64][]*domain.Contrat{
			2: {{ID: 200, NumeroContrat: strPtr("CTR-1")}},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "contact@fournisseur.example")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if ctx.PersonneMorale == nil || ctx.PersonneMorale.ID != 20 {
		t.Errorf("PersonneMorale = %+v", ctx.PersonneMorale)
	}
	if len(ctx.Contrats) != 1 || ctx.Contrats[0].ID != 200 {
		t.Errorf("Contrats = %+v", ctx.Contrats)
	}
}

func TestEnrichirExpediteurPersonneMoraleNonFournisseurSansContrats(t *testing.T) {
	faux := false
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"syndic@copro.example": {ID: 3, EstPhysique: &faux, Reference: "PER3"},
		},
		personnesMorale: map[int64]*domain.PersonneMorale{
			3: {ID: 30, EstFournisseur: &faux},
		},
		contrats: map[int64][]*domain.Contrat{
			3: {{ID: 999}}, // ne devrait jamais être consulté (EstFournisseur=false)
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "syndic@copro.example")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if ctx.Contrats != nil {
		t.Errorf("Contrats = %+v, attendu nil (pas fournisseur, ListContratsParFournisseur ne doit pas être appelé)", ctx.Contrats)
	}
}

func strPtr(s string) *string { return &s }

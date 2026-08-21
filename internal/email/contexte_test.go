package email

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

type fakeRepo struct {
	personnes               map[string]*domain.Personne // par email
	personnesPhysique       map[int64]*domain.PersonnePhysique
	personnesMorale         map[int64]*domain.PersonneMorale
	lots                    map[int64][]repository.LotAssocie
	contrats                map[int64][]repository.ContratAssocie
	coproprietesGerees      map[int64][]repository.CoproprieteAssociee
	coproprietesConseilSynd map[int64][]repository.CoproprieteAssociee
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

func (f *fakeRepo) ListContratsParFournisseur(_ context.Context, entrepriseID int64) ([]repository.ContratAssocie, error) {
	return f.contrats[entrepriseID], nil
}

func (f *fakeRepo) ListCoproprietesParGestionnaire(_ context.Context, personneID int64) ([]repository.CoproprieteAssociee, error) {
	return f.coproprietesGerees[personneID], nil
}

func (f *fakeRepo) ListCoproprietesConseilSyndicalParPersonne(_ context.Context, personneID int64) ([]repository.CoproprieteAssociee, error) {
	return f.coproprietesConseilSynd[personneID], nil
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
	if len(ctx.Roles) != 0 {
		t.Errorf("Roles = %+v, attendu vide", ctx.Roles)
	}
}

func TestEnrichirExpediteurOccupantEtCoproprietaire(t *testing.T) {
	vrai := true
	nom := "Dupont"
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"jean@example.com": {ID: 1, EstPhysique: &vrai, EstOccupant: &vrai, EstCoproprietaire: &vrai, Reference: "PER1"},
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
	if !ctx.ARole(domain.RoleOccupant) || !ctx.ARole(domain.RoleCoproprietaire) {
		t.Errorf("Roles = %+v, attendu occupant ET coproprietaire", ctx.Roles)
	}
	if ctx.ARole(domain.RoleFournisseur) || ctx.ARole(domain.RoleGestionnaire) {
		t.Errorf("Roles = %+v, attendu ni fournisseur ni gestionnaire", ctx.Roles)
	}
	if ctx.PersonnePhysique == nil || ctx.PersonnePhysique.ID != 10 {
		t.Errorf("PersonnePhysique = %+v", ctx.PersonnePhysique)
	}
	if len(ctx.Lots) != 1 || ctx.Lots[0].LotReference != "LOT1" {
		t.Errorf("Lots = %+v", ctx.Lots)
	}
	if len(ctx.Coproprietes) != 1 || ctx.Coproprietes[0].CoproprieteReference != "COP1" {
		t.Errorf("Coproprietes = %+v", ctx.Coproprietes)
	}
	// Ni fournisseur ni gestionnaire : pas d'appel réseau inutile attendu.
	if ctx.Contrats != nil {
		t.Errorf("Contrats = %+v, attendu nil (pas fournisseur)", ctx.Contrats)
	}
	if ctx.CoproprietesGestion != nil {
		t.Errorf("CoproprietesGestion = %+v, attendu nil (pas gestionnaire)", ctx.CoproprietesGestion)
	}
}

func TestEnrichirExpediteurCoproprietesDedupliquees(t *testing.T) {
	vrai := true
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"proprio@example.com": {ID: 6, EstPhysique: &vrai, EstCoproprietaire: &vrai, Reference: "PER6"},
		},
		lots: map[int64][]repository.LotAssocie{
			6: {
				{LotID: 100, LotReference: "LOT1", CoproprieteID: 1, CoproprieteReference: "COP1", EstProprietaire: &vrai},
				{LotID: 101, LotReference: "LOT2", CoproprieteID: 1, CoproprieteReference: "COP1", EstProprietaire: &vrai},
				{LotID: 200, LotReference: "LOT3", CoproprieteID: 2, CoproprieteReference: "COP2", EstProprietaire: &vrai},
			},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "proprio@example.com")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if len(ctx.Lots) != 3 {
		t.Fatalf("Lots = %+v, attendu 3 (pas de déduplication sur les lots)", ctx.Lots)
	}
	if len(ctx.Coproprietes) != 2 {
		t.Fatalf("Coproprietes = %+v, attendu 2 (dédupliquées : COP1 x2 lots -> 1 entrée)", ctx.Coproprietes)
	}
}

func TestEnrichirExpediteurFournisseurAvecContrats(t *testing.T) {
	faux := false
	vrai := true
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"contact@fournisseur.example": {ID: 2, EstPhysique: &faux, Reference: "PER2"},
		},
		personnesMorale: map[int64]*domain.PersonneMorale{
			2: {ID: 20, EstFournisseur: &vrai},
		},
		contrats: map[int64][]repository.ContratAssocie{
			2: {{ContratID: 200, NumeroContrat: strPtr("CTR-1"), CoproprieteID: 1, CoproprieteReference: "COP1"}},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "contact@fournisseur.example")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if !ctx.ARole(domain.RoleFournisseur) {
		t.Errorf("Roles = %+v, attendu fournisseur", ctx.Roles)
	}
	if ctx.PersonneMorale == nil || ctx.PersonneMorale.ID != 20 {
		t.Errorf("PersonneMorale = %+v", ctx.PersonneMorale)
	}
	if len(ctx.Contrats) != 1 || ctx.Contrats[0].ContratID != 200 {
		t.Errorf("Contrats = %+v", ctx.Contrats)
	}
	if ctx.Lots != nil {
		t.Errorf("Lots = %+v, attendu nil (ni occupant ni coproprietaire)", ctx.Lots)
	}
}

func TestEnrichirExpediteurGestionnaireAvecCoproprietes(t *testing.T) {
	vrai := true
	nom := "Martin"
	nomCop := "Residence Horizon"
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"martin@cabinet.example": {ID: 3, EstPhysique: &vrai, EstGestionnaire: &vrai, Reference: "PER3"},
		},
		personnesPhysique: map[int64]*domain.PersonnePhysique{
			3: {ID: 30, Nom: &nom},
		},
		coproprietesGerees: map[int64][]repository.CoproprieteAssociee{
			3: {{CoproprieteID: 1, CoproprieteNom: &nomCop, CoproprieteReference: "COP1"}},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "martin@cabinet.example")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if !ctx.ARole(domain.RoleGestionnaire) {
		t.Errorf("Roles = %+v, attendu gestionnaire", ctx.Roles)
	}
	if len(ctx.CoproprietesGestion) != 1 || ctx.CoproprietesGestion[0].CoproprieteReference != "COP1" {
		t.Errorf("CoproprietesGestion = %+v", ctx.CoproprietesGestion)
	}
}

func TestEnrichirExpediteurPlusieursRoles(t *testing.T) {
	vrai := true
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"multi@example.com": {
				ID: 4, EstPhysique: &vrai,
				EstOccupant: &vrai, EstCoproprietaire: &vrai, EstGestionnaire: &vrai,
			},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "multi@example.com")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	for _, r := range []domain.Role{domain.RoleOccupant, domain.RoleCoproprietaire, domain.RoleGestionnaire} {
		if !ctx.ARole(r) {
			t.Errorf("Roles = %+v, attendu %q parmi les rôles cumulés", ctx.Roles, r)
		}
	}
	if ctx.ARole(domain.RoleFournisseur) {
		t.Errorf("Roles = %+v, attendu pas fournisseur (personne physique)", ctx.Roles)
	}
}

func TestEnrichirExpediteurConnuSansRole(t *testing.T) {
	faux := false
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"syndic@copro.example": {ID: 5, EstPhysique: &faux, Reference: "PER5"},
		},
		personnesMorale: map[int64]*domain.PersonneMorale{
			5: {ID: 50, EstFournisseur: &faux},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "syndic@copro.example")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if len(ctx.Roles) != 0 {
		t.Errorf("Roles = %+v, attendu vide", ctx.Roles)
	}
	if ctx.Contrats != nil || ctx.Lots != nil || ctx.CoproprietesGestion != nil {
		t.Errorf("attendu aucun appel complémentaire sans rôle : Contrats=%+v Lots=%+v CoproprietesGestion=%+v", ctx.Contrats, ctx.Lots, ctx.CoproprietesGestion)
	}
}

func TestEnrichirExpediteurMembreConseilSyndical(t *testing.T) {
	vrai := true
	nomCop := "Le Clos des Vignes"
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"presidente@example.com": {ID: 1, EstPhysique: &vrai, EstCoproprietaire: &vrai, Reference: "PER1"},
		},
		coproprietesConseilSynd: map[int64][]repository.CoproprieteAssociee{
			1: {{CoproprieteID: 1, CoproprieteNom: &nomCop, CoproprieteReference: "COP1"}},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "presidente@example.com")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if !ctx.ARole(domain.RoleCoproprietaire) || !ctx.ARole(domain.RoleConseilSyndical) {
		t.Errorf("Roles = %+v, attendu coproprietaire ET conseil_syndical", ctx.Roles)
	}
	if len(ctx.CoproprietesConseilSyndical) != 1 || ctx.CoproprietesConseilSyndical[0].CoproprieteReference != "COP1" {
		t.Errorf("CoproprietesConseilSyndical = %+v", ctx.CoproprietesConseilSyndical)
	}
}

func TestEnrichirExpediteurCoproprietaireSansMandatConseilSyndical(t *testing.T) {
	vrai := true
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"proprio@example.com": {ID: 1, EstPhysique: &vrai, EstCoproprietaire: &vrai, Reference: "PER1"},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "proprio@example.com")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if ctx.ARole(domain.RoleConseilSyndical) {
		t.Errorf("Roles = %+v, attendu pas conseil_syndical (aucun mandat actif)", ctx.Roles)
	}
	if ctx.CoproprietesConseilSyndical != nil {
		t.Errorf("CoproprietesConseilSyndical = %+v, attendu nil", ctx.CoproprietesConseilSyndical)
	}
}

func TestEnrichirExpediteurNonCoproprietairePasDeRequeteConseilSyndical(t *testing.T) {
	vrai := true
	repo := &fakeRepo{
		personnes: map[string]*domain.Personne{
			"locataire@example.com": {ID: 1, EstPhysique: &vrai, EstOccupant: &vrai, Reference: "PER1"},
		},
		// Si EnrichirExpediteur interrogeait quand même le conseil syndical
		// pour un simple occupant (non coproprietaire), ce mandat serait vu à tort.
		coproprietesConseilSynd: map[int64][]repository.CoproprieteAssociee{
			1: {{CoproprieteID: 1, CoproprieteReference: "COP1"}},
		},
	}

	ctx, err := EnrichirExpediteur(context.Background(), repo, "locataire@example.com")
	if err != nil {
		t.Fatalf("EnrichirExpediteur: %v", err)
	}
	if ctx.ARole(domain.RoleConseilSyndical) {
		t.Errorf("Roles = %+v, attendu pas conseil_syndical (occupant non coproprietaire)", ctx.Roles)
	}
}

func strPtr(s string) *string { return &s }

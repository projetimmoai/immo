package email

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

type fakeActionDecideur struct {
	decision      claudeapi.DecisionAction
	err           error
	actionsRecues []domain.Action
}

func (f *fakeActionDecideur) DecideAction(_ context.Context, actions []domain.Action, _ domain.ContexteRoutage, _, _ string) (claudeapi.DecisionAction, error) {
	f.actionsRecues = actions
	return f.decision, f.err
}

func actionsTest() []*domain.Action {
	return []*domain.Action{
		{ID: 1, Description: domain.ActionSinistre},
		{ID: 2, Description: domain.ActionIncident},
		{ID: 3, Description: domain.ActionAssembleeGenerale},
		{ID: 4, Description: domain.ActionAutre},
		{ID: 5, Description: domain.ActionComptabilite},
		{ID: 6, Description: domain.ActionMutation},
		{ID: 7, Description: domain.ActionContentieux},
		{ID: 8, Description: domain.ActionTravaux},
		{ID: 9, Description: domain.ActionControleGestion},
		{ID: 10, Description: domain.ActionDemandeAdministrative},
	}
}

func TestRouterEmailActionConnueAppelleLeBonGestionnaire(t *testing.T) {
	decideur := &fakeActionDecideur{decision: claudeapi.DecisionAction{Action: domain.ActionIncident, Confiance: 0.9, Raison: "fuite d'eau mentionnée"}}
	ctxRoutage := domain.ContexteRoutage{CoproprieteReference: "COP1"}

	res, err := RouterEmail(context.Background(), decideur, actionsTest(), ctxRoutage, "Fuite d'eau", "Il y a une fuite dans le lot")
	if err != nil {
		t.Fatalf("RouterEmail: %v", err)
	}
	if res.Action != domain.ActionIncident {
		t.Errorf("Action = %q, attendu %q", res.Action, domain.ActionIncident)
	}
	if res.Confiance != 0.9 {
		t.Errorf("Confiance = %v, attendu 0.9", res.Confiance)
	}
	if len(decideur.actionsRecues) != 10 {
		t.Errorf("actions reçues par Claude = %+v, attendu 10", decideur.actionsRecues)
	}
}

func TestRouterEmailToutesLesActionsConnuesOntUnGestionnaire(t *testing.T) {
	// Vérifie que chaque action de la table (via ses constantes connues)
	// a bien une fonction de traitement enregistrée, pour ne pas découvrir
	// l'oubli seulement à l'exécution.
	for _, a := range actionsTest() {
		if _, ok := gestionnairesAction[a.Description]; !ok {
			t.Errorf("aucune fonction de traitement enregistrée pour l'action %q", a.Description)
		}
	}
}

func TestRouterEmailActionInconnueRejetee(t *testing.T) {
	decideur := &fakeActionDecideur{decision: claudeapi.DecisionAction{Action: "action-inventee", Confiance: 0.9, Raison: "hallucination"}}
	ctxRoutage := domain.ContexteRoutage{CoproprieteReference: "COP1"}

	res, err := RouterEmail(context.Background(), decideur, actionsTest(), ctxRoutage, "Objet", "Corps")
	if err != nil {
		t.Fatalf("RouterEmail: %v", err)
	}
	if res.Action != "" {
		t.Errorf("Action = %q, attendu vide (réponse rejetée)", res.Action)
	}
}

func TestRouterEmailAucuneActionDisponible(t *testing.T) {
	ctxRoutage := domain.ContexteRoutage{CoproprieteReference: "COP1"}
	_, err := RouterEmail(context.Background(), &fakeActionDecideur{}, nil, ctxRoutage, "Objet", "Corps")
	if err == nil {
		t.Fatal("attendu une erreur (aucune action disponible), obtenu nil")
	}
}

func TestNouveauContexteRoutage(t *testing.T) {
	personne := &domain.Personne{ID: 1, Reference: "PER1"}
	role := domain.RoleClient
	numeroContrat := "CTR-1"

	ec := &Contexte{
		Personne: personne,
		Lots: []repository.LotAssocie{
			{LotID: 100, LotReference: "LOT1", CoproprieteID: 1},
			{LotID: 200, LotReference: "LOT2", CoproprieteID: 2}, // autre copropriete, ne doit pas apparaître
		},
		Contrats: []repository.ContratAssocie{
			{ContratID: 300, NumeroContrat: &numeroContrat, CoproprieteID: 1},
		},
	}
	res := ResolutionCopropriete{CoproprieteID: int64Ptr(1), CoproprieteReference: "COP1", Role: &role}

	cr := NouveauContexteRoutage(ec, res)
	if cr == nil {
		t.Fatal("NouveauContexteRoutage = nil, attendu non nil (CoproprieteID renseigné)")
	}
	if cr.Personne != personne {
		t.Errorf("Personne = %+v, attendu %+v", cr.Personne, personne)
	}
	if cr.Role == nil || *cr.Role != domain.RoleClient {
		t.Errorf("Role = %v, attendu client", cr.Role)
	}
	if cr.CoproprieteID != 1 || cr.CoproprieteReference != "COP1" {
		t.Errorf("Copropriete = %d/%s, attendu 1/COP1", cr.CoproprieteID, cr.CoproprieteReference)
	}
	if len(cr.LotsReferences) != 1 || cr.LotsReferences[0] != "LOT1" {
		t.Errorf("LotsReferences = %+v, attendu [LOT1] (LOT2 est dans une autre copropriete)", cr.LotsReferences)
	}
	if len(cr.ContratsNumeros) != 1 || cr.ContratsNumeros[0] != "CTR-1" {
		t.Errorf("ContratsNumeros = %+v, attendu [CTR-1]", cr.ContratsNumeros)
	}
}

func TestNouveauContexteRoutageCoproprieteNonIdentifiee(t *testing.T) {
	if cr := NouveauContexteRoutage(&Contexte{}, ResolutionCopropriete{}); cr != nil {
		t.Errorf("NouveauContexteRoutage = %+v, attendu nil (CoproprieteID absent)", cr)
	}
}

func int64Ptr(v int64) *int64 { return &v }

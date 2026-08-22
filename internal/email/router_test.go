package email

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

type fakeActionDecideur struct {
	decisions     []claudeapi.DecisionAction
	err           error
	actionsRecues []domain.Action
}

func (f *fakeActionDecideur) DecideAction(_ context.Context, actions []domain.Action, _ domain.ContexteRoutage, _, _ string) ([]claudeapi.DecisionAction, error) {
	f.actionsRecues = actions
	return f.decisions, f.err
}

func actionsOccupantTest() []domain.Action {
	return []domain.Action{
		{ID: 1, Description: domain.ActionSinistre},
		{ID: 2, Description: domain.ActionIncident},
		{ID: 3, Description: domain.ActionTravaux},
		{ID: 4, Description: domain.ActionDemandeAdministrative},
		{ID: 5, Description: domain.ActionMutation},
	}
}

func TestRouterVersActionsActionConnueAppelleLeBonGestionnaire(t *testing.T) {
	decideur := &fakeActionDecideur{decisions: []claudeapi.DecisionAction{
		{Action: domain.ActionIncident, Confiance: 0.9, Raison: "fuite d'eau mentionnée"},
	}}
	ctxRoutage := domain.ContexteRoutage{CoproprieteReference: "COP1"}

	resultats, err := routerVersActions(context.Background(), decideur, actionsOccupantTest(), ctxRoutage, "Fuite d'eau", "Il y a une fuite dans le lot")
	if err != nil {
		t.Fatalf("routerVersActions: %v", err)
	}
	if len(resultats) != 1 {
		t.Fatalf("resultats = %+v, attendu 1", resultats)
	}
	if resultats[0].Action != domain.ActionIncident {
		t.Errorf("Action = %q, attendu %q", resultats[0].Action, domain.ActionIncident)
	}
	if resultats[0].Confiance != 0.9 {
		t.Errorf("Confiance = %v, attendu 0.9", resultats[0].Confiance)
	}
	if len(decideur.actionsRecues) != 5 {
		t.Errorf("actions reçues par Claude = %+v, attendu 5", decideur.actionsRecues)
	}
}

func TestRouterVersActionsActionInconnueRejetee(t *testing.T) {
	decideur := &fakeActionDecideur{decisions: []claudeapi.DecisionAction{
		{Action: "action-inventee", Confiance: 0.9, Raison: "hallucination"},
	}}
	ctxRoutage := domain.ContexteRoutage{CoproprieteReference: "COP1"}

	resultats, err := routerVersActions(context.Background(), decideur, actionsOccupantTest(), ctxRoutage, "Objet", "Corps")
	if err != nil {
		t.Fatalf("routerVersActions: %v", err)
	}
	if len(resultats) != 1 || resultats[0].Action != "" {
		t.Errorf("resultats = %+v, attendu 1 résultat avec Action vide (réponse rejetée)", resultats)
	}
}

// TestRouterVersActionsPlusieursDemandesDispatchentChacune vérifie qu'un
// e-mail dont Claude identifie 2 demandes distinctes produit bien 2
// ResolutionAction, chacune dispatchée vers sa propre fonction de
// traitement — pas une seule action retenue arbitrairement.
func TestRouterVersActionsPlusieursDemandesDispatchentChacune(t *testing.T) {
	decideur := &fakeActionDecideur{decisions: []claudeapi.DecisionAction{
		{Action: domain.ActionIncident, Confiance: 0.9, Raison: "digicode en panne"},
		{Action: domain.ActionDemandeAdministrative, Confiance: 0.85, Raison: "demande de règlement de copropriété"},
	}}
	ctxRoutage := domain.ContexteRoutage{CoproprieteReference: "COP1"}

	resultats, err := routerVersActions(context.Background(), decideur, actionsOccupantTest(), ctxRoutage, "Objet", "Corps")
	if err != nil {
		t.Fatalf("routerVersActions: %v", err)
	}
	if len(resultats) != 2 {
		t.Fatalf("resultats = %+v, attendu 2", resultats)
	}
	if resultats[0].Action != domain.ActionIncident || resultats[1].Action != domain.ActionDemandeAdministrative {
		t.Errorf("resultats = %+v, attendu [incident, demande_administrative] dans cet ordre", resultats)
	}
}

func TestRouterVersActionsAucuneActionDisponible(t *testing.T) {
	ctxRoutage := domain.ContexteRoutage{CoproprieteReference: "COP1"}
	_, err := routerVersActions(context.Background(), &fakeActionDecideur{}, nil, ctxRoutage, "Objet", "Corps")
	if err == nil {
		t.Fatal("attendu une erreur (aucune action disponible), obtenu nil")
	}
}

func TestFiltrerActions(t *testing.T) {
	actions := []*domain.Action{
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

	filtrees := filtrerActions(actions, actionsOccupant)
	if len(filtrees) != len(actionsOccupant) {
		t.Fatalf("filtrerActions = %+v, attendu %d actions (celles de actionsOccupant)", filtrees, len(actionsOccupant))
	}
	for _, description := range actionsOccupant {
		trouvee := false
		for _, a := range filtrees {
			if a.Description == description {
				trouvee = true
				break
			}
		}
		if !trouvee {
			t.Errorf("filtrerActions = %+v, attendu %q parmi le résultat", filtrees, description)
		}
	}
}

func TestNouveauContexteRoutage(t *testing.T) {
	personne := &domain.Personne{ID: 1, Reference: "PER1"}
	role := domain.RoleCoproprietaire
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
	if cr.Role == nil || *cr.Role != domain.RoleCoproprietaire {
		t.Errorf("Role = %v, attendu coproprietaire", cr.Role)
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

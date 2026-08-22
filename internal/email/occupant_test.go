package email

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
)

// actionsTableTest simule le contenu complet de la table action (au-delà
// des seules actions occupant), pour vérifier que RouterOccupant filtre
// bien avant de proposer les actions à Claude.
func actionsTableTest() []*domain.Action {
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

func TestRouterOccupantNePropseQueLesActionsOccupant(t *testing.T) {
	decideur := &fakeActionDecideur{decisions: []claudeapi.DecisionAction{
		{Action: domain.ActionIncident, Confiance: 0.9, Raison: "ascenseur en panne"},
	}}
	ctxRoutage := domain.ContexteRoutage{CoproprieteReference: "COP1"}

	resultats, err := RouterOccupant(context.Background(), decideur, actionsTableTest(), ctxRoutage, "Ascenseur en panne", "Bonjour, l'ascenseur ne fonctionne plus.")
	if err != nil {
		t.Fatalf("RouterOccupant: %v", err)
	}
	if len(resultats) != 1 {
		t.Fatalf("resultats = %+v, attendu 1", resultats)
	}
	if resultats[0].Action != domain.ActionIncident {
		t.Errorf("Action = %q, attendu %q", resultats[0].Action, domain.ActionIncident)
	}
	if len(decideur.actionsRecues) != len(actionsOccupant) {
		t.Fatalf("actions reçues par Claude = %+v, attendu %d (actionsOccupant, pas la table entière)", decideur.actionsRecues, len(actionsOccupant))
	}
	for _, a := range decideur.actionsRecues {
		if a.Description == domain.ActionAssembleeGenerale || a.Description == domain.ActionComptabilite || a.Description == domain.ActionContentieux || a.Description == domain.ActionControleGestion || a.Description == domain.ActionAutre {
			t.Errorf("actions reçues par Claude = %+v, ne devrait pas contenir %q (pas une action occupant)", decideur.actionsRecues, a.Description)
		}
	}
}

func TestRouterOccupantToutesLesActionsOccupantOntUnGestionnaire(t *testing.T) {
	// Vérifie que chaque action proposée à un occupant a bien une fonction
	// de traitement enregistrée, pour ne pas découvrir l'oubli seulement à
	// l'exécution.
	for _, description := range actionsOccupant {
		if _, ok := gestionnairesAction[description]; !ok {
			t.Errorf("aucune fonction de traitement enregistrée pour l'action occupant %q", description)
		}
	}
}

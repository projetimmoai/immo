package claudeapi

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

// TestDecideActionCasNonAmbigu vérifie, avec un vrai appel à l'API, que la
// sortie structurée forcée fonctionne bout en bout pour DecideAction : un
// e-mail qui décrit sans ambiguïté un incident technique doit être classé
// "incident" avec une confiance élevée.
func TestDecideActionCasNonAmbigu(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	actions := []domain.Action{
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
	role := domain.RoleOccupant
	ctxRoutage := domain.ContexteRoutage{
		Role:                 &role,
		CoproprieteReference: "COP1",
		LotsReferences:       []string{"LOT1"},
	}

	decision, err := c.DecideAction(
		ctx, actions, ctxRoutage,
		"Panne d'ascenseur",
		"Bonjour, l'ascenseur de l'immeuble est en panne depuis ce matin, merci d'intervenir rapidement.",
	)
	if err != nil {
		t.Fatalf("DecideAction: %v", err)
	}
	if decision.Action != domain.ActionIncident {
		t.Errorf("Action = %q, attendu %q — decision=%+v", decision.Action, domain.ActionIncident, decision)
	}
	if decision.Confiance < 0 || decision.Confiance > 1 {
		t.Errorf("Confiance = %v, attendu entre 0 et 1", decision.Confiance)
	}
	if decision.Raison == "" {
		t.Error("Raison vide, attendu une explication")
	}
}

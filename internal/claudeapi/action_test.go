package claudeapi

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

// TestDecideActionCasNonAmbigu vérifie, avec un vrai appel à l'API, que la
// sortie structurée forcée fonctionne bout en bout pour DecideAction : un
// e-mail qui décrit sans ambiguïté un incident technique doit être classé
// "incident" avec une confiance élevée, parmi le sous-ensemble d'actions
// pertinentes pour un occupant (pas la table entière).
func TestDecideActionCasNonAmbigu(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	actions := []domain.Action{
		{ID: 1, Description: domain.ActionIncident},
		{ID: 2, Description: domain.ActionSinistre},
		{ID: 3, Description: domain.ActionTravaux},
		{ID: 4, Description: domain.ActionDemandeAdministrative},
		{ID: 5, Description: domain.ActionMutation},
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

// TestDecideActionDistingueSinistreEtIncident vérifie que Claude ne
// confond pas un sinistre (dégât des eaux) avec un simple incident
// technique, les deux étant proposés dans la même liste.
func TestDecideActionDistingueSinistreEtIncident(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	actions := []domain.Action{
		{ID: 1, Description: domain.ActionIncident},
		{ID: 2, Description: domain.ActionSinistre},
		{ID: 3, Description: domain.ActionTravaux},
		{ID: 4, Description: domain.ActionDemandeAdministrative},
		{ID: 5, Description: domain.ActionMutation},
	}
	role := domain.RoleOccupant
	ctxRoutage := domain.ContexteRoutage{
		Role:                 &role,
		CoproprieteReference: "COP1",
		LotsReferences:       []string{"LOT1"},
	}

	decision, err := c.DecideAction(
		ctx, actions, ctxRoutage,
		"Dégât des eaux dans mon appartement",
		"Bonjour, il y a eu une fuite importante ce matin qui a inondé ma salle de bain et le salon, l'eau vient du plafond. J'ai déjà appelé un plombier en urgence, merci de me dire comment déclarer ce sinistre auprès de l'assurance.",
	)
	if err != nil {
		t.Fatalf("DecideAction: %v", err)
	}
	if decision.Action != domain.ActionSinistre {
		t.Errorf("Action = %q, attendu %q — decision=%+v", decision.Action, domain.ActionSinistre, decision)
	}
}

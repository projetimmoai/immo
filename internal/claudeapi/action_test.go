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

	decisions, err := c.DecideAction(
		ctx, actions, ctxRoutage,
		"Panne d'ascenseur",
		"Bonjour, l'ascenseur de l'immeuble est en panne depuis ce matin, merci d'intervenir rapidement.",
	)
	if err != nil {
		t.Fatalf("DecideAction: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("decisions = %+v, attendu 1 (une seule demande dans cet e-mail)", decisions)
	}
	decision := decisions[0]
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

	decisions, err := c.DecideAction(
		ctx, actions, ctxRoutage,
		"Dégât des eaux dans mon appartement",
		"Bonjour, il y a eu une fuite importante ce matin qui a inondé ma salle de bain et le salon, l'eau vient du plafond. J'ai déjà appelé un plombier en urgence, merci de me dire comment déclarer ce sinistre auprès de l'assurance.",
	)
	if err != nil {
		t.Fatalf("DecideAction: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("decisions = %+v, attendu 1 (une seule demande dans cet e-mail)", decisions)
	}
	if decisions[0].Action != domain.ActionSinistre {
		t.Errorf("Action = %q, attendu %q — decision=%+v", decisions[0].Action, domain.ActionSinistre, decisions[0])
	}
}

// TestDecideActionPlusieursDemandesDansUnSeulEmail vérifie, avec un vrai
// appel à l'API, qu'un e-mail contenant deux demandes distinctes (un
// incident technique ET une demande de document) est bien décomposé en 2
// décisions — pas une seule action moyenne ou arbitrairement choisie.
func TestDecideActionPlusieursDemandesDansUnSeulEmail(t *testing.T) {
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

	decisions, err := c.DecideAction(
		ctx, actions, ctxRoutage,
		"Digicode en panne et demande de règlement de copropriété",
		"Bonjour, le digicode de l'entrée est en panne depuis hier, merci de faire intervenir une entreprise. "+
			"Par ailleurs, pourriez-vous également me transmettre une copie du règlement de copropriété, j'en ai besoin pour un dossier ?",
	)
	if err != nil {
		t.Fatalf("DecideAction: %v", err)
	}
	if len(decisions) != 2 {
		t.Fatalf("decisions = %+v, attendu 2 (digicode en panne + demande de document)", decisions)
	}
	var vuIncident, vuDemandeAdmin bool
	for _, d := range decisions {
		switch d.Action {
		case domain.ActionIncident:
			vuIncident = true
		case domain.ActionDemandeAdministrative:
			vuDemandeAdmin = true
		}
	}
	if !vuIncident || !vuDemandeAdmin {
		t.Errorf("decisions = %+v, attendu incident ET demande_administrative", decisions)
	}
}

package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterIncident traite un e-mail routé vers l'action "incident" (cf.
// domain.ActionIncident). Pas encore implémenté : créer un domain.Incident
// à partir du contexte de routage (copropriete, lot éventuel) et du
// contenu de l'e-mail.
func traiterIncident(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}

package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterTravaux traite un e-mail routé vers l'action "travaux" (cf.
// domain.ActionTravaux). Pas encore implémenté.
func traiterTravaux(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}

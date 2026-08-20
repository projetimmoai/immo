package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterAutre traite un e-mail routé vers l'action "autre" (cf.
// domain.ActionAutre) — aucune des actions connues ne correspond, mais
// Claude n'était pas non plus incertain au point de retourner
// "indetermine". Pas encore implémenté.
func traiterAutre(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}

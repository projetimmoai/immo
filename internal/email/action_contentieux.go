package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterContentieux traite un e-mail routé vers l'action "contentieux"
// (cf. domain.ActionContentieux). Pas encore implémenté.
func traiterContentieux(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}

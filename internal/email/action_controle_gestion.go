package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterControleGestion traite un e-mail routé vers l'action
// "controle_gestion" (cf. domain.ActionControleGestion). Pas encore
// implémenté.
func traiterControleGestion(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}

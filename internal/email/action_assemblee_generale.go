package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterAssembleeGenerale traite un e-mail routé vers l'action
// "assemblee_generale" (cf. domain.ActionAssembleeGenerale). Pas encore
// implémenté : aucune table dédiée aux assemblées générales pour l'instant
// (cf. NOTES.md).
func traiterAssembleeGenerale(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}

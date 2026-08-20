package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterSinistre traite un e-mail routé vers l'action "sinistre" (cf.
// domain.ActionSinistre). Pas encore implémenté : créer un domain.Sinistre
// à partir du contexte de routage (copropriete, lot éventuel) et du
// contenu de l'e-mail.
func traiterSinistre(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}

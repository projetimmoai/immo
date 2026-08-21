package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterIncident traite un e-mail classé comme incident (ex: ascenseur en
// panne, digicode en panne) — un dysfonctionnement technique ponctuel dans
// la copropriété, par opposition à un sinistre (dégât des eaux, incendie...)
// ou des travaux déjà planifiés.
//
// TODO: implémenter (création de l'incident en base, notification...).
func traiterIncident(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	return nil
}

package repository

import (
	"context"
	"fmt"
	"net/http"
)

// PersonneRoleAssociee est une ligne de la vue SQL personne_role : un rôle
// qu'une Personne joue, avec la copropriete concernée quand ce rôle y est
// scopé (occupant, coproprietaire, prestataire, conseil_syndical).
// CoproprieteID/CoproprieteNom/CoproprieteReference sont nil pour les rôles
// intrinsèques non scopés à une copropriete en particulier (gestionnaire,
// sys_admin, comptable, direction).
type PersonneRoleAssociee struct {
	Role                 string
	CoproprieteID        *int64
	CoproprieteNom       *string
	CoproprieteReference *string
}

// personneRoleRow est la représentation JSON d'une ligne de la vue personne_role.
type personneRoleRow struct {
	Role                 string  `json:"role"`
	CoproprieteID        *int64  `json:"copropriete_id"`
	CoproprieteNom       *string `json:"copropriete_nom"`
	CoproprieteReference *string `json:"copropriete_reference"`
}

func (r personneRoleRow) toDomain() PersonneRoleAssociee {
	return PersonneRoleAssociee{
		Role:                 r.Role,
		CoproprieteID:        r.CoproprieteID,
		CoproprieteNom:       r.CoproprieteNom,
		CoproprieteReference: r.CoproprieteReference,
	}
}

// ListRolesParPersonne interroge la vue personne_role : tous les rôles
// qu'une Personne joue actuellement, avec la copropriete concernée quand
// pertinent. Remplace la dérivation manuelle (lots + contrats + booléens)
// qui vivait auparavant dans ce package — personne_role reste l'unique
// endroit qui sait dériver un rôle, partagé avec les policies RLS.
func (c *Client) ListRolesParPersonne(ctx context.Context, personneID int64) ([]PersonneRoleAssociee, error) {
	var rows []personneRoleRow
	path := fmt.Sprintf(
		"/personne_role?select=role,copropriete_id,copropriete_nom,copropriete_reference&personne_id=eq.%d",
		personneID,
	)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche des rôles de la personne id=%d: %w", personneID, err)
	}
	result := make([]PersonneRoleAssociee, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.toDomain())
	}
	return result, nil
}

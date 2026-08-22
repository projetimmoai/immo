package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// emailRow est la représentation JSON d'une ligne de la table email.
type emailRow struct {
	ID                   int64      `json:"id"`
	CreatedAt            time.Time  `json:"created_at"`
	MessageID            *string    `json:"message_id"`
	DateReception        time.Time  `json:"date_reception"`
	ExpediteurEmail      string     `json:"expediteur_email"`
	ExpediteurPersonneID *int64     `json:"expediteur_personne_id"`
	Objet                *string    `json:"objet"`
	CorpsTexte           *string    `json:"corps_texte"`
	CorpsHTML            *string    `json:"corps_html"`
	CoproprieteID        *int64     `json:"copropriete_id"`
	LotID                *int64     `json:"lot_id"`
	StatutTraitementID   int64      `json:"statut_traitement_id"`
	TraiteLe             *time.Time `json:"traite_le"`
	TraitePar            *int64     `json:"traite_par"`
	ErreurTraitement     *string    `json:"erreur_traitement"`
}

func (r emailRow) toDomain() *domain.Email {
	return &domain.Email{
		ID:                   r.ID,
		CreatedAt:            r.CreatedAt,
		MessageID:            r.MessageID,
		DateReception:        r.DateReception,
		ExpediteurEmail:      r.ExpediteurEmail,
		ExpediteurPersonneID: r.ExpediteurPersonneID,
		Objet:                r.Objet,
		CorpsTexte:           r.CorpsTexte,
		CorpsHTML:            r.CorpsHTML,
		CoproprieteID:        r.CoproprieteID,
		LotID:                r.LotID,
		StatutTraitementID:   r.StatutTraitementID,
		TraiteLe:             r.TraiteLe,
		TraitePar:            r.TraitePar,
		ErreurTraitement:     r.ErreurTraitement,
	}
}

// emailInsert est la charge utile d'insertion d'un Email : uniquement les
// colonnes fournies par l'appelant (id/created_at sont générés par la base).
type emailInsert struct {
	MessageID            *string   `json:"message_id,omitempty"`
	DateReception        time.Time `json:"date_reception"`
	ExpediteurEmail      string    `json:"expediteur_email"`
	ExpediteurPersonneID *int64    `json:"expediteur_personne_id,omitempty"`
	Objet                *string   `json:"objet,omitempty"`
	CorpsTexte           *string   `json:"corps_texte,omitempty"`
	CorpsHTML            *string   `json:"corps_html,omitempty"`
	CoproprieteID        *int64    `json:"copropriete_id,omitempty"`
	LotID                *int64    `json:"lot_id,omitempty"`
	StatutTraitementID   int64     `json:"statut_traitement_id"`
}

// InsertEmail insère un nouvel Email et retourne la ligne créée (avec son ID
// et son created_at générés par la base). e.StatutTraitementID doit être
// renseigné explicitement par l'appelant (pas de valeur par défaut en base).
func (c *Client) InsertEmail(ctx context.Context, e *domain.Email) (*domain.Email, error) {
	payload := []emailInsert{{
		MessageID:            e.MessageID,
		DateReception:        e.DateReception,
		ExpediteurEmail:      e.ExpediteurEmail,
		ExpediteurPersonneID: e.ExpediteurPersonneID,
		Objet:                e.Objet,
		CorpsTexte:           e.CorpsTexte,
		CorpsHTML:            e.CorpsHTML,
		CoproprieteID:        e.CoproprieteID,
		LotID:                e.LotID,
		StatutTraitementID:   e.StatutTraitementID,
	}}
	var rows []emailRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/email", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion email (expediteur=%s): %w", e.ExpediteurEmail, err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion email (expediteur=%s): %d ligne(s) retournée(s), 1 attendue", e.ExpediteurEmail, len(rows))
	}
	return rows[0].toDomain(), nil
}

// FindEmailByMessageID cherche un Email déjà enregistré par son Message-ID
// RFC822, pour éviter de traiter deux fois le même message. Retourne
// (nil, nil), sans erreur, si aucun Email ne correspond.
func (c *Client) FindEmailByMessageID(ctx context.Context, messageID string) (*domain.Email, error) {
	var rows []emailRow
	path := "/email?select=*&message_id=eq." + escapeFilterValue(messageID) + "&limit=1"
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche email par message_id (%s): %w", messageID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}

// DeleteEmail supprime un Email par son ID (utilisé notamment par les tests
// d'intégration pour nettoyer après eux).
func (c *Client) DeleteEmail(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/email?id=eq.%d", id)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("repository: suppression email id=%d: %w", id, err)
	}
	return nil
}

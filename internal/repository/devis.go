package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// devisRow est la représentation JSON d'une ligne de la table devis.
type devisRow struct {
	ID              int64      `json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	TicketID        int64      `json:"ticket_id"`
	PrestataireID   int64      `json:"prestataire_id"`
	MontantCentimes *int64     `json:"montant_centimes"`
	StatutID        int64      `json:"statut_id"`
	DateDemande     *time.Time `json:"date_demande"`
	DateReception   *time.Time `json:"date_reception"`
	CreePar         *int64     `json:"cree_par"`
}

func (r devisRow) toDomain() *domain.Devis {
	return &domain.Devis{
		ID:              r.ID,
		CreatedAt:       r.CreatedAt,
		TicketID:        r.TicketID,
		PrestataireID:   r.PrestataireID,
		MontantCentimes: r.MontantCentimes,
		StatutID:        r.StatutID,
		DateDemande:     r.DateDemande,
		DateReception:   r.DateReception,
		CreePar:         r.CreePar,
	}
}

// devisInsert est la charge utile d'insertion d'un Devis : uniquement les
// colonnes fournies par l'appelant.
type devisInsert struct {
	TicketID      int64      `json:"ticket_id"`
	PrestataireID int64      `json:"prestataire_id"`
	StatutID      int64      `json:"statut_id"`
	DateDemande   *time.Time `json:"date_demande,omitempty"`
	CreePar       *int64     `json:"cree_par,omitempty"`
}

// InsertDevis insère une nouvelle demande de Devis (phase 3.4.3, statut
// "en_attente") et retourne la ligne créée.
func (c *Client) InsertDevis(ctx context.Context, d *domain.Devis) (*domain.Devis, error) {
	payload := []devisInsert{{
		TicketID:      d.TicketID,
		PrestataireID: d.PrestataireID,
		StatutID:      d.StatutID,
		DateDemande:   d.DateDemande,
		CreePar:       d.CreePar,
	}}
	var rows []devisRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/devis", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion devis (ticket_id=%d): %w", d.TicketID, err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion devis: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// devisPatch est la charge utile de mise à jour partielle d'un Devis :
// seuls les champs non-nil sont envoyés (omitempty).
type devisPatch struct {
	MontantCentimes *int64     `json:"montant_centimes,omitempty"`
	StatutID        *int64     `json:"statut_id,omitempty"`
	DateReception   *time.Time `json:"date_reception,omitempty"`
}

func (c *Client) updateDevis(ctx context.Context, devisID int64, patch devisPatch) error {
	path := fmt.Sprintf("/devis?id=eq.%d", devisID)
	if err := c.do(ctx, http.MethodPatch, path, patch, nil); err != nil {
		return fmt.Errorf("repository: mise à jour devis id=%d: %w", devisID, err)
	}
	return nil
}

// EnregistrerReceptionDevis enregistre le montant extrait d'un devis reçu
// (phase 3.4.6) et son passage au statut "recu".
func (c *Client) EnregistrerReceptionDevis(ctx context.Context, devisID, statutRecuID, montantCentimes int64, dateReception time.Time) error {
	return c.updateDevis(ctx, devisID, devisPatch{
		MontantCentimes: &montantCentimes,
		StatutID:        &statutRecuID,
		DateReception:   &dateReception,
	})
}

// MarquerDevisStatut change simplement le statut d'un Devis (retenu/rejete,
// une fois la décision prise).
func (c *Client) MarquerDevisStatut(ctx context.Context, devisID, statutID int64) error {
	return c.updateDevis(ctx, devisID, devisPatch{StatutID: &statutID})
}

// FindDevisByID retrouve un Devis par son ID. Retourne (nil, nil), sans
// erreur, si aucun devis ne correspond.
func (c *Client) FindDevisByID(ctx context.Context, id int64) (*domain.Devis, error) {
	var rows []devisRow
	path := fmt.Sprintf("/devis?select=*&id=eq.%d&limit=1", id)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche devis id=%d: %w", id, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}

// ListDevisByTicket retourne tous les Devis d'un Ticket (mise en
// concurrence, seuil B : plusieurs devis pour un même ticket).
func (c *Client) ListDevisByTicket(ctx context.Context, ticketID int64) ([]*domain.Devis, error) {
	var rows []devisRow
	path := fmt.Sprintf("/devis?select=*&ticket_id=eq.%d", ticketID)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: listage devis ticket_id=%d: %w", ticketID, err)
	}
	result := make([]*domain.Devis, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.toDomain())
	}
	return result, nil
}

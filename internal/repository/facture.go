package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// factureRow est la représentation JSON d'une ligne de la table facture.
type factureRow struct {
	ID                     int64      `json:"id"`
	CreatedAt              time.Time  `json:"created_at"`
	TicketID               int64      `json:"ticket_id"`
	PrestataireID          int64      `json:"prestataire_id"`
	MontantFactureCentimes int64      `json:"montant_facture_centimes"`
	StatutID               int64      `json:"statut_id"`
	DateReception          *time.Time `json:"date_reception"`
	DateValidation         *time.Time `json:"date_validation"`
	ValidePar              *int64     `json:"valide_par"`
	DatePaiement           *time.Time `json:"date_paiement"`
	PayePar                *int64     `json:"paye_par"`
	DateRapprochement      *time.Time `json:"date_rapprochement"`
	RapprochePar           *int64     `json:"rapproche_par"`
	CreePar                *int64     `json:"cree_par"`
}

func (r factureRow) toDomain() *domain.Facture {
	return &domain.Facture{
		ID:                     r.ID,
		CreatedAt:              r.CreatedAt,
		TicketID:               r.TicketID,
		PrestataireID:          r.PrestataireID,
		MontantFactureCentimes: r.MontantFactureCentimes,
		StatutID:               r.StatutID,
		DateReception:          r.DateReception,
		DateValidation:         r.DateValidation,
		ValidePar:              r.ValidePar,
		DatePaiement:           r.DatePaiement,
		PayePar:                r.PayePar,
		DateRapprochement:      r.DateRapprochement,
		RapprochePar:           r.RapprochePar,
		CreePar:                r.CreePar,
	}
}

// factureInsert est la charge utile d'insertion d'une Facture : uniquement
// les colonnes fournies par l'appelant.
type factureInsert struct {
	TicketID               int64      `json:"ticket_id"`
	PrestataireID          int64      `json:"prestataire_id"`
	MontantFactureCentimes int64      `json:"montant_facture_centimes"`
	StatutID               int64      `json:"statut_id"`
	DateReception          *time.Time `json:"date_reception,omitempty"`
	CreePar                *int64     `json:"cree_par,omitempty"`
}

// InsertFacture insère une nouvelle Facture (réception initiale, phase
// 5.5.3) et retourne la ligne créée. f.StatutID doit être renseigné
// explicitement par l'appelant (pas de valeur par défaut en base) — cf.
// FactureStatutID pour le résoudre à partir d'une description (voir les
// constantes domain.FactureStatut*).
func (c *Client) InsertFacture(ctx context.Context, f *domain.Facture) (*domain.Facture, error) {
	payload := []factureInsert{{
		TicketID:               f.TicketID,
		PrestataireID:          f.PrestataireID,
		MontantFactureCentimes: f.MontantFactureCentimes,
		StatutID:               f.StatutID,
		DateReception:          f.DateReception,
		CreePar:                f.CreePar,
	}}
	var rows []factureRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/facture", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion facture (ticket_id=%d): %w", f.TicketID, err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion facture: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// facturePatch est la charge utile de mise à jour partielle d'une Facture :
// seuls les champs non-nil sont envoyés (omitempty).
type facturePatch struct {
	StatutID          *int64     `json:"statut_id,omitempty"`
	DateValidation    *time.Time `json:"date_validation,omitempty"`
	ValidePar         *int64     `json:"valide_par,omitempty"`
	DatePaiement      *time.Time `json:"date_paiement,omitempty"`
	PayePar           *int64     `json:"paye_par,omitempty"`
	DateRapprochement *time.Time `json:"date_rapprochement,omitempty"`
	RapprochePar      *int64     `json:"rapproche_par,omitempty"`
}

func (c *Client) updateFacture(ctx context.Context, factureID int64, patch facturePatch) error {
	path := fmt.Sprintf("/facture?id=eq.%d", factureID)
	if err := c.do(ctx, http.MethodPatch, path, patch, nil); err != nil {
		return fmt.Errorf("repository: mise à jour facture id=%d: %w", factureID, err)
	}
	return nil
}

// ValiderFacture enregistre la validation du montant facturé face au devis
// ou au contrat (phase 5.5.4) : la facture est prête à payer.
func (c *Client) ValiderFacture(ctx context.Context, factureID, statutValideeID, validePar int64, dateValidation time.Time) error {
	return c.updateFacture(ctx, factureID, facturePatch{
		StatutID:       &statutValideeID,
		DateValidation: &dateValidation,
		ValidePar:      &validePar,
	})
}

// MettreFactureEnPaiement enregistre la mise en paiement d'une Facture
// (phase 5.5.5) — l'appelant (cf. service.IncidentService) est responsable
// de vérifier au préalable que la facture est validée ET la vérification de
// l'incident positive : cette fonction se contente d'écrire, elle
// n'applique aucune règle métier.
func (c *Client) MettreFactureEnPaiement(ctx context.Context, factureID, statutPayeeID, payePar int64, datePaiement time.Time) error {
	return c.updateFacture(ctx, factureID, facturePatch{
		StatutID:     &statutPayeeID,
		DatePaiement: &datePaiement,
		PayePar:      &payePar,
	})
}

// RapprocherFacture enregistre le rapprochement comptable d'une Facture
// (phase 5.5.6).
func (c *Client) RapprocherFacture(ctx context.Context, factureID, statutRapprocheeID, rapprochePar int64, dateRapprochement time.Time) error {
	return c.updateFacture(ctx, factureID, facturePatch{
		StatutID:          &statutRapprocheeID,
		DateRapprochement: &dateRapprochement,
		RapprochePar:      &rapprochePar,
	})
}

// FindFactureByTicketID retrouve la Facture d'un Ticket. Retourne (nil,
// nil), sans erreur, si aucune facture ne correspond — un ticket peut ne
// pas encore avoir reçu de facture.
func (c *Client) FindFactureByTicketID(ctx context.Context, ticketID int64) (*domain.Facture, error) {
	var rows []factureRow
	path := fmt.Sprintf("/facture?select=*&ticket_id=eq.%d&limit=1", ticketID)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche facture ticket_id=%d: %w", ticketID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}

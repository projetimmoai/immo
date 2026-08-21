package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// coproprieteRow est la représentation JSON d'une ligne de la table copropriete.
type coproprieteRow struct {
	ID                                int64      `json:"id"`
	CreatedAt                         *time.Time `json:"created_at"`
	SyndicID                          *int64     `json:"syndic_id"`
	EstSyndic                         *bool      `json:"est_syndic"`
	AdresseCodePostal                 *string    `json:"adresse_code_postal"`
	AdresseVille                      *string    `json:"adresse_ville"`
	AdressePaysCode                   *string    `json:"adresse_pays_code"`
	ExerciceDebut                     *time.Time `json:"exercice_debut"`
	ExerciceFin                       *time.Time `json:"exercice_fin"`
	AppelChargesFrequenceID           *int64     `json:"appel_charges_frequence_id"`
	AppelChargesDate                  *time.Time `json:"appel_charges_date"`
	AppelChargesNumJoursAvantEcheance *int64     `json:"appel_charges_num_jours_avant_echeance"`
	Nom                               *string    `json:"nom"`
	AdresseRegion                     *string    `json:"adresse_region"`
	ClesRepartition                   *float32   `json:"cles_repartition"`
	ArreteComptableJour               *int64     `json:"arrete_comptable_jour"`
	ArreteComptableMois               *int64     `json:"arrete_comptable_mois"`
	NumeroImmatriculation             *string    `json:"numero_immatriculation"`
	NumeroMandat                      *string    `json:"numero_mandat"`
	MandatDateDebut                   *time.Time `json:"mandat_date_debut"`
	MandatDureeEnMois                 *int64     `json:"mandat_duree_en_mois"`
	CodeICS                           *string    `json:"code_ics"`
	IBAN                              *string    `json:"iban"`
	BIC                               *string    `json:"bic"`
	MisAJour                          *time.Time `json:"mis_a_jour"`
	CreePar                           *int64     `json:"cree_par"`
	Reference                         string     `json:"reference"`
}

func (r coproprieteRow) toDomain() *domain.Copropriete {
	return &domain.Copropriete{
		ID:                                r.ID,
		CreatedAt:                         r.CreatedAt,
		SyndicID:                          r.SyndicID,
		EstSyndic:                         r.EstSyndic,
		AdresseCodePostal:                 r.AdresseCodePostal,
		AdresseVille:                      r.AdresseVille,
		AdressePaysCode:                   r.AdressePaysCode,
		ExerciceDebut:                     r.ExerciceDebut,
		ExerciceFin:                       r.ExerciceFin,
		AppelChargesFrequenceID:           r.AppelChargesFrequenceID,
		AppelChargesDate:                  r.AppelChargesDate,
		AppelChargesNumJoursAvantEcheance: r.AppelChargesNumJoursAvantEcheance,
		Nom:                               r.Nom,
		AdresseRegion:                     r.AdresseRegion,
		ClesRepartition:                   r.ClesRepartition,
		ArreteComptableJour:               r.ArreteComptableJour,
		ArreteComptableMois:               r.ArreteComptableMois,
		NumeroImmatriculation:             r.NumeroImmatriculation,
		NumeroMandat:                      r.NumeroMandat,
		MandatDateDebut:                   r.MandatDateDebut,
		MandatDureeEnMois:                 r.MandatDureeEnMois,
		CodeICS:                           r.CodeICS,
		IBAN:                              r.IBAN,
		BIC:                               r.BIC,
		MisAJour:                          r.MisAJour,
		CreePar:                           r.CreePar,
		Reference:                         r.Reference,
	}
}

// coproprieteInsert est la charge utile d'insertion d'une Copropriete :
// uniquement les colonnes fournies par l'appelant (id/created_at/reference
// sont générés par la base — reference via une séquence dédiée, cf. migration).
type coproprieteInsert struct {
	SyndicID                          *int64     `json:"syndic_id,omitempty"`
	EstSyndic                         *bool      `json:"est_syndic,omitempty"`
	AdresseCodePostal                 *string    `json:"adresse_code_postal,omitempty"`
	AdresseVille                      *string    `json:"adresse_ville,omitempty"`
	AdressePaysCode                   *string    `json:"adresse_pays_code,omitempty"`
	ExerciceDebut                     *time.Time `json:"exercice_debut,omitempty"`
	ExerciceFin                       *time.Time `json:"exercice_fin,omitempty"`
	AppelChargesFrequenceID           *int64     `json:"appel_charges_frequence_id,omitempty"`
	AppelChargesDate                  *time.Time `json:"appel_charges_date,omitempty"`
	AppelChargesNumJoursAvantEcheance *int64     `json:"appel_charges_num_jours_avant_echeance,omitempty"`
	Nom                               *string    `json:"nom,omitempty"`
	AdresseRegion                     *string    `json:"adresse_region,omitempty"`
	ClesRepartition                   *float32   `json:"cles_repartition,omitempty"`
	ArreteComptableJour               *int64     `json:"arrete_comptable_jour,omitempty"`
	ArreteComptableMois               *int64     `json:"arrete_comptable_mois,omitempty"`
	NumeroImmatriculation             *string    `json:"numero_immatriculation,omitempty"`
	NumeroMandat                      *string    `json:"numero_mandat,omitempty"`
	MandatDateDebut                   *time.Time `json:"mandat_date_debut,omitempty"`
	MandatDureeEnMois                 *int64     `json:"mandat_duree_en_mois,omitempty"`
	CodeICS                           *string    `json:"code_ics,omitempty"`
	IBAN                              *string    `json:"iban,omitempty"`
	BIC                               *string    `json:"bic,omitempty"`
	CreePar                           *int64     `json:"cree_par,omitempty"`
}

// InsertCopropriete insère une nouvelle Copropriete et retourne la ligne
// créée (avec son ID et sa Reference, ex "COP3", générés par la base).
func (c *Client) InsertCopropriete(ctx context.Context, cop *domain.Copropriete) (*domain.Copropriete, error) {
	payload := []coproprieteInsert{{
		SyndicID:                          cop.SyndicID,
		EstSyndic:                         cop.EstSyndic,
		AdresseCodePostal:                 cop.AdresseCodePostal,
		AdresseVille:                      cop.AdresseVille,
		AdressePaysCode:                   cop.AdressePaysCode,
		ExerciceDebut:                     cop.ExerciceDebut,
		ExerciceFin:                       cop.ExerciceFin,
		AppelChargesFrequenceID:           cop.AppelChargesFrequenceID,
		AppelChargesDate:                  cop.AppelChargesDate,
		AppelChargesNumJoursAvantEcheance: cop.AppelChargesNumJoursAvantEcheance,
		Nom:                               cop.Nom,
		AdresseRegion:                     cop.AdresseRegion,
		ClesRepartition:                   cop.ClesRepartition,
		ArreteComptableJour:               cop.ArreteComptableJour,
		ArreteComptableMois:               cop.ArreteComptableMois,
		NumeroImmatriculation:             cop.NumeroImmatriculation,
		NumeroMandat:                      cop.NumeroMandat,
		MandatDateDebut:                   cop.MandatDateDebut,
		MandatDureeEnMois:                 cop.MandatDureeEnMois,
		CodeICS:                           cop.CodeICS,
		IBAN:                              cop.IBAN,
		BIC:                               cop.BIC,
		CreePar:                           cop.CreePar,
	}}
	var rows []coproprieteRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/copropriete", payload, "return=representation", &rows); err != nil {
		nom := ""
		if cop.Nom != nil {
			nom = *cop.Nom
		}
		return nil, fmt.Errorf("repository: insertion copropriete (nom=%s): %w", nom, err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion copropriete: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// ListCopropriete retourne toutes les coproprietes en base.
func (c *Client) ListCopropriete(ctx context.Context) ([]*domain.Copropriete, error) {
	var rows []coproprieteRow
	if err := c.do(ctx, http.MethodGet, "/copropriete?select=*", nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: listage copropriete: %w", err)
	}
	result := make([]*domain.Copropriete, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.toDomain())
	}
	return result, nil
}

// DeleteCopropriete supprime une Copropriete par son ID (utilisé notamment
// par les tests d'intégration pour nettoyer après eux).
func (c *Client) DeleteCopropriete(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/copropriete?id=eq.%d", id)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("repository: suppression copropriete id=%d: %w", id, err)
	}
	return nil
}

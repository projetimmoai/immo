package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// incidentRow est la représentation JSON d'une ligne de la table incident.
type incidentRow struct {
	TicketID               int64      `json:"ticket_id"`
	CategorieTechniqueID   *int64     `json:"categorie_technique_id"`
	UrgenceID              *int64     `json:"urgence_id"`
	DateResolution         *time.Time `json:"date_resolution"`
	PrestataireID          *int64     `json:"prestataire_id"`
	MontantEstimeCentimes  *int64     `json:"montant_estime_centimes"`
	RapportIntervention    *string    `json:"rapport_intervention"`
	ModeVerificationID     *int64     `json:"mode_verification_id"`
	VerificationResultatID *int64     `json:"verification_resultat_id"`
}

func (r incidentRow) toDomain() *domain.Incident {
	return &domain.Incident{
		TicketID:               r.TicketID,
		CategorieTechniqueID:   r.CategorieTechniqueID,
		UrgenceID:              r.UrgenceID,
		DateResolution:         r.DateResolution,
		PrestataireID:          r.PrestataireID,
		MontantEstimeCentimes:  r.MontantEstimeCentimes,
		RapportIntervention:    r.RapportIntervention,
		ModeVerificationID:     r.ModeVerificationID,
		VerificationResultatID: r.VerificationResultatID,
	}
}

// CreerIncidentInput regroupe les champs nécessaires pour créer, en une
// seule opération atomique, le Ticket et son Incident détail — cf.
// InsertIncident.
type CreerIncidentInput struct {
	ActionID              int64  // FK -> action.id, NOT NULL
	StatutID              int64  // FK -> ticket_statut.id, NOT NULL
	SourceID              int64  // FK -> ticket_source.id, NOT NULL
	CoproprieteID         int64  // FK -> copropriete.id, NOT NULL
	LotID                 *int64 // FK -> lot.id
	ParentID              *int64 // FK -> ticket.id
	CreePar               *int64 // FK -> personne.id
	CategorieTechniqueID  *int64 // FK -> categorie_technique.id
	UrgenceID             *int64 // FK -> niveau_urgence.id
	MontantEstimeCentimes *int64
}

// creerIncidentRPCArgs est la charge utile envoyée à la fonction Postgres
// creer_incident (RPC) — les noms de champs JSON doivent correspondre
// exactement aux noms des paramètres de la fonction.
type creerIncidentRPCArgs struct {
	PActionID              int64  `json:"p_action_id"`
	PStatutID              int64  `json:"p_statut_id"`
	PSourceID              int64  `json:"p_source_id"`
	PCoproprieteID         int64  `json:"p_copropriete_id"`
	PLotID                 *int64 `json:"p_lot_id"`
	PParentID              *int64 `json:"p_parent_id"`
	PCreePar               *int64 `json:"p_cree_par"`
	PCategorieTechniqueID  *int64 `json:"p_categorie_technique_id"`
	PUrgenceID             *int64 `json:"p_urgence_id"`
	PMontantEstimeCentimes *int64 `json:"p_montant_estime_centimes"`
}

// InsertIncident crée un Ticket (action = incident) et son Incident détail
// en une seule opération atomique, via la fonction Postgres creer_incident
// appelée en RPC — deux écritures liées, conformément à CLAUDE.md (l'API
// REST ne permet pas d'ouvrir une transaction à cheval sur plusieurs appels
// HTTP). Retourne le Ticket créé (avec son ID et sa référence générés par la
// base) et l'Incident correspondant (reconstruit depuis in : la fonction ne
// retourne que le ticket, les valeurs de l'incident sont déjà connues de
// l'appelant).
func (c *Client) InsertIncident(ctx context.Context, in CreerIncidentInput) (*domain.Ticket, *domain.Incident, error) {
	args := creerIncidentRPCArgs{
		PActionID:              in.ActionID,
		PStatutID:              in.StatutID,
		PSourceID:              in.SourceID,
		PCoproprieteID:         in.CoproprieteID,
		PLotID:                 in.LotID,
		PParentID:              in.ParentID,
		PCreePar:               in.CreePar,
		PCategorieTechniqueID:  in.CategorieTechniqueID,
		PUrgenceID:             in.UrgenceID,
		PMontantEstimeCentimes: in.MontantEstimeCentimes,
	}
	var row ticketRow
	if err := c.CallRPC(ctx, "creer_incident", args, &row); err != nil {
		return nil, nil, fmt.Errorf("repository: création incident (copropriete_id=%d): %w", in.CoproprieteID, err)
	}
	ticket := row.toDomain()
	incident := &domain.Incident{
		TicketID:              ticket.ID,
		CategorieTechniqueID:  in.CategorieTechniqueID,
		UrgenceID:             in.UrgenceID,
		MontantEstimeCentimes: in.MontantEstimeCentimes,
	}
	return ticket, incident, nil
}

// incidentPatch est la charge utile de mise à jour partielle d'un Incident :
// seuls les champs non-nil sont envoyés (omitempty), pour ne modifier que ce
// que l'appelant fournit explicitement.
type incidentPatch struct {
	PrestataireID          *int64     `json:"prestataire_id,omitempty"`
	MontantEstimeCentimes  *int64     `json:"montant_estime_centimes,omitempty"`
	RapportIntervention    *string    `json:"rapport_intervention,omitempty"`
	ModeVerificationID     *int64     `json:"mode_verification_id,omitempty"`
	VerificationResultatID *int64     `json:"verification_resultat_id,omitempty"`
	DateResolution         *time.Time `json:"date_resolution,omitempty"`
}

// updateIncident met à jour partiellement l'Incident identifié par
// ticketID : seuls les champs non-nil de patch sont modifiés. Non exportée :
// les appelants passent par les fonctions Set* ci-dessous, qui documentent
// chacune à quelle étape du graphe elles correspondent.
func (c *Client) updateIncident(ctx context.Context, ticketID int64, patch incidentPatch) error {
	path := fmt.Sprintf("/incident?ticket_id=eq.%d", ticketID)
	if err := c.do(ctx, http.MethodPatch, path, patch, nil); err != nil {
		return fmt.Errorf("repository: mise à jour incident ticket_id=%d: %w", ticketID, err)
	}
	return nil
}

// SetIncidentPrestataire enregistre le prestataire retenu pour un Incident
// (phase 3.3, sélection du prestataire).
func (c *Client) SetIncidentPrestataire(ctx context.Context, ticketID, prestataireID int64) error {
	return c.updateIncident(ctx, ticketID, incidentPatch{PrestataireID: &prestataireID})
}

// SetIncidentRapportIntervention enregistre le rapport de fin d'intervention
// du prestataire (phase 4.5).
func (c *Client) SetIncidentRapportIntervention(ctx context.Context, ticketID int64, rapport string) error {
	return c.updateIncident(ctx, ticketID, incidentPatch{RapportIntervention: &rapport})
}

// SetIncidentModeVerification enregistre le mode de vérification retenu
// (phase 5.0), et immédiatement le résultat quand le mode le détermine déjà
// (ex: "jugee_inutile" vaut positive) — resultatID peut être nil si le
// résultat reste à venir (ex: confirmation demandée à l'occupant).
func (c *Client) SetIncidentModeVerification(ctx context.Context, ticketID, modeID int64, resultatID *int64) error {
	return c.updateIncident(ctx, ticketID, incidentPatch{ModeVerificationID: &modeID, VerificationResultatID: resultatID})
}

// SetIncidentVerificationResultat enregistre le résultat de la vérification
// (phase 5.1/5.2), et la date de résolution si le résultat est positif.
func (c *Client) SetIncidentVerificationResultat(ctx context.Context, ticketID, resultatID int64, dateResolution *time.Time) error {
	return c.updateIncident(ctx, ticketID, incidentPatch{VerificationResultatID: &resultatID, DateResolution: dateResolution})
}

// FindIncidentByTicketID retrouve un Incident par son TicketID. Retourne
// (nil, nil), sans erreur, si aucun incident ne correspond.
func (c *Client) FindIncidentByTicketID(ctx context.Context, ticketID int64) (*domain.Incident, error) {
	var rows []incidentRow
	path := fmt.Sprintf("/incident?select=*&ticket_id=eq.%d&limit=1", ticketID)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche incident ticket_id=%d: %w", ticketID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}

// categorieTechniqueRow est la représentation JSON d'une ligne de la table
// categorie_technique.
type categorieTechniqueRow struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Description *string   `json:"description"`
}

// ListCategorieTechnique retourne tout le catalogue categorie_technique
// (partagé entre Incident et Contrat, cf. domain.CategorieTechnique) —
// utilisé notamment pour proposer les catégories possibles à Claude lors de
// la qualification d'un incident (cf. claudeapi.QualifierIncident).
func (c *Client) ListCategorieTechnique(ctx context.Context) ([]domain.CategorieTechnique, error) {
	var rows []categorieTechniqueRow
	if err := c.do(ctx, http.MethodGet, "/categorie_technique?select=*", nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: listage categorie_technique: %w", err)
	}
	result := make([]domain.CategorieTechnique, 0, len(rows))
	for _, r := range rows {
		result = append(result, domain.CategorieTechnique{ID: r.ID, CreatedAt: r.CreatedAt, Description: r.Description})
	}
	return result, nil
}

// niveauUrgenceRow est la représentation JSON d'une ligne de la table
// niveau_urgence.
type niveauUrgenceRow struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
}

// ListNiveauUrgence retourne tout le catalogue niveau_urgence — utilisé
// notamment pour proposer les niveaux possibles à Claude lors de la
// qualification d'un incident (cf. claudeapi.QualifierIncident).
func (c *Client) ListNiveauUrgence(ctx context.Context) ([]domain.NiveauUrgence, error) {
	var rows []niveauUrgenceRow
	if err := c.do(ctx, http.MethodGet, "/niveau_urgence?select=*", nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: listage niveau_urgence: %w", err)
	}
	result := make([]domain.NiveauUrgence, 0, len(rows))
	for _, r := range rows {
		result = append(result, domain.NiveauUrgence{ID: r.ID, CreatedAt: r.CreatedAt, Description: r.Description})
	}
	return result, nil
}

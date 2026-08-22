package service

import (
	"context"
	"fmt"
	"time"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

// incidentRepo est la portion de repository.Client utilisée ici — une
// interface étroite plutôt que le type concret, pour pouvoir tester avec des
// faux (cf. incident_test.go).
type incidentRepo interface {
	ActionID(ctx context.Context, description string) (int64, error)
	TicketStatutID(ctx context.Context, description string) (int64, error)
	CategorieTechniqueID(ctx context.Context, description string) (int64, error)
	NiveauUrgenceID(ctx context.Context, description string) (int64, error)
	ListCategorieTechnique(ctx context.Context) ([]domain.CategorieTechnique, error)
	ListNiveauUrgence(ctx context.Context) ([]domain.NiveauUrgence, error)
	InsertIncident(ctx context.Context, in repository.CreerIncidentInput) (*domain.Ticket, *domain.Incident, error)
	FindCoproprieteByID(ctx context.Context, id int64) (*domain.Copropriete, error)
	FindContratActif(ctx context.Context, coproprieteID, categorieTechniqueID int64) (*repository.ContratActif, error)
	SetIncidentPrestataire(ctx context.Context, ticketID, prestataireID int64) error
	UpdateTicketStatut(ctx context.Context, ticketID, statutID int64) error
	SetIncidentRapportIntervention(ctx context.Context, ticketID int64, rapport string) error
	ModeVerificationID(ctx context.Context, description string) (int64, error)
	VerificationResultatID(ctx context.Context, description string) (int64, error)
	SetIncidentModeVerification(ctx context.Context, ticketID, modeID int64, resultatID *int64) error
	SetIncidentVerificationResultat(ctx context.Context, ticketID, resultatID int64, dateResolution *time.Time) error
	FindIncidentByTicketID(ctx context.Context, ticketID int64) (*domain.Incident, error)
	FactureStatutID(ctx context.Context, description string) (int64, error)
	InsertFacture(ctx context.Context, f *domain.Facture) (*domain.Facture, error)
	ValiderFacture(ctx context.Context, factureID, statutValideeID, validePar int64, dateValidation time.Time) error
	MettreFactureEnPaiement(ctx context.Context, factureID, statutPayeeID, payePar int64, datePaiement time.Time) error
	RapprocherFacture(ctx context.Context, factureID, statutRapprocheeID, rapprochePar int64, dateRapprochement time.Time) error
	FindFactureByTicketID(ctx context.Context, ticketID int64) (*domain.Facture, error)
}

// incidentQualifieur est la portion de claudeapi.Client utilisée ici — une
// interface étroite pour pouvoir tester avec un faux.
type incidentQualifieur interface {
	QualifierIncident(ctx context.Context, categories []domain.CategorieTechnique, urgences []domain.NiveauUrgence, objet, corpsTexte string) (claudeapi.QualificationIncident, error)
}

// IncidentService orchestre le cycle de vie d'un Incident — la tranche
// verticale "cas simple" du graphe de cycle de vie d'un incident (cf.
// docs/cycle-vie-incident.md) : création qualifiée, sélection du
// prestataire quand un contrat actif existe, comparaison au plafond D
// (ordre de service), suivi de l'intervention, vérification (voie
// confirmation occupant ou jugée inutile), facture et mise en paiement
// gatée sur la vérification, rapprochement comptable et clôture.
//
// Volontairement hors de cette tranche (laissés pour un prochain jalon,
// cf. décomposition discutée) : devis/mise en concurrence, décision ou avis
// du conseil syndical, vote en assemblée générale, réclamation/litige,
// vérification par un gestionnaire humain sur place, sélection d'un
// prestataire dans un répertoire par zone d'intervention (pas encore
// modélisé), et l'automatisation des relances (aucun worker périodique
// pour l'instant : les étapes "en attente" sont enregistrées mais ne sont
// pas relancées automatiquement).
type IncidentService struct {
	Repo   incidentRepo
	Claude incidentQualifieur
}

// CreerIncidentInput regroupe ce que l'appelant (cf. internal/email,
// traiterIncident) sait déjà au moment de la création : la source du
// ticket, la copropriété retenue, et le contenu du signalement.
type CreerIncidentInput struct {
	SourceID      int64  // FK -> ticket_source.id, NOT NULL
	CoproprieteID int64  // FK -> copropriete.id, NOT NULL
	LotID         *int64 // FK -> lot.id ; nil = partie commune (pas encore résolu par cette tranche, cf. doc)
	ParentID      *int64 // FK -> ticket.id, si ce signalement fait suite à un autre ticket
	CreePar       *int64 // FK -> personne.id (gestionnaire à l'origine, nil si créé automatiquement depuis un signalement occupant)
	Objet         string
	CorpsTexte    string
}

// CreerIncident couvre les phases 1.10 (création du ticket), 2 (qualification)
// et le début de la phase 3 (sélection du prestataire, comparaison au
// plafond D) du graphe.
//
// Sélection du prestataire simplifiée pour cette tranche (cf.
// repository.FindContratActif) : seul un contrat de maintenance actif est
// cherché — pas de répertoire de prestataires par zone d'intervention.
// Si aucun contrat actif n'existe, ou si le montant estimé dépasse le
// plafond D (ordre de service, nécessitant un devis — pas encore
// implémenté), le ticket reste en statut "en_attente_gestionnaire" : un
// humain doit reprendre la main.
func (s *IncidentService) CreerIncident(ctx context.Context, in CreerIncidentInput) (*domain.Ticket, *domain.Incident, error) {
	if in.SourceID <= 0 {
		return nil, nil, fmt.Errorf("service: création incident: source_id est obligatoire")
	}
	if in.CoproprieteID <= 0 {
		return nil, nil, fmt.Errorf("service: création incident: copropriete_id est obligatoire")
	}

	actionID, err := s.Repo.ActionID(ctx, domain.ActionIncident)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: résolution de l'action: %w", err)
	}
	statutNouveauID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutNouveau)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: résolution du statut initial: %w", err)
	}

	categories, err := s.Repo.ListCategorieTechnique(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: chargement des catégories techniques: %w", err)
	}
	urgences, err := s.Repo.ListNiveauUrgence(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: chargement des niveaux d'urgence: %w", err)
	}

	qualif, err := s.Claude.QualifierIncident(ctx, categories, urgences, in.Objet, in.CorpsTexte)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: qualification: %w", err)
	}

	var categorieTechniqueID *int64
	if qualif.CategorieTechnique != nil {
		id, err := s.Repo.CategorieTechniqueID(ctx, *qualif.CategorieTechnique)
		if err != nil {
			return nil, nil, fmt.Errorf("service: création incident: résolution de la catégorie technique %q: %w", *qualif.CategorieTechnique, err)
		}
		categorieTechniqueID = &id
	}
	urgenceID, err := s.Repo.NiveauUrgenceID(ctx, qualif.Urgence)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: résolution du niveau d'urgence %q: %w", qualif.Urgence, err)
	}

	ticket, incident, err := s.Repo.InsertIncident(ctx, repository.CreerIncidentInput{
		ActionID:              actionID,
		StatutID:              statutNouveauID,
		SourceID:              in.SourceID,
		CoproprieteID:         in.CoproprieteID,
		LotID:                 in.LotID,
		ParentID:              in.ParentID,
		CreePar:               in.CreePar,
		CategorieTechniqueID:  categorieTechniqueID,
		UrgenceID:             &urgenceID,
		MontantEstimeCentimes: qualif.MontantEstimeCentimes,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: %w", err)
	}

	if err := s.selectionnerPrestataireEtDemarrer(ctx, ticket, incident, categorieTechniqueID); err != nil {
		return ticket, incident, fmt.Errorf("service: création incident %s: sélection du prestataire: %w", ticket.Reference, err)
	}
	return ticket, incident, nil
}

// selectionnerPrestataireEtDemarrer couvre la phase 3.3 (sélection du
// prestataire, simplifiée : contrat actif seulement) puis la comparaison au
// plafond D (phase 3.4.1). Ne retourne une erreur que pour un problème
// technique (accès base) — l'absence de prestataire ou un montant au-delà
// du plafond D ne sont pas des erreurs : ils laissent le ticket en attente
// d'un gestionnaire humain (cf. doc du champ StatutID correspondant).
func (s *IncidentService) selectionnerPrestataireEtDemarrer(ctx context.Context, ticket *domain.Ticket, incident *domain.Incident, categorieTechniqueID *int64) error {
	statutAttenteGestionnaireID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteGestionnaire)
	if err != nil {
		return fmt.Errorf("résolution du statut en_attente_gestionnaire: %w", err)
	}

	if categorieTechniqueID == nil {
		// Catégorie indéterminée : impossible de chercher un contrat par
		// catégorie technique — recherche humaine du prestataire.
		return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutAttenteGestionnaireID)
	}

	contrat, err := s.Repo.FindContratActif(ctx, ticket.CoproprieteID, *categorieTechniqueID)
	if err != nil {
		return fmt.Errorf("recherche d'un contrat actif: %w", err)
	}
	if contrat == nil {
		// Phase 3.3.3 : aucun prestataire du répertoire ne correspond (ici,
		// aucun contrat actif) → recherche humaine.
		return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutAttenteGestionnaireID)
	}
	if err := s.Repo.SetIncidentPrestataire(ctx, ticket.ID, contrat.EntrepriseID); err != nil {
		return fmt.Errorf("enregistrement du prestataire retenu: %w", err)
	}

	// Phase 3.4.1 : comparaison au plafond D (ordre de service).
	cop, err := s.Repo.FindCoproprieteByID(ctx, ticket.CoproprieteID)
	if err != nil {
		return fmt.Errorf("chargement de la copropriete: %w", err)
	}
	depasse := cop == nil || cop.PlafondOrdreServiceCentimes == nil ||
		incident.MontantEstimeCentimes == nil ||
		*incident.MontantEstimeCentimes > *cop.PlafondOrdreServiceCentimes
	if depasse {
		// Au-delà du plafond D : un devis est nécessaire (phase 3.4.2 et
		// suivantes) — pas encore implémenté dans cette tranche, un
		// gestionnaire humain doit reprendre la main.
		return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutAttenteGestionnaireID)
	}

	// Sous le plafond D : intervention immédiate, sans devis préalable
	// (phase 3.4.1) → suivi de l'intervention, en attente du prestataire.
	statutAttenteTiersID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteTiers)
	if err != nil {
		return fmt.Errorf("résolution du statut en_attente_tiers: %w", err)
	}
	return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutAttenteTiersID)
}

// EnregistrerRapportIntervention couvre la phase 4.5 (rapport de fin
// d'intervention du prestataire) puis la phase 5.0 (choix du mode de
// vérification).
//
// Choix du mode de vérification simplifié pour cette tranche : une règle
// déterministe (urgence faible → jugée inutile, sinon confirmation
// demandée à l'occupant) plutôt qu'un appel à Claude — à raffiner dans un
// prochain jalon (cf. doc, phase 5.0.1, qui prévoit aussi une vérification
// humaine sur place pour les cas incertains, pas encore implémentée).
func (s *IncidentService) EnregistrerRapportIntervention(ctx context.Context, ticketID int64, rapport string) error {
	if err := s.Repo.SetIncidentRapportIntervention(ctx, ticketID, rapport); err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: %w", ticketID, err)
	}

	incident, err := s.Repo.FindIncidentByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: chargement de l'incident: %w", ticketID, err)
	}
	if incident == nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: incident introuvable", ticketID)
	}

	jugeeBenigne := incident.UrgenceID != nil
	if jugeeBenigne {
		faibleID, err := s.Repo.NiveauUrgenceID(ctx, domain.NiveauUrgenceFaible)
		if err != nil {
			return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du niveau faible: %w", ticketID, err)
		}
		jugeeBenigne = *incident.UrgenceID == faibleID
	}

	if jugeeBenigne {
		modeID, err := s.Repo.ModeVerificationID(ctx, domain.ModeVerificationJugeeInutile)
		if err != nil {
			return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du mode jugée inutile: %w", ticketID, err)
		}
		positiveID, err := s.Repo.VerificationResultatID(ctx, domain.VerificationResultatPositive)
		if err != nil {
			return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du résultat positif: %w", ticketID, err)
		}
		if err := s.Repo.SetIncidentModeVerification(ctx, ticketID, modeID, &positiveID); err != nil {
			return fmt.Errorf("service: rapport d'intervention ticket_id=%d: enregistrement du mode de vérification: %w", ticketID, err)
		}
		maintenant := time.Now().UTC()
		return s.Repo.SetIncidentVerificationResultat(ctx, ticketID, positiveID, &maintenant)
	}

	modeID, err := s.Repo.ModeVerificationID(ctx, domain.ModeVerificationConfirmationOccupant)
	if err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du mode confirmation occupant: %w", ticketID, err)
	}
	if err := s.Repo.SetIncidentModeVerification(ctx, ticketID, modeID, nil); err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: enregistrement du mode de vérification: %w", ticketID, err)
	}
	statutAttenteEmetteurID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteEmetteur)
	if err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du statut en_attente_emetteur: %w", ticketID, err)
	}
	return s.Repo.UpdateTicketStatut(ctx, ticketID, statutAttenteEmetteurID)
}

// EnregistrerConfirmationOccupant couvre la phase 5.1.3/5.1.5 : la voie
// confirmation de l'occupant. Une non-résolution laisse le ticket en
// attente d'un gestionnaire humain — la réclamation auprès du prestataire
// (phase 5.3) n'est pas encore implémentée dans cette tranche.
func (s *IncidentService) EnregistrerConfirmationOccupant(ctx context.Context, ticketID int64, resolu bool) error {
	resultatDescription := domain.VerificationResultatNegative
	if resolu {
		resultatDescription = domain.VerificationResultatPositive
	}
	resultatID, err := s.Repo.VerificationResultatID(ctx, resultatDescription)
	if err != nil {
		return fmt.Errorf("service: confirmation occupant ticket_id=%d: résolution du résultat: %w", ticketID, err)
	}

	var dateResolution *time.Time
	if resolu {
		maintenant := time.Now().UTC()
		dateResolution = &maintenant
	}
	if err := s.Repo.SetIncidentVerificationResultat(ctx, ticketID, resultatID, dateResolution); err != nil {
		return fmt.Errorf("service: confirmation occupant ticket_id=%d: %w", ticketID, err)
	}

	if resolu {
		// Vérification positive : la mise en paiement de la facture devient
		// possible (cf. MettreEnPaiement), mais rien à faire ici sur le
		// statut — la facture suit son propre circuit (cf. EnregistrerFacture).
		return nil
	}

	// Non-résolution (phase 5.1.5) : réclamation auprès du prestataire
	// (phase 5.3) pas encore implémentée — un gestionnaire humain reprend
	// la main.
	statutAttenteGestionnaireID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteGestionnaire)
	if err != nil {
		return fmt.Errorf("service: confirmation occupant ticket_id=%d: résolution du statut en_attente_gestionnaire: %w", ticketID, err)
	}
	return s.Repo.UpdateTicketStatut(ctx, ticketID, statutAttenteGestionnaireID)
}

// EnregistrerFacture couvre la phase 5.5.1-5.5.3 (réception et extraction de
// la facture du prestataire).
func (s *IncidentService) EnregistrerFacture(ctx context.Context, ticketID, prestataireID, montantFactureCentimes int64, dateReception time.Time, creePar *int64) (*domain.Facture, error) {
	statutRecueID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutRecue)
	if err != nil {
		return nil, fmt.Errorf("service: enregistrement facture ticket_id=%d: résolution du statut recue: %w", ticketID, err)
	}
	facture, err := s.Repo.InsertFacture(ctx, &domain.Facture{
		TicketID:               ticketID,
		PrestataireID:          prestataireID,
		MontantFactureCentimes: montantFactureCentimes,
		StatutID:               statutRecueID,
		DateReception:          &dateReception,
		CreePar:                creePar,
	})
	if err != nil {
		return nil, fmt.Errorf("service: enregistrement facture ticket_id=%d: %w", ticketID, err)
	}
	return facture, nil
}

// ValiderFacture couvre la phase 5.5.4 : validation du montant facturé face
// au devis ou au contrat.
func (s *IncidentService) ValiderFacture(ctx context.Context, factureID, validePar int64) error {
	statutValideeID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutValidee)
	if err != nil {
		return fmt.Errorf("service: validation facture id=%d: résolution du statut validee: %w", factureID, err)
	}
	if err := s.Repo.ValiderFacture(ctx, factureID, statutValideeID, validePar, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: validation facture id=%d: %w", factureID, err)
	}
	return nil
}

// MettreEnPaiement couvre la phase 5.5.5 : mise en paiement, gatée sur la
// facture validée ET la vérification positive de l'incident — un travail
// mal exécuté ne doit pas être payé (exception d'inexécution, cf.
// docs/cycle-vie-incident.md §5.5).
func (s *IncidentService) MettreEnPaiement(ctx context.Context, ticketID, payePar int64) error {
	incident, err := s.Repo.FindIncidentByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: chargement de l'incident: %w", ticketID, err)
	}
	if incident == nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: incident introuvable", ticketID)
	}
	positiveID, err := s.Repo.VerificationResultatID(ctx, domain.VerificationResultatPositive)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: résolution du résultat positif: %w", ticketID, err)
	}
	if incident.VerificationResultatID == nil || *incident.VerificationResultatID != positiveID {
		return fmt.Errorf("service: mise en paiement ticket_id=%d refusée : la vérification n'est pas positive (exception d'inexécution)", ticketID)
	}

	facture, err := s.Repo.FindFactureByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: chargement de la facture: %w", ticketID, err)
	}
	if facture == nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d refusée : aucune facture enregistrée", ticketID)
	}
	statutValideeID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutValidee)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: résolution du statut validee: %w", ticketID, err)
	}
	if facture.StatutID != statutValideeID {
		return fmt.Errorf("service: mise en paiement ticket_id=%d refusée : la facture n'est pas validée", ticketID)
	}

	statutPayeeID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutPayee)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: résolution du statut payee: %w", ticketID, err)
	}
	if err := s.Repo.MettreFactureEnPaiement(ctx, facture.ID, statutPayeeID, payePar, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: %w", ticketID, err)
	}
	return nil
}

// RapprocherFacture couvre la phase 5.5.6 (rapprochement comptable) puis la
// phase 5.6.1 (clôture : statut "résolu"). La clôture définitive après
// période de grâce (5.6.2) et l'archivage (5.6.3) ne sont pas couverts ici :
// ils supposent une automatisation périodique, hors de cette tranche (cf.
// doc de IncidentService).
func (s *IncidentService) RapprocherFacture(ctx context.Context, ticketID, factureID, rapprochePar int64) error {
	statutRapprocheeID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutRapprochee)
	if err != nil {
		return fmt.Errorf("service: rapprochement facture id=%d: résolution du statut rapprochee: %w", factureID, err)
	}
	if err := s.Repo.RapprocherFacture(ctx, factureID, statutRapprocheeID, rapprochePar, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: rapprochement facture id=%d: %w", factureID, err)
	}

	statutResoluID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutResolu)
	if err != nil {
		return fmt.Errorf("service: rapprochement facture id=%d: résolution du statut résolu: %w", factureID, err)
	}
	if err := s.Repo.UpdateTicketStatut(ctx, ticketID, statutResoluID); err != nil {
		return fmt.Errorf("service: rapprochement facture id=%d: clôture du ticket_id=%d: %w", factureID, ticketID, err)
	}
	return nil
}

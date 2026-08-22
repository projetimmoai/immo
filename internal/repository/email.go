package repository

import (
	"context"
	"fmt"
	"net/http"

	"github.com/projetimmoai/immo/internal/domain"
)

// emailRow est la représentation JSON d'une ligne de la table email (table
// détail, cf. domain.Email — le squelette commun vit sur TicketSource).
type emailRow struct {
	TicketSourceID  int64   `json:"ticket_source_id"`
	MessageID       *string `json:"message_id"`
	ExpediteurEmail string  `json:"expediteur_email"`
	Objet           *string `json:"objet"`
	CorpsTexte      *string `json:"corps_texte"`
	CorpsHTML       *string `json:"corps_html"`
}

func (r emailRow) toDomain() *domain.Email {
	return &domain.Email{
		TicketSourceID:  r.TicketSourceID,
		MessageID:       r.MessageID,
		ExpediteurEmail: r.ExpediteurEmail,
		Objet:           r.Objet,
		CorpsTexte:      r.CorpsTexte,
		CorpsHTML:       r.CorpsHTML,
	}
}

// emailInsert est la charge utile d'insertion d'un Email : uniquement les
// colonnes fournies par l'appelant.
type emailInsert struct {
	TicketSourceID  int64   `json:"ticket_source_id"`
	MessageID       *string `json:"message_id,omitempty"`
	ExpediteurEmail string  `json:"expediteur_email"`
	Objet           *string `json:"objet,omitempty"`
	CorpsTexte      *string `json:"corps_texte,omitempty"`
	CorpsHTML       *string `json:"corps_html,omitempty"`
}

// InsertEmail insère une nouvelle TicketSource (type "email") et sa table
// détail Email, et retourne les deux lignes créées. source.TypeID est
// ignoré et toujours résolu ici (l'appelant ne le fournit pas).
//
// Deux écritures liées (ticket_source puis email) via deux appels REST
// séquentiels plutôt qu'une transaction : conforme à CLAUDE.md pour le cas
// général (RPC plpgsql) uniquement quand un vrai appelant a besoin
// d'atomicité — aucun ne l'exige encore ici (worker-email ne persiste pas
// encore, cf. cmd/worker-email). À revoir en RPC si un échec partiel
// (ticket_source créée sans son email) devient un risque réel une fois ce
// chemin d'écriture branché.
func (c *Client) InsertEmail(ctx context.Context, source *domain.TicketSource, e *domain.Email) (*domain.TicketSource, *domain.Email, error) {
	typeID, err := c.TicketSourceTypeID(ctx, domain.TicketSourceTypeEmail)
	if err != nil {
		return nil, nil, fmt.Errorf("repository: insertion email: résolution du type de source: %w", err)
	}
	source.TypeID = typeID

	createdSource, err := c.InsertTicketSource(ctx, source)
	if err != nil {
		return nil, nil, fmt.Errorf("repository: insertion email (expediteur=%s): %w", e.ExpediteurEmail, err)
	}

	payload := []emailInsert{{
		TicketSourceID:  createdSource.ID,
		MessageID:       e.MessageID,
		ExpediteurEmail: e.ExpediteurEmail,
		Objet:           e.Objet,
		CorpsTexte:      e.CorpsTexte,
		CorpsHTML:       e.CorpsHTML,
	}}
	var rows []emailRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/email", payload, "return=representation", &rows); err != nil {
		return nil, nil, fmt.Errorf("repository: insertion email (expediteur=%s): %w", e.ExpediteurEmail, err)
	}
	if len(rows) != 1 {
		return nil, nil, fmt.Errorf("repository: insertion email (expediteur=%s): %d ligne(s) retournée(s), 1 attendue", e.ExpediteurEmail, len(rows))
	}
	return createdSource, rows[0].toDomain(), nil
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

// DeleteEmail supprime la ligne détail Email par son TicketSourceID
// (utilisé notamment par les tests d'intégration pour nettoyer après eux).
// Ne supprime pas la TicketSource elle-même — cf. DeleteTicketSource, qui
// supprime les deux (ON DELETE CASCADE).
func (c *Client) DeleteEmail(ctx context.Context, ticketSourceID int64) error {
	path := fmt.Sprintf("/email?ticket_source_id=eq.%d", ticketSourceID)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("repository: suppression email ticket_source_id=%d: %w", ticketSourceID, err)
	}
	return nil
}

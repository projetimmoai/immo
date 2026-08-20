package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// logRow est la représentation JSON d'une ligne de la table log.
type logRow struct {
	ID            int64     `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	LogTypeID     int64     `json:"log_type_id"`
	Message       *string   `json:"message"`
	EmailID       *int64    `json:"email_id"`
	CoproprieteID *int64    `json:"copropriete_id"`
	PersonneID    *int64    `json:"personne_id"`
}

func (r logRow) toDomain() *domain.Log {
	return &domain.Log{
		ID:            r.ID,
		CreatedAt:     r.CreatedAt,
		LogTypeID:     r.LogTypeID,
		Message:       r.Message,
		EmailID:       r.EmailID,
		CoproprieteID: r.CoproprieteID,
		PersonneID:    r.PersonneID,
	}
}

// logInsert est la charge utile d'insertion d'un Log : uniquement les
// colonnes fournies par l'appelant (id/created_at sont générés par la base).
type logInsert struct {
	LogTypeID     int64   `json:"log_type_id"`
	Message       *string `json:"message,omitempty"`
	EmailID       *int64  `json:"email_id,omitempty"`
	CoproprieteID *int64  `json:"copropriete_id,omitempty"`
	PersonneID    *int64  `json:"personne_id,omitempty"`
}

// InsertLog consigne un évènement dans log. l.LogTypeID doit être renseigné
// explicitement par l'appelant (pas de valeur par défaut en base) — cf.
// LogTypeID pour le résoudre à partir d'une description (voir les
// constantes domain.LogType*).
func (c *Client) InsertLog(ctx context.Context, l *domain.Log) (*domain.Log, error) {
	payload := []logInsert{{
		LogTypeID:     l.LogTypeID,
		Message:       l.Message,
		EmailID:       l.EmailID,
		CoproprieteID: l.CoproprieteID,
		PersonneID:    l.PersonneID,
	}}
	var rows []logRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/log", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion log (log_type_id=%d): %w", l.LogTypeID, err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion log: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// DeleteLog supprime un Log par son ID (utilisé notamment par les tests
// d'intégration pour nettoyer après eux).
func (c *Client) DeleteLog(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/log?id=eq.%d", id)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("repository: suppression log id=%d: %w", id, err)
	}
	return nil
}

// Package claudeapi est la couche d'accès à l'API Claude (Anthropic),
// utilisée pour la classification des e-mails (une analyse plus fine que
// l'heuristique mots-clés de internal/email.DecideRoute). Contrairement à
// internal/repository (Supabase) et internal/drive/internal/gmailapi
// (Google), Anthropic fournit un SDK Go officiel — inutile de refaire du
// REST à la main ici.
package claudeapi

import (
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

// Model est le modèle Claude utilisé pour la classification des e-mails :
// Haiku 4.5, suffisant pour une tâche répétitive et peu coûteuse — pas
// besoin d'un modèle plus capable (et plus cher) comme Opus pour ça.
const Model = anthropic.ModelClaudeHaiku4_5

// Client est le client d'accès à l'API Claude.
type Client struct {
	anthropic anthropic.Client
}

// NewClientFromEnv construit un Client à partir de la variable
// d'environnement ANTHROPIC_API_KEY (lue automatiquement par le SDK).
func NewClientFromEnv() (*Client, error) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return nil, fmt.Errorf("claudeapi: variable d'environnement ANTHROPIC_API_KEY manquante")
	}
	return &Client{anthropic: anthropic.NewClient()}, nil
}

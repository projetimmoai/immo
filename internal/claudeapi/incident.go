package claudeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/projetimmoai/immo/internal/domain"
)

// qualificationIncidentTool est le nom de l'outil forcé utilisé pour obtenir
// une réponse structurée de Claude (cf. decisionActionTool/
// decisionCoproprieteTool pour le même principe).
const qualificationIncidentTool = "qualifier_incident"

// categorieIndeterminee est la valeur sentinelle retournée par Claude pour
// "categorie_technique" quand aucune des catégories fournies ne correspond
// (cf. roleIndetermine, même contrainte : un enum JSON Schema ne peut pas
// mélanger null avec des valeurs string).
const categorieIndeterminee = "indeterminee"

// QualificationIncident est la réponse structurée de Claude à
// QualifierIncident (phase 2 du graphe de cycle de vie, cf.
// docs/cycle-vie-incident.md) : une estimation indicative, jamais un
// engagement — elle sert seulement à orienter la décision de traitement
// (comparaison aux seuils légaux, phase 3).
type QualificationIncident struct {
	CategorieTechnique    *string // description parmi celles fournies ; nil si indéterminable
	Urgence               string  // description parmi celles fournies (cf. domain.NiveauUrgence*) — toujours renseignée, "faible" par défaut en cas de doute
	MontantEstimeCentimes *int64  // estimation indicative en centimes ; nil si non estimable à partir du texte
	Confiance             float64 // entre 0 (aucune confiance) et 1 (certitude)
	Raison                string
}

// QualifierIncident demande à Claude, à partir du contenu d'un signalement
// déjà classé comme incident, d'en extraire la catégorie technique (parmi le
// catalogue categories), le niveau d'urgence perçu (parmi urgences) et une
// estimation indicative du coût — phase 2.1/2.2 du graphe.
func (c *Client) QualifierIncident(ctx context.Context, categories []domain.CategorieTechnique, urgences []domain.NiveauUrgence, objet, corpsTexte string) (QualificationIncident, error) {
	categorieDescriptions := make([]string, 0, len(categories)+1)
	categorieDescriptions = append(categorieDescriptions, categorieIndeterminee)
	for _, cat := range categories {
		if cat.Description != nil {
			categorieDescriptions = append(categorieDescriptions, *cat.Description)
		}
	}
	urgenceDescriptions := make([]string, len(urgences))
	for i, u := range urgences {
		urgenceDescriptions[i] = u.Description
	}

	tool := anthropic.ToolParam{
		Name:        qualificationIncidentTool,
		Description: anthropic.String("Enregistre la qualification (catégorie technique, urgence, coût estimé) d'un incident signalé."),
		Strict:      anthropic.Bool(true),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"categorie_technique": map[string]any{
					"type":        "string",
					"enum":        categorieDescriptions,
					"description": "Le domaine technique qui décrit le mieux la panne, parmi les catégories fournies, ou \"" + categorieIndeterminee + "\" si aucune ne correspond.",
				},
				"urgence": map[string]any{
					"type":        "string",
					"enum":        urgenceDescriptions,
					"description": "Le niveau d'urgence perçu à la lecture du signalement, parmi les niveaux fournis.",
				},
				"montant_estime_euros": map[string]any{
					"type":        []string{"integer", "null"},
					"description": "Estimation indicative du coût de la réparation en euros (nombre entier, jamais un montant engageant) ; null si le texte ne permet aucune estimation raisonnable.",
				},
				"confiance": map[string]any{
					"type":        "number",
					"description": "Indice de confiance dans la qualification, un nombre entre 0 (aucune confiance) et 1 (certitude) inclus.",
				},
				"raison": map[string]any{
					"type":        "string",
					"description": "Explication brève (une phrase) de la qualification.",
				},
			},
			Required:    []string{"categorie_technique", "urgence", "montant_estime_euros", "confiance", "raison"},
			ExtraFields: map[string]any{"additionalProperties": false},
		},
	}

	resp, err := c.anthropic.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     Model,
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{{
			Text: "Tu aides un cabinet de gestion de copropriétés à qualifier les incidents techniques " +
				"signalés (panne, dysfonctionnement...). On te donne le contenu d'un signalement déjà " +
				"classé comme incident, le catalogue des catégories techniques possibles et les niveaux " +
				"d'urgence possibles. Détermine la catégorie technique qui décrit le mieux la panne, le " +
				"niveau d'urgence perçu, et une estimation très approximative du coût de réparation en " +
				"euros si le texte donne assez d'indices (surface, ampleur, matériel visiblement en " +
				"cause) — sinon renvoie null plutôt que d'inventer un chiffre. Reflète toujours ton " +
				"incertitude dans confiance.",
		}},
		Tools:      []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{OfTool: &anthropic.ToolChoiceToolParam{Name: qualificationIncidentTool}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptQualificationIncident(categorieDescriptions, urgenceDescriptions, objet, corpsTexte))),
		},
	})
	if err != nil {
		return QualificationIncident{}, fmt.Errorf("claudeapi: appel de qualification d'incident: %w", err)
	}

	for _, block := range resp.Content {
		toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok || toolUse.Name != qualificationIncidentTool {
			continue
		}
		var out struct {
			CategorieTechnique string  `json:"categorie_technique"`
			Urgence            string  `json:"urgence"`
			MontantEstimeEuros *int64  `json:"montant_estime_euros"`
			Confiance          float64 `json:"confiance"`
			Raison             string  `json:"raison"`
		}
		if err := json.Unmarshal([]byte(toolUse.JSON.Input.Raw()), &out); err != nil {
			return QualificationIncident{}, fmt.Errorf("claudeapi: décodage de la qualification d'incident: %w", err)
		}
		q := QualificationIncident{Urgence: out.Urgence, Confiance: clamp01(out.Confiance), Raison: out.Raison}
		if out.CategorieTechnique != "" && out.CategorieTechnique != categorieIndeterminee {
			q.CategorieTechnique = &out.CategorieTechnique
		}
		if out.MontantEstimeEuros != nil {
			centimes := *out.MontantEstimeEuros * 100
			q.MontantEstimeCentimes = &centimes
		}
		return q, nil
	}

	return QualificationIncident{}, fmt.Errorf("claudeapi: aucune qualification d'incident retournée (stop_reason=%s)", resp.StopReason)
}

// promptQualificationIncident construit le message utilisateur envoyé à
// Claude : le catalogue disponible puis le contenu du signalement.
func promptQualificationIncident(categorieDescriptions, urgenceDescriptions []string, objet, corpsTexte string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Catégories techniques disponibles : %s\n", strings.Join(categorieDescriptions, ", "))
	fmt.Fprintf(&sb, "Niveaux d'urgence disponibles : %s\n", strings.Join(urgenceDescriptions, ", "))
	sb.WriteString("\nObjet du signalement : ")
	sb.WriteString(objet)
	sb.WriteString("\n\nContenu du signalement :\n")
	sb.WriteString(corpsTexte)
	return sb.String()
}

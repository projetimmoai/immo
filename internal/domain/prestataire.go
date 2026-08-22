package domain

import "time"

// PrestataireCategorieTechnique associe une Personne (personne_morale
// prestataire) à une catégorie technique qu'elle sait traiter —
// indépendamment de tout contrat particulier. C'est le catalogue général
// du répertoire de prestataires (cf. docs/cycle-vie-incident.md, phase
// 3.3.2) : un prestataire peut y figurer sans être encore sous contrat
// pour aucune copropriété (contrat associe, lui, un prestataire à UNE
// copropriété pour UNE catégorie).
type PrestataireCategorieTechnique struct {
	ID                   int64
	CreatedAt            time.Time
	PersonneID           int64 // FK -> personne.id, NOT NULL
	CategorieTechniqueID int64 // FK -> categorie_technique.id, NOT NULL
}

// PrestataireZoneIntervention associe une Personne (personne_morale
// prestataire) à une ville qu'elle couvre — indépendamment de tout contrat
// particulier. Une Personne peut couvrir plusieurs villes (plusieurs
// lignes).
type PrestataireZoneIntervention struct {
	ID         int64
	CreatedAt  time.Time
	PersonneID int64 // FK -> personne.id, NOT NULL
	Ville      string
}

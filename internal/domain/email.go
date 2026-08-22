package domain

import "time"

// Valeurs connues de action.description (voir migration 001). Ce sont des
// tables de référence éditables en base : ces constantes servent de clés de
// recherche (par description), jamais d'ID en dur.
const (
	ActionSinistre              = "sinistre"
	ActionIncident              = "incident"
	ActionAssembleeGenerale     = "assemblee_generale"
	ActionAutre                 = "autre"
	ActionComptabilite          = "comptabilite"
	ActionMutation              = "mutation"
	ActionContentieux           = "contentieux"
	ActionTravaux               = "travaux"
	ActionControleGestion       = "controle_gestion"
	ActionDemandeAdministrative = "demande_administrative"
)

// Valeurs connues de email_statut_traitement.description.
const (
	EmailStatutNouveau   = "nouveau"
	EmailStatutClassifie = "classifie"
	EmailStatutTraite    = "traite"
	EmailStatutIgnore    = "ignore"
	EmailStatutErreur    = "erreur"
)

// Action est la table de référence racine du routage des e-mails (sinistre,
// incident, AG, autre...), et plus largement de toute décision de routage.
type Action struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// SousAction est une sous-catégorie d'une Action (ex: "degat_des_eaux" sous
// "sinistre"), pouvant elle-même avoir des sous-sous-actions : une
// hiérarchie récursive de profondeur arbitraire. ActionID identifie
// toujours l'Action racine de la hiérarchie (dénormalisé, valable à
// n'importe quelle profondeur) ; ParentID est nul pour une sous_action de
// premier niveau (directement sous l'Action), et pointe sinon vers sa
// sous_action parente. La Description n'est unique que parmi les
// sous_action partageant le même (ActionID, ParentID) — cf. contrainte
// UNIQUE NULLS NOT DISTINCT en base.
type SousAction struct {
	ID          int64
	CreatedAt   time.Time
	Description string
	ActionID    int64  // FK -> action.id (action racine, à toute profondeur)
	ParentID    *int64 // FK -> sous_action.id, nul si sous_action de premier niveau
}

// EmailStatutTraitement est une table de référence pour le statut de
// traitement d'un Email par le worker (nouveau, classifie, traite...).
type EmailStatutTraitement struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Email est le journal de tout e-mail reçu par le worker, traité ou non.
//
// Pas d'ActionID/SousActionID ici : un même e-mail peut donner lieu à
// plusieurs demandes distinctes (cf. email.routerVersActions), donc à
// plusieurs Ticket — chacun porte sa propre action (cf. domain.Ticket).
// CoproprieteID/LotID restent ici : déterminés une seule fois par e-mail
// (cf. email.DetermineCopropriete), même si aucun ticket n'est encore créé.
type Email struct {
	ID                   int64
	CreatedAt            time.Time
	MessageID            *string // header RFC822 Message-ID, sert à dédupliquer
	DateReception        time.Time
	ExpediteurEmail      string
	ExpediteurPersonneID *int64 // FK -> personne.id, résolu si l'expéditeur est connu
	Objet                *string
	CorpsTexte           *string
	CorpsHTML            *string
	CoproprieteID        *int64 // FK -> copropriete.id, résolu si identifiable
	LotID                *int64 // FK -> lot.id, résolu si identifiable
	StatutTraitementID   int64  // FK -> email_statut_traitement.id, NOT NULL : à fixer explicitement à l'insertion (pas de DEFAULT en base)
	TraiteLe             *time.Time
	TraitePar            *int64 // FK -> personne.id (collaborateur)
	ErreurTraitement     *string
}

// EmailPieceJointe est une pièce jointe d'un Email, stockée dans Supabase
// Storage (cf. internal/storage, bucket storage.Bucket) et référencée ici
// par son chemin d'objet — pas une URL permanente : le fichier n'étant pas
// public, l'accès se fait via une URL signée temporaire générée à la
// demande (storage.Client.SignedURL), jamais stockée en base.
type EmailPieceJointe struct {
	ID             int64
	CreatedAt      time.Time
	EmailID        int64
	NomFichier     string
	TypeMime       *string
	TailleOctets   *int64
	CheminStockage *string // chemin de l'objet dans storage.Bucket
}

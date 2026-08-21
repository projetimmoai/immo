package repository

import (
	"fmt"
	"strings"
	"time"
)

// dateLayout est le format utilisé par PostgREST pour sérialiser une colonne
// SQL "date" (sans heure ni fuseau) — différent du format RFC 3339 utilisé
// pour les colonnes "timestamptz", que time.Time décode nativement.
const dateLayout = "2006-01-02"

// Date encapsule un time.Time représentant une colonne SQL "date" (par
// opposition à "timestamptz") : PostgREST la sérialise en "YYYY-MM-DD", que
// time.Time ne sait pas décoder nativement (il attend du RFC 3339) — sans
// ce type, le décodage JSON échoue dès que la colonne a une valeur non
// nulle (découvert en pratique sur contrat.date_debut).
type Date struct {
	time.Time
}

// MarshalJSON sérialise au format "YYYY-MM-DD" attendu par PostgREST pour
// une colonne "date".
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Format(dateLayout) + `"`), nil
}

// UnmarshalJSON décode une date PostgREST au format "YYYY-MM-DD".
func (d *Date) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		d.Time = time.Time{}
		return nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("repository: décodage de la date %q: %w", s, err)
	}
	d.Time = t
	return nil
}

// dateToTimePtr convertit un *Date (lu depuis PostgREST) en *time.Time pour
// le domaine (internal/domain), qui reste un time.Time simple — il n'a pas
// à connaître le détail du format d'échange PostgREST pour les colonnes
// "date".
func dateToTimePtr(d *Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time
	return &t
}

// timePtrToDate convertit un *time.Time du domaine en *Date pour l'envoyer
// à PostgREST au format "YYYY-MM-DD" attendu par une colonne SQL "date".
func timePtrToDate(t *time.Time) *Date {
	if t == nil {
		return nil
	}
	return &Date{Time: *t}
}

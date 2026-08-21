package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

// fakeRepo permet de tester CreateCopropriete sans dépendre de Supabase.
type fakeRepo struct {
	insertErr error

	nextID int64
}

func (f *fakeRepo) InsertCopropriete(_ context.Context, cop *domain.Copropriete) (*domain.Copropriete, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	f.nextID = 42
	reference := "COP42"
	return &domain.Copropriete{ID: f.nextID, Nom: cop.Nom, Reference: reference}, nil
}

func validInput() CreateCoproprieteInput {
	return CreateCoproprieteInput{Nom: "Residence Test", CreeParID: 1}
}

func TestCreateCoproprieteSucces(t *testing.T) {
	repo := &fakeRepo{}
	s := &CoproprieteService{Repo: repo}

	created, err := s.CreateCopropriete(context.Background(), validInput())
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if created.ID != 42 || created.Reference != "COP42" {
		t.Errorf("copropriete créée inattendue: %+v", created)
	}
}

func TestCreateCoproprieteEchecBaseDeDonnees(t *testing.T) {
	wantErr := errors.New("connexion refusée")
	repo := &fakeRepo{insertErr: wantErr}
	s := &CoproprieteService{Repo: repo}

	created, err := s.CreateCopropriete(context.Background(), validInput())
	if created != nil {
		t.Errorf("attendu nil, obtenu %+v", created)
	}
	if err == nil {
		t.Fatal("attendu une erreur, obtenu nil")
	}
	if !strings.Contains(err.Error(), "base de données") {
		t.Errorf("erreur ne mentionne pas la base de données: %v", err)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("erreur d'origine perdue: %v", err)
	}
}

func TestCreateCoproprieteNomManquant(t *testing.T) {
	s := &CoproprieteService{Repo: &fakeRepo{}}
	in := validInput()
	in.Nom = ""

	if _, err := s.CreateCopropriete(context.Background(), in); err == nil {
		t.Fatal("attendu une erreur (nom obligatoire), obtenu nil")
	}
}

func TestCreateCoproprieteCreeParManquant(t *testing.T) {
	s := &CoproprieteService{Repo: &fakeRepo{}}
	in := validInput()
	in.CreeParID = 0

	if _, err := s.CreateCopropriete(context.Background(), in); err == nil {
		t.Fatal("attendu une erreur (cree_par obligatoire), obtenu nil")
	}
}

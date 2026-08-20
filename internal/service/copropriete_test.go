package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

// fakeRepo et fakeDrive permettent de tester la logique de compensation de
// CreateCopropriete sans dépendre de Supabase ni de Google Drive.

type fakeRepo struct {
	insertErr error
	deleteErr error

	nextID    int64
	deletedID int64 // dernier ID passé à DeleteCopropriete, 0 si jamais appelé
}

func (f *fakeRepo) InsertCopropriete(_ context.Context, cop *domain.Copropriete) (*domain.Copropriete, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	f.nextID = 42
	reference := "COP42"
	return &domain.Copropriete{ID: f.nextID, Nom: cop.Nom, Reference: reference}, nil
}

func (f *fakeRepo) DeleteCopropriete(_ context.Context, id int64) error {
	f.deletedID = id
	return f.deleteErr
}

type fakeDrive struct {
	err error
}

func (f *fakeDrive) CreateCoproprieteFolders(_ context.Context, _, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "drive-folder-id", nil
}

func validInput() CreateCoproprieteInput {
	return CreateCoproprieteInput{Nom: "Residence Test", CreeParID: 1}
}

func TestCreateCoproprieteSucces(t *testing.T) {
	repo := &fakeRepo{}
	drv := &fakeDrive{}
	s := &CoproprieteService{Repo: repo, Drive: drv}

	created, err := s.CreateCopropriete(context.Background(), validInput())
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if created.ID != 42 || created.Reference != "COP42" {
		t.Errorf("copropriete créée inattendue: %+v", created)
	}
	if repo.deletedID != 0 {
		t.Errorf("DeleteCopropriete n'aurait pas dû être appelé, appelé avec id=%d", repo.deletedID)
	}
}

func TestCreateCoproprieteEchecBaseDeDonnees(t *testing.T) {
	wantErr := errors.New("connexion refusée")
	repo := &fakeRepo{insertErr: wantErr}
	drv := &fakeDrive{}
	s := &CoproprieteService{Repo: repo, Drive: drv}

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

func TestCreateCoproprieteEchecDriveAvecRollbackReussi(t *testing.T) {
	driveErr := errors.New("quota Drive dépassé")
	repo := &fakeRepo{}
	drv := &fakeDrive{err: driveErr}
	s := &CoproprieteService{Repo: repo, Drive: drv}

	created, err := s.CreateCopropriete(context.Background(), validInput())
	if created != nil {
		t.Errorf("attendu nil (rien ne doit rester créé), obtenu %+v", created)
	}
	if err == nil {
		t.Fatal("attendu une erreur, obtenu nil")
	}
	if !strings.Contains(err.Error(), "Drive") {
		t.Errorf("erreur ne mentionne pas Drive: %v", err)
	}
	if !errors.Is(err, driveErr) {
		t.Errorf("erreur d'origine perdue: %v", err)
	}
	if repo.deletedID != 42 {
		t.Errorf("le rollback DeleteCopropriete aurait dû être appelé avec id=42, deletedID=%d", repo.deletedID)
	}
}

func TestCreateCoproprieteEchecDriveEtEchecRollback(t *testing.T) {
	driveErr := errors.New("quota Drive dépassé")
	deleteErr := errors.New("Supabase injoignable")
	repo := &fakeRepo{deleteErr: deleteErr}
	drv := &fakeDrive{err: driveErr}
	s := &CoproprieteService{Repo: repo, Drive: drv}

	created, err := s.CreateCopropriete(context.Background(), validInput())
	// Cas limite documenté : la ligne DB existe réellement (rollback en
	// échec), donc on la retourne pour que l'appelant puisse la corriger.
	if created == nil || created.ID != 42 {
		t.Errorf("attendu la copropriete orpheline (id=42) pour permettre une correction manuelle, obtenu %+v", created)
	}
	if err == nil {
		t.Fatal("attendu une erreur, obtenu nil")
	}
	for _, want := range []string{"id=42", "manuellement"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("erreur ne mentionne pas %q: %v", want, err)
		}
	}
	if !errors.Is(err, driveErr) {
		t.Errorf("erreur Drive d'origine perdue: %v", err)
	}
}

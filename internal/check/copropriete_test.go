package check

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/drive"
)

type fakeRepo struct {
	coproprietes []*domain.Copropriete
}

func (f *fakeRepo) ListCopropriete(_ context.Context) ([]*domain.Copropriete, error) {
	return f.coproprietes, nil
}

type fakeDrive struct {
	rootID  string
	folders map[string][]drive.Folder // parentID -> enfants
}

func (f *fakeDrive) RootFolderID() string { return f.rootID }

func (f *fakeDrive) ListChildFolders(_ context.Context, parentID string) ([]drive.Folder, error) {
	return f.folders[parentID], nil
}

func coproprieteTest(id int64, reference string) *domain.Copropriete {
	nom := "Nom Test"
	return &domain.Copropriete{ID: id, Reference: reference, Nom: &nom}
}

func categoriesCompletes() []drive.Folder {
	var folders []drive.Folder
	for _, cat := range drive.CoproprieteSousDossiers {
		folders = append(folders, drive.Folder{ID: cat + "-id", Name: cat})
	}
	return folders
}

func TestCheckCoproprietesCoherent(t *testing.T) {
	repo := &fakeRepo{coproprietes: []*domain.Copropriete{coproprieteTest(1, "COP1")}}
	drv := &fakeDrive{
		rootID: "root",
		folders: map[string][]drive.Folder{
			"root":        {{ID: "cop1-folder", Name: "COP1 - Nom Test"}, {ID: "cabinet-id", Name: "_cabinet"}},
			"cop1-folder": categoriesCompletes(),
		},
	}

	anomalies, err := CheckCoproprietes(context.Background(), repo, drv)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if len(anomalies) != 0 {
		t.Errorf("attendu aucune anomalie, obtenu %+v", anomalies)
	}
}

func TestCheckCoproprietesDossierManquant(t *testing.T) {
	repo := &fakeRepo{coproprietes: []*domain.Copropriete{coproprieteTest(1, "COP1")}}
	drv := &fakeDrive{
		rootID:  "root",
		folders: map[string][]drive.Folder{"root": {}},
	}

	anomalies, err := CheckCoproprietes(context.Background(), repo, drv)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if len(anomalies) != 1 || anomalies[0].Categorie != CoproprieteDossierManquant {
		t.Fatalf("attendu 1 anomalie CoproprieteDossierManquant, obtenu %+v", anomalies)
	}
	if anomalies[0].CoproprieteID == nil || *anomalies[0].CoproprieteID != 1 {
		t.Errorf("CoproprieteID attendu 1, obtenu %v", anomalies[0].CoproprieteID)
	}
}

func TestCheckCoproprietesDossierAmbigu(t *testing.T) {
	repo := &fakeRepo{coproprietes: []*domain.Copropriete{coproprieteTest(1, "COP1")}}
	drv := &fakeDrive{
		rootID: "root",
		folders: map[string][]drive.Folder{
			"root": {
				{ID: "a", Name: "COP1 - Nom Test"},
				{ID: "b", Name: "COP1 - Nom Test (copie)"},
			},
		},
	}

	anomalies, err := CheckCoproprietes(context.Background(), repo, drv)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if len(anomalies) != 1 || anomalies[0].Categorie != CoproprieteDossierAmbigu {
		t.Fatalf("attendu 1 anomalie CoproprieteDossierAmbigu, obtenu %+v", anomalies)
	}
}

func TestCheckCoproprietesSousDossierManquant(t *testing.T) {
	repo := &fakeRepo{coproprietes: []*domain.Copropriete{coproprieteTest(1, "COP1")}}
	// Toutes les catégories sauf "sinistres".
	var incomplet []drive.Folder
	for _, cat := range drive.CoproprieteSousDossiers {
		if cat == "sinistres" {
			continue
		}
		incomplet = append(incomplet, drive.Folder{ID: cat + "-id", Name: cat})
	}
	drv := &fakeDrive{
		rootID: "root",
		folders: map[string][]drive.Folder{
			"root":        {{ID: "cop1-folder", Name: "COP1 - Nom Test"}},
			"cop1-folder": incomplet,
		},
	}

	anomalies, err := CheckCoproprietes(context.Background(), repo, drv)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if len(anomalies) != 1 || anomalies[0].Categorie != CoproprieteSousDossierManquant {
		t.Fatalf("attendu 1 anomalie CoproprieteSousDossierManquant, obtenu %+v", anomalies)
	}
}

func TestCheckCoproprietesDossierOrphelin(t *testing.T) {
	repo := &fakeRepo{} // aucune copropriete en base
	drv := &fakeDrive{
		rootID: "root",
		folders: map[string][]drive.Folder{
			"root": {{ID: "orphan", Name: "COP99 - Copropriete Supprimee"}},
		},
	}

	anomalies, err := CheckCoproprietes(context.Background(), repo, drv)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if len(anomalies) != 1 || anomalies[0].Categorie != DossierCoproprieteOrphelin {
		t.Fatalf("attendu 1 anomalie DossierCoproprieteOrphelin, obtenu %+v", anomalies)
	}
}

func TestCheckCoproprietesIgnoreCabinet(t *testing.T) {
	repo := &fakeRepo{} // aucune copropriete en base
	drv := &fakeDrive{
		rootID: "root",
		folders: map[string][]drive.Folder{
			"root": {{ID: "cabinet-id", Name: "_cabinet"}},
		},
	}

	anomalies, err := CheckCoproprietes(context.Background(), repo, drv)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if len(anomalies) != 0 {
		t.Errorf("\"_cabinet\" ne devrait jamais être signalé comme orphelin, obtenu %+v", anomalies)
	}
}

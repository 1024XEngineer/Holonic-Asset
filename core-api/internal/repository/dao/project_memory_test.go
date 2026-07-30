package dao_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

func TestMemoryProjectDaoUpdateAppliesOnlyProvidedFields(t *testing.T) {
	projectDao := dao.NewMemoryProjectDao()
	projectID, err := projectDao.CreateProject(context.Background(), &dao.Project{
		UserID:      7,
		Name:        "Prototype",
		Description: "keep this value",
		Reference:   "old-reference",
		Style:       "pixel",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	newReference := "new-reference"
	if err := projectDao.Update(context.Background(), &dao.ProjectUpdate{
		ID:        projectID,
		Reference: &newReference,
	}); err != nil {
		t.Fatalf("update project: %v", err)
	}

	project, err := projectDao.FindByID(context.Background(), projectID)
	if err != nil {
		t.Fatalf("find project: %v", err)
	}
	if project.Reference != "new-reference" {
		t.Fatalf("expected new reference, got %q", project.Reference)
	}
	if project.UserID != 7 || project.Name != "Prototype" || project.Description != "keep this value" || project.Style != "pixel" {
		t.Fatalf("partial update changed omitted fields: %+v", project)
	}
}

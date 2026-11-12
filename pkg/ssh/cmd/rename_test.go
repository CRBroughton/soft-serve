package cmd

import (
	"errors"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestRenameRequiresRepoAdmin checks that a read-write collaborator cannot
// rename a repository (which breaks every clone URL), while a repo admin can.
func TestRenameRequiresRepoAdmin(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)

	ownerCtx := withUser(t, ctx, be, "owner", false)
	owner := proto.UserFromContext(ownerCtx)
	_, err := be.CreateRepository(ownerCtx, "repo", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	collabCtx := withUser(t, ctx, be, "collab", false)
	is.NoErr(be.AddCollaborator(ownerCtx, "repo", "collab", access.ReadWriteAccess))

	// A read-write collaborator must not be able to rename the repo.
	err = runRepo(t, collabCtx, "rename", "repo", "renamed")
	if !errors.Is(err, proto.ErrUnauthorized) {
		t.Fatalf("collab rename: expected ErrUnauthorized, got %v", err)
	}
	_, err = be.Repository(ctx, "repo")
	is.NoErr(err) // repo must keep its name

	// An admin-access collaborator can rename the repo.
	adminCtx := withUser(t, ctx, be, "admin-collab", false)
	is.NoErr(be.AddCollaborator(ownerCtx, "repo", "admin-collab", access.AdminAccess))
	is.NoErr(runRepo(t, adminCtx, "rename", "repo", "renamed"))
	_, err = be.Repository(ctx, "renamed")
	is.NoErr(err)

	// So can the owner.
	is.NoErr(runRepo(t, ownerCtx, "rename", "renamed", "repo"))
	_, err = be.Repository(ctx, "repo")
	is.NoErr(err)
}

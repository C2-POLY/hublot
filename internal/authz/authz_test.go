// SPDX-License-Identifier: AGPL-3.0-or-later

package authz

import (
	"testing"

	"github.com/google/uuid"
)

var (
	shop  = Project{ID: uuid.MustParse("00000000-0000-0000-0000-00000000000a")}
	app   = Project{ID: uuid.MustParse("00000000-0000-0000-0000-00000000000b")}
	open  = Project{ID: uuid.MustParse("00000000-0000-0000-0000-00000000000c"), ReportersSeeAll: true}
	marie = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	paul  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

func member(roles map[uuid.UUID]Role) Subject {
	return Subject{UserID: marie, Org: OrgMember, Roles: roles}
}

func req(p Project, author uuid.UUID, private bool) Request {
	return Request{Project: p, OriginProject: p.ID, AuthorID: author, Private: private}
}

// The role table of PRD §5 bis, row by row, on someone else's public request.
func TestRoleTable(t *testing.T) {
	others := req(shop, paul, false)
	cases := []struct {
		role                                                 Role
		viewOthers, create, comment, roadmap, triage, access bool
	}{
		{RoleViewer, true, false, false, true, false, false},
		{RoleReporter, false, true, false, true, false, false},
		{RoleContributor, true, true, true, true, false, false},
		{RoleTriager, true, true, true, true, true, false},
		{RoleManager, true, true, true, true, true, true},
	}
	for _, c := range cases {
		t.Run(c.role.String(), func(t *testing.T) {
			s := member(map[uuid.UUID]Role{shop.ID: c.role})
			check(t, "view others", Can(s, ViewRequest, others), c.viewOthers)
			check(t, "create", Can(s, CreateRequest, shop), c.create)
			check(t, "comment others", Can(s, CommentRequest, others), c.comment)
			check(t, "roadmap", Can(s, ViewProject, shop), c.roadmap)
			check(t, "triage", Can(s, TriageRequest, others), c.triage)
			check(t, "access", Can(s, ManageAccess, shop), c.access)
		})
	}
}

func TestDenyByDefault(t *testing.T) {
	nobody := member(nil)
	check(t, "view project", Can(nobody, ViewProject, shop), false)
	check(t, "create", Can(nobody, CreateRequest, shop), false)
	check(t, "view request", Can(nobody, ViewRequest, req(shop, paul, false)), false)
	check(t, "unknown resource", Can(nobody, ViewRequest, "shop"), false)
	check(t, "unknown action", Can(member(map[uuid.UUID]Role{shop.ID: RoleManager}), Action("x"), shop), false)
}

func TestNoLeakAcrossProjects(t *testing.T) {
	s := member(map[uuid.UUID]Role{shop.ID: RoleManager})
	other := req(app, paul, false)
	for _, a := range []Action{ViewRequest, CommentRequest, TriageRequest} {
		check(t, string(a), Can(s, a, other), false)
	}
	check(t, "view app", Can(s, ViewProject, app), false)
	check(t, "attachment of app", Can(s, ViewAttachment, Attachment{AuthorID: paul, Request: &other}), false)
}

func TestReporterSeesOwnOnly(t *testing.T) {
	s := member(map[uuid.UUID]Role{shop.ID: RoleReporter, open.ID: RoleReporter})
	check(t, "own", Can(s, ViewRequest, req(shop, marie, false)), true)
	check(t, "comment own", Can(s, CommentRequest, req(shop, marie, false)), true)
	check(t, "others", Can(s, ViewRequest, req(shop, paul, false)), false)
	check(t, "others, project opens to reporters", Can(s, ViewRequest, req(open, paul, false)), true)
	check(t, "comment others, project opens to reporters", Can(s, CommentRequest, req(open, paul, false)), false)
	check(t, "private others, project opens to reporters", Can(s, ViewRequest, req(open, paul, true)), false)
}

func TestPrivateRequests(t *testing.T) {
	private := req(shop, paul, true)
	for role, want := range map[Role]bool{
		RoleViewer: false, RoleReporter: false, RoleContributor: false, RoleTriager: true, RoleManager: true,
	} {
		check(t, role.String(), Can(member(map[uuid.UUID]Role{shop.ID: role}), ViewRequest, private), want)
	}
	author := Subject{UserID: paul, Org: OrgMember, Roles: map[uuid.UUID]Role{shop.ID: RoleReporter}}
	check(t, "author", Can(author, ViewRequest, private), true)
	check(t, "admin", Can(Subject{UserID: marie, Org: OrgAdmin}, ViewRequest, private), true)
}

func TestTransferredRequestStaysVisibleToItsAuthor(t *testing.T) {
	moved := Request{Project: app, OriginProject: shop.ID, AuthorID: marie}
	s := member(map[uuid.UUID]Role{shop.ID: RoleReporter})
	check(t, "author", Can(s, ViewRequest, moved), true)
	other := Subject{UserID: paul, Org: OrgMember, Roles: map[uuid.UUID]Role{shop.ID: RoleContributor}}
	check(t, "contributor of the old project", Can(other, ViewRequest, moved), false)
	revoked := member(nil)
	check(t, "author without any role left", Can(revoked, ViewRequest, moved), false)
}

func TestAttachments(t *testing.T) {
	s := member(map[uuid.UUID]Role{shop.ID: RoleReporter})
	check(t, "own unlinked", Can(s, ViewAttachment, Attachment{AuthorID: marie}), true)
	check(t, "someone else's unlinked", Can(s, ViewAttachment, Attachment{AuthorID: paul}), false)
	others := req(shop, paul, false)
	check(t, "on a request I cannot see", Can(s, ViewAttachment, Attachment{AuthorID: paul, Request: &others}), false)
	mine := req(shop, marie, false)
	check(t, "on my request", Can(s, ViewAttachment, Attachment{AuthorID: paul, Request: &mine}), true)
	check(t, "wrong action", Can(s, CreateRequest, Attachment{AuthorID: marie}), false)
}

func TestAdmins(t *testing.T) {
	for _, org := range []OrgRole{OrgOwner, OrgAdmin} {
		s := Subject{UserID: marie, Org: org}
		check(t, string(org)+" view", Can(s, ViewRequest, req(app, paul, true)), true)
		check(t, string(org)+" triage", Can(s, TriageRequest, req(app, paul, false)), true)
		check(t, string(org)+" access", Can(s, ManageAccess, app), true)
		check(t, string(org)+" audited", ViaOrgRoleOnly(s, app.ID), true)
	}
	granted := Subject{UserID: marie, Org: OrgAdmin, Roles: map[uuid.UUID]Role{app.ID: RoleViewer}}
	check(t, "admin with a grant is not audited", ViaOrgRoleOnly(granted, app.ID), false)
	check(t, "member never audited", ViaOrgRoleOnly(member(nil), app.ID), false)
}

func TestNilUserIsNeverAuthor(t *testing.T) {
	anon := Subject{Roles: map[uuid.UUID]Role{shop.ID: RoleReporter}}
	check(t, "nil author", Can(anon, ViewRequest, Request{Project: shop, OriginProject: shop.ID}), false)
}

func check(t *testing.T, what string, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", what, got, want)
	}
}

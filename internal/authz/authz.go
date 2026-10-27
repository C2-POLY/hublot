// SPDX-License-Identifier: AGPL-3.0-or-later

// Package authz holds the single authorization function of the portal (PRD §5 bis).
// Every handler asks Can before reading or writing anything; nothing is hidden only in the UI.
// Deny by default: a subject without a grant on a project sees nothing of it.
package authz

import "github.com/google/uuid"

type OrgRole string

const (
	OrgOwner  OrgRole = "owner"
	OrgAdmin  OrgRole = "admin"
	OrgMember OrgRole = "member"
)

// Role is a per-project role. The order matters: a higher value includes the lower ones,
// except that a viewer sees every request while a reporter sees only their own.
type Role int

const (
	RoleNone Role = iota
	RoleViewer
	RoleReporter
	RoleContributor
	RoleTriager
	RoleManager
)

var roleNames = map[string]Role{
	"viewer":      RoleViewer,
	"reporter":    RoleReporter,
	"contributor": RoleContributor,
	"triager":     RoleTriager,
	"manager":     RoleManager,
}

// ParseRole returns RoleNone for an unknown name.
func ParseRole(name string) Role { return roleNames[name] }

func (r Role) String() string {
	for name, role := range roleNames {
		if role == r {
			return name
		}
	}
	return "none"
}

// Subject is the person acting, with their effective role per project
// (the highest of their direct grant and their groups' grants).
type Subject struct {
	UserID uuid.UUID
	Org    OrgRole
	Roles  map[uuid.UUID]Role
}

func (s Subject) IsAdmin() bool { return s.Org == OrgOwner || s.Org == OrgAdmin }

func (s Subject) role(project uuid.UUID) Role { return s.Roles[project] }

// Project is what authz needs to know about a project.
type Project struct {
	ID              uuid.UUID
	ReportersSeeAll bool
}

// Request is what authz needs to know about a request.
type Request struct {
	Project       Project
	OriginProject uuid.UUID
	AuthorID      uuid.UUID
	Private       bool
}

// Transfer is a request about to move to another project.
type Transfer struct {
	Request Request
	To      Project
}

// Attachment is what authz needs to know about an attachment.
type Attachment struct {
	AuthorID uuid.UUID
	Request  *Request // nil while not yet linked to a request
}

type Action string

const (
	ViewProject    Action = "project.view"    // see the project and its roadmap
	CreateRequest  Action = "request.create"  // file a request in the project
	ManageAccess   Action = "project.access"  // manage the project's grants
	ViewRequest    Action = "request.view"    // read a request
	CommentRequest Action = "request.comment" // comment on a request
	TriageProject  Action = "project.triage"  // see the project's triage inbox
	TriageRequest  Action = "request.triage"  // classify, prioritise, close
	// TransferRequest needs triage rights on both the request's project and the destination.
	TransferRequest Action = "request.transfer"
	ViewAttachment  Action = "attachment.view" // download an attachment
)

// Can is the only authorization decision of the portal. Unknown action or resource: denied.
func Can(s Subject, a Action, resource any) bool {
	switch r := resource.(type) {
	case Project:
		return canProject(s, a, r)
	case Request:
		return canRequest(s, a, r)
	case Attachment:
		return canAttachment(s, a, r)
	case Transfer:
		return a == TransferRequest && r.To.ID != r.Request.Project.ID &&
			canRequest(s, TriageRequest, r.Request) && canProject(s, TriageProject, r.To)
	}
	return false
}

func canProject(s Subject, a Action, p Project) bool {
	if s.IsAdmin() {
		return a == ViewProject || a == CreateRequest || a == ManageAccess || a == TriageProject
	}
	role := s.role(p.ID)
	switch a {
	case ViewProject:
		return role >= RoleViewer
	case CreateRequest:
		return role >= RoleReporter
	case TriageProject:
		return role >= RoleTriager
	case ManageAccess:
		return role >= RoleManager
	}
	return false
}

func canRequest(s Subject, a Action, r Request) bool {
	switch a {
	case ViewRequest:
		return canViewRequest(s, r)
	case CommentRequest:
		if !canViewRequest(s, r) {
			return false
		}
		return s.IsAdmin() || s.role(r.Project.ID) >= RoleContributor || isAuthor(s, r)
	case TriageRequest:
		return s.IsAdmin() || s.role(r.Project.ID) >= RoleTriager
	}
	return false
}

func canViewRequest(s Subject, r Request) bool {
	if s.IsAdmin() {
		return true
	}
	role := s.role(r.Project.ID)
	// The author keeps seeing a request transferred elsewhere, as long as they
	// still have a role on the project it was filed in or on its new project.
	if isAuthor(s, r) && (role >= RoleViewer || s.role(r.OriginProject) >= RoleViewer) {
		return true
	}
	if role >= RoleTriager {
		return true
	}
	if r.Private {
		return false
	}
	switch role {
	case RoleViewer, RoleContributor:
		return true
	case RoleReporter:
		return r.Project.ReportersSeeAll
	}
	return false
}

func canAttachment(s Subject, a Action, at Attachment) bool {
	if a != ViewAttachment {
		return false
	}
	if at.Request == nil {
		return at.AuthorID == s.UserID
	}
	return canViewRequest(s, *at.Request)
}

func isAuthor(s Subject, r Request) bool {
	return s.UserID != uuid.Nil && r.AuthorID == s.UserID
}

// ViaOrgRoleOnly reports whether the subject reaches the project only through their
// organisation role: such consultations are audited (PRD §5 bis).
func ViaOrgRoleOnly(s Subject, project uuid.UUID) bool {
	return s.IsAdmin() && s.role(project) == RoleNone
}

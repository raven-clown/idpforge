package httpserver

import "github.com/gofiber/fiber/v2"

type createRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) handleCreateRole(c *fiber.Ctx) error {
	var req createRoleRequest
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	role, err := s.rbacAdm.CreateRole(c.Context(), req.Name, req.Description)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "could not create role")
	}
	return c.Status(fiber.StatusCreated).JSON(role)
}

func (s *Server) handleListRoles(c *fiber.Ctx) error {
	roles, err := s.rbacAdm.ListRoles(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list roles")
	}
	return c.JSON(fiber.Map{"roles": roles})
}

func (s *Server) handleGetRole(c *fiber.Ctx) error {
	role, err := s.rbacAdm.GetRole(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "role not found")
	}
	return c.JSON(role)
}

func (s *Server) handleListRolePermissions(c *fiber.Ctx) error {
	perms, err := s.rbacAdm.ListRolePermissions(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list role permissions")
	}
	return c.JSON(fiber.Map{"permissions": perms})
}

func (s *Server) handleDeleteRole(c *fiber.Ctx) error {
	if err := s.rbacAdm.DeleteRole(c.Context(), c.Params("id")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not delete role")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type createPermissionRequest struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func (s *Server) handleCreatePermission(c *fiber.Ctx) error {
	var req createPermissionRequest
	if err := c.BodyParser(&req); err != nil || req.Resource == "" || req.Action == "" {
		return fiber.NewError(fiber.StatusBadRequest, "resource and action are required")
	}
	perm, err := s.rbacAdm.CreatePermission(c.Context(), req.Resource, req.Action)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "could not create permission")
	}
	return c.Status(fiber.StatusCreated).JSON(perm)
}

func (s *Server) handleListPermissions(c *fiber.Ctx) error {
	perms, err := s.rbacAdm.ListPermissions(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list permissions")
	}
	return c.JSON(fiber.Map{"permissions": perms})
}

type permissionRefRequest struct {
	PermissionID string `json:"permission_id"`
}

func (s *Server) handleGrantPermissionToRole(c *fiber.Ctx) error {
	var req permissionRefRequest
	if err := c.BodyParser(&req); err != nil || req.PermissionID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "permission_id is required")
	}
	if err := s.rbacAdm.GrantPermissionToRole(c.Context(), c.Params("id"), req.PermissionID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not grant permission")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) handleRevokePermissionFromRole(c *fiber.Ctx) error {
	if err := s.rbacAdm.RevokePermissionFromRole(c.Context(), c.Params("id"), c.Params("permission_id")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not revoke permission")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) handleListUserRoles(c *fiber.Ctx) error {
	roles, err := s.rbacAdm.ListUserRoles(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list user roles")
	}
	return c.JSON(fiber.Map{"roles": roles})
}

type roleRefRequest struct {
	RoleID string `json:"role_id"`
}

func (s *Server) handleAssignRoleToUser(c *fiber.Ctx) error {
	var req roleRefRequest
	if err := c.BodyParser(&req); err != nil || req.RoleID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role_id is required")
	}
	if err := s.rbacAdm.AssignRoleToUser(c.Context(), c.Params("id"), req.RoleID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not assign role")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) handleRemoveRoleFromUser(c *fiber.Ctx) error {
	if err := s.rbacAdm.RemoveRoleFromUser(c.Context(), c.Params("id"), c.Params("role_id")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not remove role")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type createGroupRequest struct {
	Name          string  `json:"name"`
	ParentGroupID *string `json:"parent_group_id"`
}

func (s *Server) handleCreateGroup(c *fiber.Ctx) error {
	var req createGroupRequest
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	group, err := s.rbacAdm.CreateGroup(c.Context(), req.Name, req.ParentGroupID)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "could not create group")
	}
	return c.Status(fiber.StatusCreated).JSON(group)
}

func (s *Server) handleListGroups(c *fiber.Ctx) error {
	groups, err := s.rbacAdm.ListGroups(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list groups")
	}
	return c.JSON(fiber.Map{"groups": groups})
}

func (s *Server) handleAssignRoleToGroup(c *fiber.Ctx) error {
	var req roleRefRequest
	if err := c.BodyParser(&req); err != nil || req.RoleID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role_id is required")
	}
	if err := s.rbacAdm.AssignRoleToGroup(c.Context(), c.Params("id"), req.RoleID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not assign role")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) handleRemoveRoleFromGroup(c *fiber.Ctx) error {
	if err := s.rbacAdm.RemoveRoleFromGroup(c.Context(), c.Params("id"), c.Params("role_id")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not remove role")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) handleAddUserToGroup(c *fiber.Ctx) error {
	if err := s.rbacAdm.AddUserToGroup(c.Context(), c.Params("user_id"), c.Params("id")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not add user to group")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) handleRemoveUserFromGroup(c *fiber.Ctx) error {
	if err := s.rbacAdm.RemoveUserFromGroup(c.Context(), c.Params("user_id"), c.Params("id")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not remove user from group")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

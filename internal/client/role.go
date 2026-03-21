package client

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type Role struct {
	ID            int64   `json:"id,omitempty"`
	Name          string  `json:"name,omitempty"`
	UserIDs       []int64 `json:"user_ids,omitempty"`
	GroupIDs      []int64 `json:"group_ids,omitempty"`
	PermissionIDs []int64 `json:"permission_ids,omitempty"`
}

type RoleCreateRequest struct {
	Name string `json:"name"`
}

type RoleUpdateRequest struct {
	Name string `json:"name"`
}

type roleResponse struct {
	ID     int64 `json:"id"`
	Result Role  `json:"result"`
}

type roleCreateResponse struct {
	ID     int64 `json:"id"`
	Result Role  `json:"result"`
}

type roleListResponse struct {
	Count  int    `json:"count"`
	Result []Role `json:"result"`
}

type rolePermissionsRequest struct {
	PermissionViewMenuIDs []int64 `json:"permission_view_menu_ids"`
}

type rolePermissionsResponse struct {
	Result rolePermissionsRequest `json:"result"`
}

func (c *Client) CreateRole(ctx context.Context, request RoleCreateRequest) (*Role, error) {
	var response roleCreateResponse

	if err := c.Post(ctx, "/api/v1/security/roles/", request, &response); err != nil {
		return nil, err
	}

	role := response.Result

	if role.ID == 0 {
		role.ID = response.ID
	}

	return &role, nil
}

func (c *Client) GetRole(ctx context.Context, id int64) (*Role, error) {
	var response roleResponse

	if err := c.Get(ctx, rolePath(id), &response); err != nil {
		return nil, err
	}

	role := response.Result

	if role.ID == 0 {
		role.ID = response.ID
	}

	return &role, nil
}

func (c *Client) UpdateRole(ctx context.Context, id int64, request RoleUpdateRequest) (*Role, error) {
	var response roleResponse

	if err := c.Put(ctx, rolePath(id), request, &response); err != nil {
		return nil, err
	}

	role := response.Result
	role.ID = id

	return &role, nil
}

func (c *Client) DeleteRole(ctx context.Context, id int64) error {
	var response map[string]any

	return c.Delete(ctx, rolePath(id), &response)
}

func (c *Client) ListRoles(ctx context.Context, pageSize int) ([]Role, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}

	roles := make([]Role, 0, pageSize)

	for page := 0; ; page++ {
		if err := validatePagination(ctx, page, c.paginationLimit()); err != nil {
			return nil, err
		}

		var response roleListResponse

		requestPath := fmt.Sprintf("/api/v1/security/roles/search/?q=%s", securityListQuery(page, pageSize))
		if err := c.Get(ctx, requestPath, &response); err != nil {
			return nil, err
		}

		roles = append(roles, response.Result...)

		if len(response.Result) == 0 || len(response.Result) < pageSize {
			return roles, nil
		}

		if response.Count > 0 && len(roles) >= response.Count {
			return roles, nil
		}
	}
}

func (c *Client) SetRolePermissions(ctx context.Context, id int64, permissionIDs []int64) error {
	normalized := append([]int64(nil), permissionIDs...)
	if normalized == nil {
		normalized = []int64{}
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })

	var response rolePermissionsResponse

	return c.Post(ctx, fmt.Sprintf("%s/permissions", rolePath(id)), rolePermissionsRequest{
		PermissionViewMenuIDs: normalized,
	}, &response)
}

type Permission struct {
	ID             int64             `json:"id,omitempty"`
	Permission     PermissionNameRef `json:"permission"`
	ViewMenu       PermissionNameRef `json:"view_menu"`
	PermissionName string            `json:"-"`
	ViewMenuName   string            `json:"-"`
}

type PermissionNameRef struct {
	Name string `json:"name,omitempty"`
}

type permissionResponse struct {
	ID     int64      `json:"id"`
	Result Permission `json:"result"`
}

type permissionListResponse struct {
	Count  int          `json:"count"`
	Result []Permission `json:"result"`
}

func (c *Client) GetPermission(ctx context.Context, id int64) (*Permission, error) {
	var response permissionResponse

	if err := c.Get(ctx, permissionPath(id), &response); err != nil {
		return nil, err
	}

	permission := permissionFromResponse(response)

	return &permission, nil
}

func (c *Client) ListPermissions(ctx context.Context, pageSize int) ([]Permission, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}

	permissions := make([]Permission, 0, pageSize)

	for page := 0; ; page++ {
		if err := validatePagination(ctx, page, c.paginationLimit()); err != nil {
			return nil, err
		}

		var response permissionListResponse

		requestPath := fmt.Sprintf("/api/v1/security/permissions-resources/?q=%s", securityListQuery(page, pageSize))
		if err := c.Get(ctx, requestPath, &response); err != nil {
			return nil, err
		}

		for _, permission := range response.Result {
			permissions = append(permissions, permissionFromResponse(permissionResponse{Result: permission}))
		}

		if len(response.Result) == 0 || len(response.Result) < pageSize {
			return permissions, nil
		}

		if response.Count > 0 && len(permissions) >= response.Count {
			return permissions, nil
		}
	}
}

func permissionFromResponse(response permissionResponse) Permission {
	permission := response.Result

	if permission.ID == 0 {
		permission.ID = response.ID
	}

	permission.PermissionName = permission.Permission.Name
	permission.ViewMenuName = permission.ViewMenu.Name

	return permission
}

func rolePath(id int64) string {
	return fmt.Sprintf("/api/v1/security/roles/%d", id)
}

func permissionPath(id int64) string {
	return fmt.Sprintf("/api/v1/security/permissions-resources/%d", id)
}

func securityListQuery(page int, pageSize int) string {
	raw := fmt.Sprintf("(page:%d,page_size:%d)", page, pageSize)

	return url.QueryEscape(strings.TrimSpace(raw))
}

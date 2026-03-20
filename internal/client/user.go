package client

import (
	"context"
	"fmt"
)

type User struct {
	ID        int64      `json:"id,omitempty"`
	Username  string     `json:"username,omitempty"`
	FirstName string     `json:"first_name,omitempty"`
	LastName  string     `json:"last_name,omitempty"`
	Email     string     `json:"email,omitempty"`
	Active    bool       `json:"active"`
	Roles     []RoleRef  `json:"roles,omitempty"`
	Groups    []GroupRef `json:"groups,omitempty"`
}

type RoleRef struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type GroupRef struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type UserCreateRequest struct {
	Username  string  `json:"username"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Active    bool    `json:"active"`
	Password  string  `json:"password"`
	Roles     []int64 `json:"roles"`
	Groups    []int64 `json:"groups"`
}

type UserUpdateRequest struct {
	Username  string  `json:"username"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Active    bool    `json:"active"`
	Password  *string `json:"password,omitempty"`
	Roles     []int64 `json:"roles"`
	Groups    []int64 `json:"groups"`
}

type userResponse struct {
	ID     int64 `json:"id"`
	Result User  `json:"result"`
}

type userCreateResponse struct {
	ID int64 `json:"id"`
}

func (c *Client) CreateUser(ctx context.Context, request UserCreateRequest) (*User, error) {
	var response userCreateResponse

	if err := c.Post(ctx, "/api/v1/security/users/", request, &response); err != nil {
		return nil, err
	}

	return &User{ID: response.ID}, nil
}

func (c *Client) GetUser(ctx context.Context, id int64) (*User, error) {
	var response userResponse

	if err := c.Get(ctx, userPath(id), &response); err != nil {
		return nil, err
	}

	user := response.Result
	if user.ID == 0 {
		user.ID = response.ID
	}

	return &user, nil
}

func (c *Client) UpdateUser(ctx context.Context, id int64, request UserUpdateRequest) error {
	var response map[string]any

	return c.Put(ctx, userPath(id), request, &response)
}

func (c *Client) DeleteUser(ctx context.Context, id int64) error {
	var response map[string]any

	return c.Delete(ctx, userPath(id), &response)
}

func userPath(id int64) string {
	return fmt.Sprintf("/api/v1/security/users/%d", id)
}

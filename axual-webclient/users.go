package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) GetUsers() (*UsersResponse, error) {
	o := UsersResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/users", c.ApiURL), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetUser(id string) (*UserResponse, error) {
	o := UserResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/users/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateUser(id string, data UserRequest) (*UserResponse, error) {
	var roles []UserRole
	roles = data.Roles
	err := c.UpdateUserRoles(id, roles)
	if err != nil {
		return nil, err
	}

	o := UserResponse{}
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/users/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (c *Client) DeleteUser(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/users/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateUser(data UserRequest) (*UserResponse, error) {
	o := UserResponse{}

	up := UserRequestWithPass{data, "kkdiennc"}
	marshal, err := json.Marshal(up)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("POST", fmt.Sprintf("%s/users", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	err = c.UpdateUserRoles(o.Uid, data.Roles)
	if err != nil {
		return nil, err
	}
	user, err := c.GetUser(o.Uid)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (c *Client) UpdateUserRoles(id string, data []UserRole) error {
	marshal, err := json.Marshal(data)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/users/%s/roles", c.ApiURL, id), strings.NewReader(string(marshal)), headers, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) FindUserByEmail(email string) (*UsersResponse, error) {
	o := UsersResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/users/search/findByEmailAddress?email=%s", c.ApiURL, url.QueryEscape(email)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

type ErrUserRole struct {
	msg string
}

func (err ErrUserRole) Error() string {
	return err.msg
}

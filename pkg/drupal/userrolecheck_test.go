package drupal_test

import (
	"fmt"
	"os/exec"
	"reflect"
	"testing"

	"github.com/salsadigitalauorg/shipshape/internal"
	"github.com/salsadigitalauorg/shipshape/pkg/command"
	"github.com/salsadigitalauorg/shipshape/pkg/drupal"
	"github.com/salsadigitalauorg/shipshape/pkg/shipshape"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	c := drupal.UserRoleCheck{}
	c.Init(drupal.UserRole)
	assert.True(t, c.RequiresDb)
}

func TestUserRoleMerge(t *testing.T) {
	assert := assert.New(t)

	c := drupal.UserRoleCheck{
		DrushCommand: drupal.DrushCommand{
			DrushPath: "/path/to/drush",
		},
		Roles:        []string{"role1"},
		AllowedUsers: []int{1, 2},
	}
	c.Merge(&drupal.UserRoleCheck{
		DrushCommand: drupal.DrushCommand{
			DrushPath: "/new/path/to/drush",
		},
		Roles:        []string{"role2"},
		AllowedUsers: []int{2, 3},
	})
	assert.EqualValues(drupal.UserRoleCheck{
		DrushCommand: drupal.DrushCommand{
			DrushPath: "/new/path/to/drush",
		},
		Roles:        []string{"role2"},
		AllowedUsers: []int{2, 3},
	}, c)
}

func TestFetchData(t *testing.T) {
	assert := assert.New(t)

	t.Run("drushNotFound", func(t *testing.T) {
		c := drupal.UserRoleCheck{}
		c.FetchData()
		assert.Equal(shipshape.Fail, c.Result.Status)
		assert.EqualValues([]string{"vendor/drush/drush/drush: no such file or directory"}, c.Result.Failures)

	})

	curShellCommander := command.ShellCommander
	defer func() { command.ShellCommander = curShellCommander }()
	t.Run("drushError", func(t *testing.T) {
		command.ShellCommander = func(name string, arg ...string) command.IShellCommand {
			return internal.TestShellCommand{
				OutputterFunc: func() ([]byte, error) {
					return nil, &exec.ExitError{Stderr: []byte("unable to run drush command")}
				},
			}
		}
		c := drupal.UserRoleCheck{}
		c.FetchData()
		assert.Equal(shipshape.Fail, c.Result.Status)
		assert.EqualValues([]string{"unable to run drush command"}, c.Result.Failures)
	})

	// correct data.
	t.Run("correctData", func(t *testing.T) {
		command.ShellCommander = func(name string, arg ...string) command.IShellCommand {
			return internal.TestShellCommand{
				OutputterFunc: func() ([]byte, error) {
					return []byte(`{"1":{"roles":["authenticated"]}}`), nil
				},
			}
		}
		c := drupal.UserRoleCheck{}
		c.FetchData()
		assert.NotEqual(shipshape.Fail, c.Result.Status)
		assert.NotEqual(shipshape.Pass, c.Result.Status)
		assert.Equal([]byte(`{"1":{"roles":["authenticated"]}}`), c.DataMap["user-info"])
	})
}

func TestUnmarshalData(t *testing.T) {
	assert := assert.New(t)

	// Empty datamap.
	c := drupal.UserRoleCheck{}
	c.UnmarshalDataMap()
	assert.Equal(shipshape.Fail, c.Result.Status)
	assert.EqualValues([]string{"no data provided"}, c.Result.Failures)

	// Incorrect json.
	c = drupal.UserRoleCheck{
		CheckBase: shipshape.CheckBase{
			DataMap: map[string][]byte{
				"user-info": []byte(`{"1":{"roles":"authenticated"]}}`)},
		},
	}
	c.UnmarshalDataMap()
	assert.Equal(shipshape.Fail, c.Result.Status)
	assert.EqualValues([]string{"invalid character ']' after object key:value pair"}, c.Result.Failures)

	// Correct json.
	c = drupal.UserRoleCheck{
		CheckBase: shipshape.CheckBase{
			DataMap: map[string][]byte{
				"user-info": []byte(`{"1":{"roles":["authenticated"]}}`)},
		},
	}
	c.UnmarshalDataMap()
	assert.NotEqual(shipshape.Fail, c.Result.Status)
	assert.NotEqual(shipshape.Pass, c.Result.Status)
	userRolesVal := reflect.ValueOf(c).FieldByName("userRoles")
	assert.Equal("map[int][]string{1:[]string{\"authenticated\"}}", fmt.Sprintf("%#v", userRolesVal))
}

func TestRunCheck(t *testing.T) {
	assert := assert.New(t)

	// No disallowed roles provided.
	c := drupal.UserRoleCheck{
		CheckBase: shipshape.CheckBase{
			DataMap: map[string][]byte{
				"user-info": []byte(`{"1":{"roles":["authenticated"]}}`)},
		},
	}
	c.UnmarshalDataMap()
	c.RunCheck(false)
	assert.Equal(shipshape.Fail, c.Result.Status)
	assert.EqualValues([]string{"no disallowed role provided"}, c.Result.Failures)

	// User has disallowed roles.
	c = drupal.UserRoleCheck{
		CheckBase: shipshape.CheckBase{
			DataMap: map[string][]byte{
				"user-info": []byte(`{"1":{"roles":["authenticated","site-admin","content-admin"]}}`)},
		},
		Roles: []string{"site-admin", "content-admin"},
	}
	c.UnmarshalDataMap()
	c.RunCheck(false)
	assert.Equal(shipshape.Fail, c.Result.Status)
	assert.EqualValues([]string{"User 1 has disallowed roles: [site-admin, content-admin]"}, c.Result.Failures)

	// User allowed to have disallowed roles.
	c = drupal.UserRoleCheck{
		CheckBase: shipshape.CheckBase{
			DataMap: map[string][]byte{
				"user-info": []byte(`
				{
					"1":{"roles":["authenticated"]},
					"2":{"roles":["authenticated","site-admin","content-admin"]}
				}
				`)},
		},
		Roles:        []string{"site-admin", "content-admin"},
		AllowedUsers: []int{2},
	}
	c.UnmarshalDataMap()
	c.RunCheck(false)
	assert.Equal(shipshape.Pass, c.Result.Status)
}

package file_test

import (
	"testing"

	. "github.com/salsadigitalauorg/shipshape/pkg/checks/file"
	"github.com/salsadigitalauorg/shipshape/pkg/config"
	"github.com/salsadigitalauorg/shipshape/pkg/result"
	"github.com/stretchr/testify/assert"
)

func TestFileCheckMerge(t *testing.T) {
	assert := assert.New(t)

	c := FileCheck{
		CheckBase:         config.CheckBase{Name: "filecheck1"},
		Path:              "file-initial",
		DisallowedPattern: "pattern-initial",
	}
	err := c.Merge(&FileCheck{
		Path: "file-final",
	})
	assert.Nil(err)
	assert.EqualValues(FileCheck{
		CheckBase:         config.CheckBase{Name: "filecheck1"},
		Path:              "file-final",
		DisallowedPattern: "pattern-initial",
	}, c)

	err = c.Merge(&FileCheck{
		DisallowedPattern: "pattern-final",
	})
	assert.Nil(err)
	assert.EqualValues(FileCheck{
		CheckBase:         config.CheckBase{Name: "filecheck1"},
		Path:              "file-final",
		DisallowedPattern: "pattern-final",
	}, c)

	err = c.Merge(&FileCheck{
		CheckBase:         config.CheckBase{Name: "filecheck2"},
		DisallowedPattern: "pattern-final",
	})
	assert.Error(err, "can only merge checks with the same name")
}

func TestFileCheckRunCheck(t *testing.T) {
	assert := assert.New(t)

	config.ProjectDir = "testdata"
	c := FileCheck{
		Path:              "file-non-existent",
		DisallowedPattern: "^(adminer|phpmyadmin|bigdump)?\\.php$",
	}
	c.Init(File)
	c.RunCheck()
	assert.Equal(result.Fail, c.Result.Status)
	assert.Equal(0, len(c.Result.Passes))
	assert.EqualValues(
		[]string{"lstat testdata/file-non-existent: no such file or directory"},
		c.Result.Failures,
	)

	c = FileCheck{
		DisallowedPattern: "^(adminer|phpmyadmin|bigdump)?\\.php$",
	}
	c.Init(File)
	c.RunCheck()
	assert.Equal(result.Fail, c.Result.Status)
	assert.Equal(0, len(c.Result.Passes))
	assert.EqualValues(
		[]string{
			"Illegal file found: testdata/adminer.php",
			"Illegal file found: testdata/sub/phpmyadmin.php",
		},
		c.Result.Failures,
	)

	c = FileCheck{
		Path:              "correct",
		DisallowedPattern: "^(adminer|phpmyadmin|bigdump)?\\.php$",
	}
	c.Init(File)
	c.RunCheck()

	assert.Equal(result.Pass, c.Result.Status)
	assert.Equal(0, len(c.Result.Failures))
	assert.EqualValues([]string{"No illegal files"}, c.Result.Passes)
}

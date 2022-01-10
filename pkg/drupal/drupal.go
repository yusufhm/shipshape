package drupal

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"salsadigitalauorg/shipshape/pkg/core"
)

func (c *DrupalConfigBase) RunCheck() error {
	if c.Data == nil {
		c.Result.Error = "no data to run check on"
		return nil
	}

	err := c.UnmarshalData(c.Data)
	if err != nil {
		return err
	}

	for _, cv := range c.Values {
		kvr, err := c.CheckKeyValue(cv)
		switch kvr {
		case core.KeyValueError:
			c.Result.Failures = append(
				c.Result.Failures,
				err.Error(),
			)
			c.Result.Status = core.Fail
		case core.KeyValueNotFound:
			c.Result.Failures = append(
				c.Result.Failures,
				fmt.Sprintf("No value found for '%s'", cv.Key),
			)
			c.Result.Status = core.Fail
		case core.KeyValueNotEqual:
			c.Result.Failures = append(
				c.Result.Failures,
				fmt.Sprintf("'%s' is not equal to '%s'", cv.Key, cv.Value),
			)
			c.Result.Status = core.Fail
		case core.KeyValueEqual:
			c.Result.Passes = append(
				c.Result.Passes,
				fmt.Sprintf("'%s' is equal to '%s'", cv.Key, cv.Value),
			)
			if c.Result.Status == "" {
				c.Result.Status = core.Pass
			}
		}
	}
	return nil
}

func (c *DrupalFileConfigCheck) FetchData() error {
	var err error
	fullpath := filepath.Join(c.ProjectDir, c.ConfigPath, c.ConfigName+".yml")
	c.Data, err = ioutil.ReadFile(fullpath)
	if err != nil {
		return err
	}
	return nil
}

func (c *DrupalFileModuleCheck) RunCheck() error {
	if c.Data == nil {
		c.Result.Error = "no data to run check on"
		return nil
	}

	err := c.UnmarshalData(c.Data)
	if err != nil {
		return err
	}

	for _, m := range c.Required {
		kvr, err := c.CheckKeyValue(core.KeyValue{
			Key:   "module." + m,
			Value: "0",
		})
		// It could be a value different from 0, which still means it's enabled.
		if kvr == core.KeyValueEqual || kvr == core.KeyValueNotEqual {
			c.Result.Passes = append(
				c.Result.Passes,
				fmt.Sprintf("'%s' is enabled", m),
			)
			if c.Result.Status == "" {
				c.Result.Status = core.Pass
			}
		} else if kvr == core.KeyValueError {
			c.Result.Failures = append(
				c.Result.Failures,
				err.Error(),
			)
			c.Result.Status = core.Fail
		} else {
			c.Result.Failures = append(
				c.Result.Failures,
				fmt.Sprintf("'%s' is not enabled", m),
			)
			c.Result.Status = core.Fail
		}
	}
	for _, m := range c.Disallowed {
		kvr, err := c.CheckKeyValue(core.KeyValue{
			Key:   "module." + m,
			Value: "0",
		})
		// It could be a value different from 0, which still means it's enabled.
		if kvr == core.KeyValueEqual || kvr == core.KeyValueNotEqual {
			c.Result.Failures = append(
				c.Result.Failures,
				fmt.Sprintf("'%s' is enabled", m),
			)
			c.Result.Status = core.Fail
		} else if kvr == core.KeyValueError {
			c.Result.Failures = append(
				c.Result.Failures,
				err.Error(),
			)
			c.Result.Status = core.Fail
		} else {
			c.Result.Passes = append(
				c.Result.Passes,
				fmt.Sprintf("'%s' is not enabled", m),
			)
			if c.Result.Status == "" {
				c.Result.Status = core.Pass
			}
		}
	}
	err = c.DrupalFileConfigCheck.RunCheck()
	return err
}

func (c *DrupalFileModuleCheck) Init(pd string, ct core.CheckType) {
	c.CheckBase.Init(pd, ct)
	c.ConfigName = "core.extension"
}
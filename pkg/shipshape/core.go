package shipshape

import (
	"bufio"
	"fmt"
	"sort"
	"text/tabwriter"
)

var ProjectDir string

// Init acts as the constructor of a check and sets some initial values.
func (c *CheckBase) Init(pd string, ct CheckType) {
	ProjectDir = pd
	if c.Result.CheckType == "" {
		c.Result = Result{Name: c.Name, CheckType: ct}
	}
}

// GetName returns the name of a check.
func (c *CheckBase) GetName() string {
	return c.Name
}

// RequiresData indicates whether the check requires a DataMap to run against.
// It is designed as opt-out, so remember to set it to false if you are creating
// a check that does not require the DataMap.
func (c *CheckBase) RequiresData() bool { return true }

// FetchData contains the logic for fetching the data over which the check is
// going to run.
// This is where c.DataMap should be populated.
func (c *CheckBase) FetchData() {}

// HasData determines whether the dataMap has been populated or not.
// The Check can optionally be marked as failed if the dataMap is not populated.
func (c *CheckBase) HasData(failCheck bool) bool {
	if c.DataMap == nil {
		if failCheck {
			c.AddFail("no data available")
		}
		return false
	}
	return true
}

// UnmarshalDataMap attempts to parse the DataMap into a structure that
// can be used to execute the check. Any failure here should fail the check.
func (c *CheckBase) UnmarshalDataMap() {}

// AddFail appends a Fail to the Result and sets the Check as Fail.
func (c *CheckBase) AddFail(msg string) {
	c.Result.Status = Fail
	c.Result.Failures = append(
		c.Result.Failures,
		msg,
	)
}

// AddPass appends a Pass to the Result.
func (c *CheckBase) AddPass(msg string) {
	c.Result.Passes = append(
		c.Result.Passes,
		msg,
	)
}

// RunCheck contains the core logic for running the check and generating
// the result.
// This is where c.Result should be populated.
func (c *CheckBase) RunCheck() {
	c.AddFail("not implemented")
}

// GetResult returns the value of c.Result.
func (c *CheckBase) GetResult() *Result {
	return &c.Result
}

// Status calculates and returns the overall result of all check results.
func (rl *ResultList) Status() CheckStatus {
	for _, r := range rl.Results {
		if r.Status == Fail {
			return Fail
		}
	}
	return Pass
}

// Sort reorders the Passes & Failures in order to get consistent output.
func (r *Result) Sort() {
	if len(r.Failures) > 0 {
		sort.Slice(r.Failures, func(i int, j int) bool {
			return r.Failures[i] < r.Failures[j]
		})
	}

	if len(r.Passes) > 0 {
		sort.Slice(r.Passes, func(i int, j int) bool {
			return r.Passes[i] < r.Passes[j]
		})
	}
}

// Sort reorders the results by name.
func (rl *ResultList) Sort() {
	sort.Slice(rl.Results, func(i int, j int) bool {
		return rl.Results[i].Name < rl.Results[j].Name
	})
}

// TableDisplay generates the tabular output for the ResultList.
func (rl *ResultList) TableDisplay(w *tabwriter.Writer) {
	var linePass, lineFail string

	if len(rl.Results) == 0 {
		fmt.Fprint(w, "No result available; ensure your shipshape.yml is configured correctly.\n")
		w.Flush()
		return
	}

	fmt.Fprintf(w, "NAME\tSTATUS\tPASSES\tFAILS\n")
	for _, r := range rl.Results {
		linePass = ""
		lineFail = ""
		if len(r.Passes) > 0 {
			linePass = r.Passes[0]
		}
		if len(r.Failures) > 0 {
			lineFail = r.Failures[0]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Name, r.Status, linePass, lineFail)

		if len(r.Passes) > 1 || len(r.Failures) > 1 {
			numPasses := len(r.Passes)
			numFailures := len(r.Failures)

			// How many additional lines?
			numAddLines := numPasses
			if numFailures > numPasses {
				numAddLines = numFailures
			}

			for i := 1; i < numAddLines; i++ {
				linePass = ""
				lineFail = ""
				if numPasses > i {
					linePass = r.Passes[i]
				}
				if numFailures > i {
					lineFail = r.Failures[i]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "", "", linePass, lineFail)
			}
		}
	}
	w.Flush()
}

// SimpleDisplay output only failures to the writer.
func (rl *ResultList) SimpleDisplay(w *bufio.Writer) {
	if len(rl.Results) == 0 || rl.Status() == Pass {
		fmt.Fprint(w, "Ship is in top shape; no breach detected!\n")
		w.Flush()
		return
	}

	fmt.Fprint(w, "Breaches were detected!\n\n")
	for _, r := range rl.Results {
		if len(r.Failures) == 0 {
			continue
		}
		fmt.Fprintf(w, "  ### %s\n", r.Name)
		for _, f := range r.Failures {
			fmt.Fprintf(w, "     -- %s\n", f)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}
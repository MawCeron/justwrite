package ui

import "testing"

// withTempConfigDir points os.UserConfigDir() at a scratch directory, on
// every OS this project ships for, so a test never touches the real user's
// settings.
func withTempConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir) // linux
	t.Setenv("AppData", dir)         // windows
	t.Setenv("HOME", dir)            // darwin: $HOME/Library/Application Support
}

func TestConfigRoundTrip(t *testing.T) {
	withTempConfigDir(t)

	if c := loadConfig(); c.SessionGoal != 0 || c.DocGoal != 0 {
		t.Fatalf("config = %+v before anything was ever set, want both 0", c)
	}

	c := config{SessionGoal: 500, DocGoal: 50000}
	if err := c.save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := loadConfig()
	if got.SessionGoal != 500 || got.DocGoal != 50000 {
		t.Errorf("config = %+v, want %+v", got, c)
	}
}

func TestSaveOverwritesThePreviousValues(t *testing.T) {
	withTempConfigDir(t)

	config{SessionGoal: 300}.save()
	config{SessionGoal: 750, DocGoal: 90000}.save()

	got := loadConfig()
	if got.SessionGoal != 750 || got.DocGoal != 90000 {
		t.Errorf("config = %+v, want SessionGoal 750, DocGoal 90000", got)
	}
}

// A goal left at 0 is omitted from the file rather than written as "= 0", so
// a fresh install and a goal the user cleared read back identically.
func TestClearedGoalIsOmittedFromTheFile(t *testing.T) {
	withTempConfigDir(t)

	config{SessionGoal: 500, DocGoal: 50000}.save()
	config{SessionGoal: 0, DocGoal: 50000}.save() // session goal cleared

	got := loadConfig()
	if got.SessionGoal != 0 {
		t.Errorf("SessionGoal = %d, want 0", got.SessionGoal)
	}
	if got.DocGoal != 50000 {
		t.Errorf("DocGoal = %d, want 50000 (untouched)", got.DocGoal)
	}
}

package core

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"git.plutolab.org/plutolab/kosh/internal/config"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/encoding"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/ui"
)

type ProfileService interface {
	LoadProfiles(string, bool) ([]model.Profile, error)
	GetProfilePath() (string, error)
	ResolveProfile(string) (model.Profile, bool, error)
	SwitchProfile(profile model.Profile) error
	DeleteProfile(profile model.Profile) error
}

type KoshProfile struct{}

func NewProfileService() *KoshProfile {
	return &KoshProfile{}
}

// LoadProfiles lists the profiles whose name matches filter. Matching ignores
// case, but a profile is always reported under the name it carries on disk:
// the casing the user picked is preserved, and only comparisons fold it.
func (p *KoshProfile) LoadProfiles(filter string, exactMatch bool) ([]model.Profile, error) {
	profilePath, err := p.GetProfilePath()
	if err != nil {
		return nil, err
	}

	profileEntries, err := os.ReadDir(profilePath)
	if os.IsNotExist(err) {
		// No profiles directory yet just means no profiles.
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	filter = strings.ToLower(strings.TrimSpace(filter))
	profiles := make([]model.Profile, 0, len(profileEntries))
	for _, entry := range profileEntries {
		if entry.IsDir() {
			continue
		}

		name, isDB := strings.CutSuffix(entry.Name(), ".db")
		if !isDB {
			continue
		}

		// Fold both sides. Folding only the filter would leave a profile
		// stored as "Work" unmatchable by any spelling, including its own.
		folded := strings.ToLower(name)
		valid := filter == "" // empty string means allow all
		if !valid && exactMatch {
			valid = folded == filter
		} else if !valid {
			valid = strings.Contains(folded, filter)
		}

		if valid {
			profiles = append(profiles, model.Profile(name))
			if exactMatch {
				return profiles, nil
			}
		}
	}

	return profiles, nil
}

func (p *KoshProfile) GetProfilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	profilePath := filepath.Join(homeDir, ".kosh", "profiles")
	return profilePath, nil
}

// ResolveProfile reports whether profileName names an existing profile, and
// returns the name as it is spelled on disk - which may differ in case from the
// one asked for. Callers must use the returned name for anything that then
// touches the file. On a case-sensitive filesystem "work" and "Work" are
// different paths, so opening a profile under the spelling the user typed
// rather than the stored one would silently create an empty vault beside it.
//
// The lookup folds case on every platform, not only on the ones whose
// filesystem does. That is what keeps a profiles directory portable: two
// profiles differing only in case can coexist on Linux but collide the moment
// the directory is copied to Windows or macOS, so they are refused everywhere.
func (p *KoshProfile) ResolveProfile(profileName string) (model.Profile, bool, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return "", false, nil
	}

	// Matching against real directory entries also settles locality: a name
	// like "../../other", an absolute path or an NTFS stream can never equal an
	// entry, so it resolves to nothing instead of reaching outside the
	// profiles directory.
	profiles, err := p.LoadProfiles(profileName, true)
	if err != nil {
		return "", false, err
	}
	if len(profiles) == 0 {
		return "", false, nil
	}

	// A directory carried over from a case-sensitive machine can legitimately
	// hold both "work" and "Work". Prefer the exact spelling when it is there.
	for _, profile := range profiles {
		if string(profile) == profileName {
			return profile, true, nil
		}
	}

	slog.Debug("resolved profile", "requested", profileName, "stored", profiles[0])
	return profiles[0], true, nil
}

func (p *KoshProfile) SwitchProfile(profile model.Profile) error {
	// Change ActiveProfile in the config file.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.ActiveProfile = string(profile)
	if err := config.Save(cfg); err != nil {
		return err
	}

	ui.SetProfile(string(profile))
	return nil
}

func (p *KoshProfile) DeleteProfile(profile model.Profile) error {
	// Overwrite the file's content before deleting
	profileDir, err := p.GetProfilePath()
	if err != nil {
		return fmt.Errorf("failed to get profiles directory path")
	}

	// This is the one destructive path in the package - it scrubs the file
	// before unlinking it. A name that does not point straight at an entry of
	// the profiles directory names no profile, and must never reach
	// OverwriteFile, whatever the caller believes it resolved.
	fileName := fmt.Sprintf("%s.db", profile)
	if !filepath.IsLocal(fileName) {
		slog.Debug("refusing to delete a profile name that is not local", "name", profile)
		return constants.ErrProfileDoesNotExist
	}

	profilePath := filepath.Join(profileDir, fileName)
	err = crypto.OverwriteFile(profilePath)
	if err != nil {
		slog.Debug("failed to scrub file", "error", err)
		return err
	}

	// Delete the file
	if err := os.Remove(profilePath); err != nil {
		return err
	}
	return nil
}

var (
	illegalCharRegex     = regexp.MustCompile(`[^a-zA-Z0-9\s_-]`)
	whitespaceRegex      = regexp.MustCompile(`\s+`)
	multiUnderscoreRegex = regexp.MustCompile(`_+`)
	multiHyphenRegex     = regexp.MustCompile(`-+`)
	windowsReservedRegex = regexp.MustCompile(`^(?i)(con|prn|aux|nul|com[1-9]|lpt[1-9])$`)
)

// SanitizeProfileName cleans up profile names to be
// suitable file names.
//
// Accents are folded to ASCII, characters
// outside `A–Z a–z 0–9 _ - space` are deleted, whitespace runs become a single underscore, repeated
// `_`/`-` collapse, leading and trailing `_`/`-` are stripped, and the result is capped at 252
// characters. Case is preserved.
func SanitizeProfileName(name string) (string, error) {
	clean, err := encoding.RemoveAccent(name)
	if err != nil {
		return "", err
	}

	clean = illegalCharRegex.ReplaceAllString(strings.TrimSpace(clean), "")
	clean = whitespaceRegex.ReplaceAllString(clean, "_")
	clean = multiUnderscoreRegex.ReplaceAllString(clean, "_")
	clean = multiHyphenRegex.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "_-")

	if clean == "" {
		return "", constants.ErrInvalidProfileName
	}

	// Enforce standard file system limit (Max 255 characters)
	// `.db` takes 3 characters so adjusting for that we get
	// 252 characters remaining for the profile.
	// Trim again afterwards: the cut can land on a separator that the earlier
	// Trim had removed from the end.
	if len(clean) > 252 {
		clean = strings.Trim(clean[:252], "_-")
	}

	if windowsReservedRegex.MatchString(clean) {
		return "", constants.ErrReservedProfileName
	}

	return clean, nil
}

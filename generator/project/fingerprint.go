package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ObservationClass describes why one value participates in a fingerprint.
type ObservationClass string

const (
	// ObservationBook records the selected Book version.
	ObservationBook ObservationClass = "book"
	// ObservationModule records Go and Hatmax module configuration.
	ObservationModule ObservationClass = "module"
	// ObservationRepositoryRule records repository instructions.
	ObservationRepositoryRule ObservationClass = "repository_rule"
	// ObservationCommand records a repository-owned command.
	ObservationCommand ObservationClass = "command"
	// ObservationSelectedInput records an explicitly selected file or directory.
	ObservationSelectedInput ObservationClass = "selected_input"
	// ObservationPlannedSurface records content owned by a planned surface.
	ObservationPlannedSurface ObservationClass = "planned_surface"
	// ObservationDirtyOverlap records dirty state intersecting relevant content.
	ObservationDirtyOverlap ObservationClass = "dirty_overlap"
)

// FingerprintRequest selects the Book, inputs, and surfaces that can affect a
// plan.
type FingerprintRequest struct {
	BookVersion          int
	SelectedPaths        []string
	SelectedDependencies []string
	PlannedSurfaces      []string
}

// Observation is one stable input to a project fingerprint.
type Observation struct {
	Key     string
	Class   ObservationClass
	Path    string
	Surface string
	Digest  string
}

// Fingerprint identifies the relevant state of one inspected project.
type Fingerprint struct {
	Value                string
	BookVersion          int
	SelectedPaths        []string
	SelectedDependencies []string
	PlannedSurfaces      []string
	Observations         []Observation
}

type relevantFile struct {
	class   ObservationClass
	surface string
}

// Fingerprint computes a stable digest over selected planning inputs without
// hashing unrelated project content.
func (i Inventory) Fingerprint(request FingerprintRequest) (Fingerprint, error) {
	if request.BookVersion < 1 {
		return Fingerprint{}, projectError("project_book_version_invalid", "", "Book version must be positive")
	}

	selectedPaths, err := normalizeSelectedPaths(i.Root, request.SelectedPaths)
	if err != nil {
		return Fingerprint{}, err
	}

	surfaces, err := normalizeSurfaces(request.PlannedSurfaces)
	if err != nil {
		return Fingerprint{}, err
	}

	selectedDependencies, err := normalizeDependencies(request.SelectedDependencies)
	if err != nil {
		return Fingerprint{}, err
	}

	observations := make([]Observation, 0)

	observations, err = i.addSemanticObservations(observations, request.BookVersion, selectedDependencies)
	if err != nil {
		return Fingerprint{}, err
	}

	relevantFiles, markers, err := i.selectRelevantFiles(selectedPaths, surfaces)
	if err != nil {
		return Fingerprint{}, err
	}

	observations = append(observations, markers...)

	observations, err = i.addRepositoryRuleObservations(observations, relevantFiles)
	if err != nil {
		return Fingerprint{}, err
	}

	observations, err = i.addRelevantFileObservations(observations, relevantFiles)
	if err != nil {
		return Fingerprint{}, err
	}

	observations = i.addDirtyObservations(observations, relevantFiles, selectedPaths)
	sortObservations(observations)

	value, err := digestObservations(observations)
	if err != nil {
		return Fingerprint{}, err
	}

	return Fingerprint{
		Value:                value,
		BookVersion:          request.BookVersion,
		SelectedPaths:        cloneStrings(selectedPaths),
		SelectedDependencies: cloneStrings(selectedDependencies),
		PlannedSurfaces:      cloneStrings(surfaces),
		Observations:         observations,
	}, nil
}

func normalizeSelectedPaths(root string, paths []string) ([]string, error) {
	result := make([]string, 0, len(paths))
	for _, path := range uniqueSorted(paths) {
		_, err := projectPath(root, path)
		if err != nil {
			return nil, err
		}

		result = append(result, filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))))
	}

	return result, nil
}

func normalizeSurfaces(surfaces []string) ([]string, error) {
	result := uniqueSorted(surfaces)
	for _, surface := range result {
		if _, exists := knownSurfaces[surface]; !exists {
			return nil, projectError("project_surface_unknown", "", "unknown planned surface %q", surface)
		}
	}

	return result, nil
}

func normalizeDependencies(dependencies []string) ([]string, error) {
	result := uniqueSorted(dependencies)
	for _, dependency := range result {
		if strings.TrimSpace(dependency) == "" || strings.ContainsAny(dependency, " \t\r\n") {
			return nil, projectError("project_dependency_invalid", "", "invalid module path %q", dependency)
		}
	}

	return result, nil
}

func (i Inventory) addSemanticObservations(
	observations []Observation,
	bookVersion int,
	selectedDependencies []string,
) ([]Observation, error) {
	observations = append(observations, Observation{
		Key:    "book:version",
		Class:  ObservationBook,
		Digest: fmt.Sprintf("%d", bookVersion),
	})

	relevantDependencies := i.relevantDependencies(selectedDependencies)
	moduleState := struct {
		Path                 string
		GoVersion            string
		HatmaxSource         HatmaxSource
		HatmaxVersion        string
		SelectedDependencies []string
		Dependencies         []Dependency
	}{
		Path:                 i.Module.Path,
		GoVersion:            i.Module.GoVersion,
		HatmaxSource:         i.Module.Hatmax.Source,
		HatmaxVersion:        i.Module.Hatmax.Version,
		SelectedDependencies: selectedDependencies,
		Dependencies:         relevantDependencies,
	}

	digest, err := digestValue(moduleState)
	if err != nil {
		return nil, err
	}

	observations = append(observations, Observation{
		Key:    "module:configuration",
		Class:  ObservationModule,
		Path:   "go.mod",
		Digest: digest,
	})

	for _, command := range i.Commands {
		commandDigest, commandErr := digestValue(command)
		if commandErr != nil {
			return nil, commandErr
		}

		observations = append(observations, Observation{
			Key:    "command:" + string(command.Kind) + ":" + command.Source + ":" + command.Name,
			Class:  ObservationCommand,
			Path:   command.Source,
			Digest: commandDigest,
		})
	}

	return observations, nil
}

func (i Inventory) relevantDependencies(selected []string) []Dependency {
	requested := make(map[string]struct{}, len(selected)+1)
	requested[HatmaxModulePath] = struct{}{}

	for _, dependency := range selected {
		requested[dependency] = struct{}{}
	}

	result := make([]Dependency, 0, len(requested))
	for _, dependency := range i.Module.Dependencies {
		if _, relevant := requested[dependency.Path]; relevant {
			result = append(result, dependency)
		}
	}

	return result
}

func (i Inventory) selectRelevantFiles(selectedPaths, surfaces []string) (map[string]relevantFile, []Observation, error) {
	relevant := make(map[string]relevantFile)
	markers := make([]Observation, 0, len(selectedPaths)+len(surfaces))

	for _, path := range selectedPaths {
		absolute, err := projectPath(i.Root, path)
		if err != nil {
			return nil, nil, err
		}

		info, err := os.Lstat(absolute)
		if errors.Is(err, os.ErrNotExist) {
			markers = append(markers, Observation{
				Key:    "file:" + path,
				Class:  ObservationSelectedInput,
				Path:   path,
				Digest: "missing",
			})

			continue
		}

		if err != nil {
			return nil, nil, unexpectedReadError(path, err)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return nil, nil, projectError("project_path_unsupported", path, "selected paths cannot be symbolic links")
		}

		if !info.IsDir() {
			relevant[path] = relevantFile{class: ObservationSelectedInput}

			continue
		}

		markers = append(markers, Observation{
			Key:    "file:" + path,
			Class:  ObservationSelectedInput,
			Path:   path,
			Digest: "directory",
		})

		for _, file := range i.files {
			if pathsOverlap(file.path, path) && file.path != path {
				relevant[file.path] = relevantFile{class: ObservationSelectedInput}
			}
		}
	}

	for _, surface := range surfaces {
		markers = append(markers, Observation{
			Key:     "surface:" + surface,
			Class:   ObservationPlannedSurface,
			Surface: surface,
			Digest:  "selected",
		})

		for _, file := range i.files {
			if !containsValue(file.surfaces, surface) {
				continue
			}

			if _, selected := relevant[file.path]; selected {
				continue
			}

			relevant[file.path] = relevantFile{
				class:   ObservationPlannedSurface,
				surface: surface,
			}
		}
	}

	return relevant, markers, nil
}

func (i Inventory) addRepositoryRuleObservations(
	observations []Observation,
	relevant map[string]relevantFile,
) ([]Observation, error) {
	for _, path := range i.Rules.Instructions {
		if _, selected := relevant[path]; selected {
			continue
		}

		digest, err := i.digestProjectFile(path)
		if err != nil {
			return nil, err
		}

		observations = append(observations, Observation{
			Key:    "rule:" + path,
			Class:  ObservationRepositoryRule,
			Path:   path,
			Digest: digest,
		})
	}

	protectedDigest, err := digestValue(i.Rules.ProtectedPaths)
	if err != nil {
		return nil, err
	}

	observations = append(observations, Observation{
		Key:    "rule:protected_paths",
		Class:  ObservationRepositoryRule,
		Digest: protectedDigest,
	})

	commandSources := make(map[string]struct{})
	for _, command := range i.Commands {
		commandSources[command.Source] = struct{}{}
	}

	for _, path := range sortedKeys(commandSources) {
		if _, selected := relevant[path]; selected {
			continue
		}

		digest, digestErr := i.digestProjectFile(path)
		if digestErr != nil {
			return nil, digestErr
		}

		observations = append(observations, Observation{
			Key:    "command:file:" + path,
			Class:  ObservationCommand,
			Path:   path,
			Digest: digest,
		})
	}

	if i.Module.GoWorkPath != "" {
		digest, err := digestFile(i.Module.GoWorkPath, i.maximumFileSize)
		if err != nil {
			return nil, err
		}

		observations = append(observations, Observation{
			Key:    "module:file:workspace",
			Class:  ObservationModule,
			Path:   "workspace/go.work",
			Digest: digest,
		})
	}

	if i.Module.Hatmax.LocalPath != "" {
		path := filepath.Join(i.Module.Hatmax.LocalPath, "go.mod")

		digest, err := digestFile(path, i.maximumFileSize)
		if err != nil {
			return nil, err
		}

		observations = append(observations, Observation{
			Key:    "module:file:local_hatmax",
			Class:  ObservationModule,
			Path:   "local-hatmax/go.mod",
			Digest: digest,
		})
	}

	return observations, nil
}

func (i Inventory) addRelevantFileObservations(
	observations []Observation,
	relevant map[string]relevantFile,
) ([]Observation, error) {
	paths := make([]string, 0, len(relevant))
	for path := range relevant {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	for _, path := range paths {
		digest, err := i.digestProjectFile(path)
		if err != nil {
			return nil, err
		}

		selection := relevant[path]
		observations = append(observations, Observation{
			Key:     "file:" + path,
			Class:   selection.class,
			Path:    path,
			Surface: selection.surface,
			Digest:  digest,
		})
	}

	return observations, nil
}

func (i Inventory) addDirtyObservations(
	observations []Observation,
	relevant map[string]relevantFile,
	selectedPaths []string,
) []Observation {
	for _, dirty := range i.Repository.Dirty {
		if !dirtyIntersectsRelevant(dirty, relevant, selectedPaths) {
			continue
		}

		digest := fmt.Sprintf("%c%c:%s", dirty.IndexStatus, dirty.WorktreeStatus, dirty.OriginalPath)
		observations = append(observations, Observation{
			Key:    "dirty:" + dirty.Path,
			Class:  ObservationDirtyOverlap,
			Path:   dirty.Path,
			Digest: digest,
		})
	}

	return observations
}

func dirtyIntersectsRelevant(dirty DirtyPath, relevant map[string]relevantFile, selectedPaths []string) bool {
	for path := range relevant {
		if pathsOverlap(dirty.Path, path) || pathsOverlap(dirty.OriginalPath, path) {
			return true
		}
	}

	for _, path := range selectedPaths {
		if pathsOverlap(dirty.Path, path) || pathsOverlap(dirty.OriginalPath, path) {
			return true
		}
	}

	return false
}

func (i Inventory) digestProjectFile(path string) (string, error) {
	absolute, err := projectPath(i.Root, path)
	if err != nil {
		return "", err
	}

	return digestFile(absolute, i.maximumFileSize)
}

func digestFile(path string, maximumSize int64) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "missing", nil
	}

	if err != nil {
		return "", unexpectedReadError(path, err)
	}

	if !info.Mode().IsRegular() {
		return "", projectError("project_path_unsupported", path, "fingerprinted paths must be regular files")
	}

	if maximumSize == 0 {
		maximumSize = defaultMaximumFileSize
	}

	if info.Size() > maximumSize {
		return "", projectError("project_limit_exceeded", path, "file exceeds the %d byte fingerprint limit", maximumSize)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", unexpectedReadError(path, err)
	}
	defer file.Close()

	digest := sha256.New()

	_, err = io.Copy(digest, file)
	if err != nil {
		return "", unexpectedReadError(path, err)
	}

	return "sha256:" + hex.EncodeToString(digest.Sum(nil)), nil
}

func digestValue(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", projectError("project_fingerprint_failed", "", "%v", err)
	}

	digest := sha256.Sum256(data)

	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func digestObservations(observations []Observation) (string, error) {
	return digestValue(observations)
}

func sortObservations(observations []Observation) {
	sort.Slice(observations, func(left, right int) bool {
		return observations[left].Key < observations[right].Key
	})
}

func containsValue(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}

// DirtyForSurfaces returns dirty paths classified under planned surfaces.
func (i Inventory) DirtyForSurfaces(surfaces []string) ([]DirtyPath, error) {
	normalized, err := normalizeSurfaces(surfaces)
	if err != nil {
		return nil, err
	}

	relevant := make(map[string]relevantFile)

	for _, file := range i.files {
		for _, surface := range normalized {
			if containsValue(file.surfaces, surface) {
				relevant[file.path] = relevantFile{class: ObservationPlannedSurface, surface: surface}

				break
			}
		}
	}

	result := make([]DirtyPath, 0)

	for _, dirty := range i.Repository.Dirty {
		if dirtyIntersectsRelevant(dirty, relevant, nil) {
			result = append(result, dirty)
		}
	}

	return result, nil
}

func observationIdentity(observation Observation) string {
	return strings.Join([]string{
		observation.Key,
		string(observation.Class),
		observation.Path,
		observation.Surface,
	}, "\x00")
}

package pluginruntime

import (
	"fmt"
	"maps"
	"sort"
	"strings"
)

// ContributionMetadata stores stable metadata for one registered contribution.
type ContributionMetadata struct {
	ID               string
	Kind             ContributionKind
	ExecutionClass   ExecutionClass
	ActivationPolicy ActivationPolicy
	Order            int
}

// ContributionRegistration stores one plugin-owned contribution registration.
type ContributionRegistration struct {
	Metadata ContributionMetadata
	Value    any
}

type registeredContribution struct {
	getPluginID string
	getMetadata ContributionMetadata
	getValue    any
}

// DevtoolsSection stores one kernel-owned devtools section contribution payload.
type DevtoolsSection struct {
	Name    string
	Summary map[string]string
	Lines   []string
}

// DevtoolsIssue stores one overlay issue summary visible to plugin actions.
type DevtoolsIssue struct {
	Source   string
	Code     string
	Message  string
	Path     string
	Docs     string
	TopFrame string
}

// DevtoolsActionContext stores one overlay action invocation context.
type DevtoolsActionContext struct {
	Issue DevtoolsIssue
}

// DevtoolsAction stores one kernel-owned overlay action payload.
type DevtoolsAction struct {
	Label        string
	MatchCodes   []string
	MatchSources []string
	Run          func(DevtoolsActionContext) error
}

// DevtoolsSectionProvider resolves devtools sections at read time.
type DevtoolsSectionProvider interface {
	GetDevtoolsSections() ([]DevtoolsSection, error)
}

// DevtoolsActionProvider resolves overlay actions at read time.
type DevtoolsActionProvider interface {
	GetDevtoolsActions() ([]DevtoolsAction, error)
}

// DevtoolsSectionProviderFunc adapts one function into a section provider.
type DevtoolsSectionProviderFunc func() ([]DevtoolsSection, error)

// GetDevtoolsSections resolves one function-backed section contribution.
func (parseFunc DevtoolsSectionProviderFunc) GetDevtoolsSections() ([]DevtoolsSection, error) {
	if parseFunc == nil {
		return nil, nil
	}
	return parseFunc()
}

// DevtoolsActionProviderFunc adapts one function into an action provider.
type DevtoolsActionProviderFunc func() ([]DevtoolsAction, error)

// GetDevtoolsActions resolves one function-backed action contribution.
func (parseFunc DevtoolsActionProviderFunc) GetDevtoolsActions() ([]DevtoolsAction, error) {
	if parseFunc == nil {
		return nil, nil
	}
	return parseFunc()
}

// validateContributionRegistration validates one contribution registration.
func validateContributionRegistration(parseRegistration ContributionRegistration) error {
	parseID := strings.TrimSpace(parseRegistration.Metadata.ID)
	if parseID == "" {
		return fmt.Errorf("pluginruntime: contribution ID is required")
	}
	switch parseRegistration.Metadata.Kind {
	case ContributionKindDevtoolsSection, ContributionKindDevtoolsAction, ContributionKindDevtoolsPanel:
	default:
		return fmt.Errorf("pluginruntime: unsupported contribution kind %q", parseRegistration.Metadata.Kind)
	}
	if parseErr := validateActivationPolicy(parseRegistration.Metadata.ActivationPolicy); parseErr != nil {
		return parseErr
	}
	switch parseRegistration.Metadata.ExecutionClass {
	case "", ExecutionClassHot, ExecutionClassWarm, ExecutionClassBackground:
	default:
		return fmt.Errorf("pluginruntime: unsupported execution class %q", parseRegistration.Metadata.ExecutionClass)
	}
	if parseRegistration.Value == nil {
		return fmt.Errorf("pluginruntime: contribution %q value is required", parseID)
	}
	return nil
}

// sortRegisteredContributions sorts one contribution slice deterministically.
func sortRegisteredContributions(parseContributions []registeredContribution) {
	sort.SliceStable(parseContributions, func(parseI int, parseJ int) bool {
		getLeft := parseContributions[parseI]
		getRight := parseContributions[parseJ]
		if getLeft.getMetadata.Order != getRight.getMetadata.Order {
			return getLeft.getMetadata.Order > getRight.getMetadata.Order
		}
		if getLeft.getPluginID != getRight.getPluginID {
			return getLeft.getPluginID < getRight.getPluginID
		}
		return getLeft.getMetadata.ID < getRight.getMetadata.ID
	})
}

// cloneDevtoolsSections returns one cloned section slice.
func cloneDevtoolsSections(parseSections []DevtoolsSection) []DevtoolsSection {
	if len(parseSections) == 0 {
		return nil
	}
	buildCloned := make([]DevtoolsSection, len(parseSections))
	for parseIndex, parseSection := range parseSections {
		buildCloned[parseIndex] = DevtoolsSection{
			Name:    parseSection.Name,
			Summary: cloneStringMap(parseSection.Summary),
			Lines:   append([]string(nil), parseSection.Lines...),
		}
	}
	return buildCloned
}

// cloneDevtoolsActions returns one cloned action slice.
func cloneDevtoolsActions(parseActions []DevtoolsAction) []DevtoolsAction {
	if len(parseActions) == 0 {
		return nil
	}
	buildCloned := make([]DevtoolsAction, len(parseActions))
	for parseIndex, parseAction := range parseActions {
		buildCloned[parseIndex] = DevtoolsAction{
			Label:        parseAction.Label,
			MatchCodes:   append([]string(nil), parseAction.MatchCodes...),
			MatchSources: append([]string(nil), parseAction.MatchSources...),
			Run:          parseAction.Run,
		}
	}
	return buildCloned
}

// cloneStringMap returns one cloned string map.
func cloneStringMap(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return nil
	}
	buildCloned := make(map[string]string, len(parseValues))
	maps.Copy(buildCloned, parseValues)
	return buildCloned
}

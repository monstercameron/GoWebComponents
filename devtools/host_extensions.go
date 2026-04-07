package devtools

import "github.com/monstercameron/GoWebComponents/plugin"

// ApplyHostExtensions installs host-owned devtools sections and overlay actions, then returns a cleanup that restores the previous state.
func ApplyHostExtensions(parseHost *plugin.Host) func() {
	parsePreviousSections := InspectExtensionSections()
	parsePreviousActions := InspectErrorOverlayActions()
	SetExtensionSections(buildHostExtensionSections(parseHost))
	SetErrorOverlayActions(buildHostOverlayActions(parseHost))
	return func() {
		SetExtensionSections(parsePreviousSections)
		SetErrorOverlayActions(parsePreviousActions)
	}
}

// buildHostExtensionSections maps plugin-host devtools sections into devtools extension sections.
func buildHostExtensionSections(parseHost *plugin.Host) []ExtensionSection {
	if parseHost == nil {
		return nil
	}
	parseSections := parseHost.DevtoolsSections()
	parseMapped := make([]ExtensionSection, 0, len(parseSections))
	for _, parseSection := range parseSections {
		parseMapped = append(parseMapped, ExtensionSection{
			Name:    parseSection.Name,
			Summary: cloneHostStringMap(parseSection.Summary),
			Lines:   append([]string(nil), parseSection.Lines...),
		})
	}
	return parseMapped
}

// buildHostOverlayActions maps plugin-host devtools actions into devtools overlay actions.
func buildHostOverlayActions(parseHost *plugin.Host) []ErrorOverlayAction {
	if parseHost == nil {
		return nil
	}
	parseActions := parseHost.DevtoolsActions()
	parseMapped := make([]ErrorOverlayAction, 0, len(parseActions))
	for _, parseAction := range parseActions {
		parseCurrent := parseAction
		parseMapped = append(parseMapped, ErrorOverlayAction{
			Label:        parseCurrent.Label,
			MatchCodes:   append([]string(nil), parseCurrent.MatchCodes...),
			MatchSources: append([]string(nil), parseCurrent.MatchSources...),
			Run: func(parseContext ErrorOverlayActionContext) {
				parseCurrent.Run(plugin.DevtoolsActionContext{
					Host:          parseHost,
					IssueCode:     parseContext.Issue.Code,
					IssueSource:   parseContext.Issue.Source,
					IssuePath:     parseContext.Issue.Path,
					IssueMessage:  parseContext.Issue.Message,
					IssueDocs:     parseContext.Issue.Docs,
					IssueTopFrame: parseContext.Issue.TopFrame,
				})
			},
		})
	}
	return parseMapped
}

// cloneHostStringMap returns a cloned string map for host-owned devtools sections.
func cloneHostStringMap(parseInput map[string]string) map[string]string {
	if len(parseInput) == 0 {
		return nil
	}
	parseCloned := make(map[string]string, len(parseInput))
	for parseKey, parseValue := range parseInput {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}

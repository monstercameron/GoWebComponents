package devtools

import "maps"

import "github.com/monstercameron/GoWebComponents/v5/plugin"

// ApplyHostExtensions registers one compatibility host source and returns a cleanup that removes it.
func ApplyHostExtensions(parseHost *plugin.Host) func() {
	parseRegistrationID := registerHostExtensionSource(parseHost)
	return func() {
		unregisterHostExtensionSource(parseRegistrationID)
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
	maps.Copy(parseCloned, parseInput)
	return parseCloned
}

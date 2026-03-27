package ui

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// ParallelRegionSpec stores the public serializable input contract for one parallel region instance.
type ParallelRegionSpec[Props any] struct {
	RendererID       string
	RegionInstanceID string
	Props            Props
	SourceIDs        []string
}

type parallelRegionRendererEntry struct {
	getRender any
}

var (
	storeParallelRegionRendererMu   sync.RWMutex
	cacheParallelRegionRendererByID = map[runtime2.RendererID]parallelRegionRendererEntry{}
)

// buildParallelRegionRuntimeSpec converts one public parallel-region spec into the runtime2 contract.
func buildParallelRegionRuntimeSpec[Props any](parseSpec ParallelRegionSpec[Props]) (runtime2.ParallelRegionSpec, error) {
	getRuntimeSpec, parseErr := runtime2.NormalizeParallelRegionSpec(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID(parseSpec.RendererID),
		RegionInstanceID: runtime2.RegionInstanceID(parseSpec.RegionInstanceID),
		Props:            parseSpec.Props,
		SourceIDs:        append([]string(nil), parseSpec.SourceIDs...),
	})
	if parseErr != nil {
		return runtime2.ParallelRegionSpec{}, parseErr
	}
	return getRuntimeSpec, nil
}

// RegisterParallelRegion registers one public parallel-region renderer and bridges it into the runtime2 registry.
func RegisterParallelRegion[Props any](parseRendererID string, parseRender func(Props) Node) error {
	getRendererID, parseRendererIDErr := runtime2.ParseRendererID(parseRendererID)
	if parseRendererIDErr != nil {
		return parseRendererIDErr
	}
	if parseRender == nil {
		return fmt.Errorf("ui: parallel-region renderer is required")
	}
	storeParallelRegionRendererMu.Lock()
	defer storeParallelRegionRendererMu.Unlock()
	if _, hasParallelRegionRenderer := cacheParallelRegionRendererByID[getRendererID]; hasParallelRegionRenderer {
		return fmt.Errorf("ui: parallel-region renderer %q is already registered", getRendererID)
	}
	if parseErr := runtime2.RegisterRenderer(getRendererID, func() {}, runtime2.RendererMetadata{}); parseErr != nil {
		return parseErr
	}
	cacheParallelRegionRendererByID[getRendererID] = parallelRegionRendererEntry{
		getRender: parseRender,
	}
	return nil
}

// ParallelRegion renders one parallel-region shell with immediate local content and runtime2 shell markers.
func ParallelRegion[Props any](parseSpec ParallelRegionSpec[Props]) Node {
	getRuntimeSpec, parseRuntimeSpecErr := buildParallelRegionRuntimeSpec(parseSpec)
	if parseRuntimeSpecErr != nil {
		panic(fmt.Sprintf("ui: parallel-region spec is invalid: %v", parseRuntimeSpecErr))
	}
	getRender, parseResolveErr := resolveParallelRegionRenderer(string(getRuntimeSpec.RendererID))
	if parseResolveErr != nil {
		panic(fmt.Sprintf("ui: parallel-region renderer resolution failed: %v", parseResolveErr))
	}
	getChild, parseRenderErr := buildParallelRegionLocalNode(getRender, getRuntimeSpec.Props)
	if parseRenderErr != nil {
		panic(fmt.Sprintf("ui: parallel-region local render failed: %v", parseRenderErr))
	}
	getShellMarker, parseMarkerErr := runtime2.BuildSSRShellMarkerAttributeValue(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: getRuntimeSpec.RegionInstanceID,
		RendererID:       getRuntimeSpec.RendererID,
	})
	if parseMarkerErr != nil {
		panic(fmt.Sprintf("ui: parallel-region shell marker build failed: %v", parseMarkerErr))
	}
	getShellProps := map[string]interface{}{
		runtime2.SSRShellMarkerAttribute: getShellMarker,
	}
	if getChild == nil {
		return runtime.CreateElement("div", getShellProps)
	}
	return runtime.CreateElement("div", getShellProps, getChild)
}

// BuildParallelRegionSourceIDs flattens declared reactive sources into a stable public source-ID list.
func BuildParallelRegionSourceIDs(parseSources ...ReactiveSource) ([]string, error) {
	getSourceIDs := make([]string, 0, len(parseSources))
	parseSeenSourceIDs := make(map[string]bool, len(parseSources))
	for _, parseSource := range parseSources {
		if parseSource == nil {
			continue
		}
		for _, parseSourceID := range parseSource.ReactiveRegionSourceIDs() {
			getNormalizedSourceIDs, parseNormalizeErr := runtime2.NormalizeSourceIDs([]string{parseSourceID})
			if parseNormalizeErr != nil {
				return nil, parseNormalizeErr
			}
			if len(getNormalizedSourceIDs) == 0 {
				continue
			}
			getSourceID := getNormalizedSourceIDs[0]
			if parseSeenSourceIDs[getSourceID] {
				continue
			}
			parseSeenSourceIDs[getSourceID] = true
			getSourceIDs = append(getSourceIDs, getSourceID)
		}
	}
	return getSourceIDs, nil
}

// resolveParallelRegionRenderer resolves one registered public parallel-region renderer by stable ID.
func resolveParallelRegionRenderer(parseRendererID string) (any, error) {
	getRendererID, parseRendererIDErr := runtime2.ParseRendererID(parseRendererID)
	if parseRendererIDErr != nil {
		return nil, parseRendererIDErr
	}
	storeParallelRegionRendererMu.RLock()
	defer storeParallelRegionRendererMu.RUnlock()
	getParallelRegionRendererEntry, hasParallelRegionRenderer := cacheParallelRegionRendererByID[getRendererID]
	if !hasParallelRegionRenderer {
		return nil, fmt.Errorf("ui: parallel-region renderer %q is not registered", getRendererID)
	}
	return getParallelRegionRendererEntry.getRender, nil
}

// buildParallelRegionLocalNode invokes one registered public renderer with validated props and returns its local-first node.
func buildParallelRegionLocalNode(parseRender any, parseProps any) (Node, error) {
	getRenderValue := reflect.ValueOf(parseRender)
	if !getRenderValue.IsValid() || getRenderValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("ui: parallel-region renderer is not callable")
	}
	if getRenderValue.Type().NumIn() != 1 || getRenderValue.Type().NumOut() != 1 {
		return nil, fmt.Errorf("ui: parallel-region renderer must accept one props argument and return one ui.Node")
	}
	getArgValue, parseArgErr := buildParallelRegionRenderArg(getRenderValue.Type().In(0), parseProps)
	if parseArgErr != nil {
		return nil, parseArgErr
	}
	getResults := getRenderValue.Call([]reflect.Value{getArgValue})
	if len(getResults) != 1 {
		return nil, fmt.Errorf("ui: parallel-region renderer returned %d values", len(getResults))
	}
	if getResults[0].IsNil() {
		return nil, nil
	}
	getNode, hasNode := getResults[0].Interface().(Node)
	if !hasNode {
		return nil, fmt.Errorf("ui: parallel-region renderer returned %T, want ui.Node", getResults[0].Interface())
	}
	return getNode, nil
}

// buildParallelRegionRenderArg maps validated public props into the registered renderer input type.
func buildParallelRegionRenderArg(parseArgType reflect.Type, parseProps any) (reflect.Value, error) {
	if parseProps == nil {
		return reflect.Zero(parseArgType), nil
	}
	getPropsValue := reflect.ValueOf(parseProps)
	if getPropsValue.Type().AssignableTo(parseArgType) {
		return getPropsValue, nil
	}
	if getPropsValue.Type().ConvertibleTo(parseArgType) {
		return getPropsValue.Convert(parseArgType), nil
	}
	return reflect.Value{}, fmt.Errorf(
		"ui: parallel-region props type %s does not match registered renderer input %s",
		getPropsValue.Type(),
		parseArgType,
	)
}

// resetParallelRegionRegistry clears the public parallel-region registry and the bridged runtime2 registry for deterministic tests.
func resetParallelRegionRegistry() {
	storeParallelRegionRendererMu.Lock()
	defer storeParallelRegionRendererMu.Unlock()
	cacheParallelRegionRendererByID = map[runtime2.RendererID]parallelRegionRendererEntry{}
	runtime2.ResetRendererRegistry()
}

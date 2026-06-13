package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

type fakePersistStorage struct {
	parseValues map[string]string
	parseGetErr error
	parseSetErr error
}

type fakePersistedItem struct {
	Name  string
	Score int
}

func (parseS *fakePersistStorage) GetItem(parseKey string) (string, bool, error) {
	if parseS.parseGetErr != nil {
		return "", false, parseS.parseGetErr
	}
	parseValue, parseOK := parseS.parseValues[parseKey]
	return parseValue, parseOK, nil
}

func (parseS *fakePersistStorage) SetItem(parseKey string, parseValue string) error {
	if parseS.parseSetErr != nil {
		return parseS.parseSetErr
	}
	if parseS.parseValues == nil {
		parseS.parseValues = map[string]string{}
	}
	parseS.parseValues[parseKey] = parseValue
	return nil
}

type fakePersistState[T any] struct {
	parseValue T
}

func (parseS *fakePersistState[T]) Get() T {
	return parseS.parseValue
}

func (parseS *fakePersistState[T]) Set(parseValue T) {
	parseS.parseValue = parseValue
}

type fakePersistErrorState struct {
	parseErr error
}

func (parseS *fakePersistErrorState) Set(parseErr error) {
	parseS.parseErr = parseErr
}

type fakePersistEventTarget struct {
	parseListenErr error
	parseName      string
	parseHandler   func(interop.BrowserEvent)
}

func (parseT *fakePersistEventTarget) Listen(parseName string, parseHandler func(interop.BrowserEvent)) (interop.Subscription, error) {
	parseT.parseName = parseName
	parseT.parseHandler = parseHandler
	if parseT.parseListenErr != nil {
		return interop.Subscription{}, parseT.parseListenErr
	}
	return interop.Subscription{}, nil
}

func withFakePersistStorage(t *testing.T, parseStorage *fakePersistStorage, parseErr error) {
	t.Helper()
	parsePrev := getPersistStorage
	getPersistStorage = func(parseArea PersistStorageArea) (persistedStorage, error) {
		return parseStorage, parseErr
	}
	t.Cleanup(func() {
		getPersistStorage = parsePrev
	})
}

func withFakePersistWindowEvents(t *testing.T, parseTarget persistedEventTarget, parseErr error) {
	t.Helper()
	parsePrev := getPersistWindowEvents
	getPersistWindowEvents = func() (persistedEventTarget, error) {
		return parseTarget, parseErr
	}
	t.Cleanup(func() {
		getPersistWindowEvents = parsePrev
	})
}

func TestLoadStoredInitialBranches(t *testing.T) {
	withFakePersistStorage(t, &fakePersistStorage{parseValues: map[string]string{"profile": `{"Name":"Ada","Score":7}`}}, nil)
	parseLoaded := loadStoredInitial("profile", fakePersistedItem{Name: "initial"}, PersistLocal)
	if parseLoaded.Name != "Ada" || parseLoaded.Score != 7 {
		t.Fatalf("loaded value = %+v, want stored JSON", parseLoaded)
	}

	withFakePersistStorage(t, &fakePersistStorage{parseValues: map[string]string{}}, nil)
	if parseGot := loadStoredInitial("missing", "fallback", PersistLocal); parseGot != "fallback" {
		t.Fatalf("missing value = %q, want fallback", parseGot)
	}

	withFakePersistStorage(t, &fakePersistStorage{parseValues: map[string]string{"bad": "{"}}, nil)
	if parseGot := loadStoredInitial("bad", "fallback", PersistLocal); parseGot != "fallback" {
		t.Fatalf("corrupt value = %q, want fallback", parseGot)
	}

	withFakePersistStorage(t, &fakePersistStorage{parseGetErr: errors.New("read failed")}, nil)
	if parseGot := loadStoredInitial("bad", "fallback", PersistLocal); parseGot != "fallback" {
		t.Fatalf("read-error value = %q, want fallback", parseGot)
	}

	withFakePersistStorage(t, nil, errors.New("storage unavailable"))
	if parseGot := loadStoredInitial("bad", "fallback", PersistLocal); parseGot != "fallback" {
		t.Fatalf("unavailable value = %q, want fallback", parseGot)
	}
}

func TestWritePersistedStateValueBranches(t *testing.T) {
	parseStorage := &fakePersistStorage{}
	withFakePersistStorage(t, parseStorage, nil)
	parseValue := &fakePersistState[string]{}
	parseErrState := &fakePersistErrorState{}

	writePersistedStateValue("name", "Ada", PersistLocal, parseValue, parseErrState)
	if parseValue.parseValue != "Ada" {
		t.Fatalf("state value = %q, want Ada", parseValue.parseValue)
	}
	if parseStorage.parseValues["name"] != `"Ada"` {
		t.Fatalf("stored JSON = %q, want quoted Ada", parseStorage.parseValues["name"])
	}
	if parseErrState.parseErr != nil {
		t.Fatalf("error state = %v, want nil", parseErrState.parseErr)
	}

	parseSetErr := errors.New("quota")
	parseStorage.parseSetErr = parseSetErr
	writePersistedStateValue("name", "Grace", PersistSession, parseValue, parseErrState)
	if parseErrState.parseErr != parseSetErr {
		t.Fatalf("set error = %v, want quota", parseErrState.parseErr)
	}

	writePersistedStateValue("fn", func() {}, PersistLocal, &fakePersistState[func()]{}, parseErrState)
	if parseErrState.parseErr == nil || !strings.Contains(parseErrState.parseErr.Error(), "unsupported type") {
		t.Fatalf("marshal error = %v, want unsupported type", parseErrState.parseErr)
	}

	withFakePersistStorage(t, nil, errors.New("storage unavailable"))
	parseUnavailableState := &fakePersistState[int]{}
	parseUnavailableErr := &fakePersistErrorState{}
	writePersistedStateValue("count", 3, PersistLocal, parseUnavailableState, parseUnavailableErr)
	if parseUnavailableState.parseValue != 3 || parseUnavailableErr.parseErr != nil {
		t.Fatalf("unavailable write state=%+v err=%v, want in-memory update without error", parseUnavailableState, parseUnavailableErr.parseErr)
	}
}

func TestSubscribePersistedStorageSyncBranches(t *testing.T) {
	parseStorage := &fakePersistStorage{parseValues: map[string]string{"profile": `{"Name":"Ada","Score":7}`}}
	parseEvents := &fakePersistEventTarget{}
	withFakePersistStorage(t, parseStorage, nil)
	withFakePersistWindowEvents(t, parseEvents, nil)
	parseState := &fakePersistState[fakePersistedItem]{}

	parseCleanup := subscribePersistedStorageSync("profile", PersistLocal, parseState)
	if parseEvents.parseName != "storage" || parseEvents.parseHandler == nil {
		t.Fatalf("storage listener was not registered: %#v", parseEvents)
	}
	parseEvents.parseHandler(interop.BrowserEvent{Type: "storage"})
	if parseState.parseValue.Name != "Ada" || parseState.parseValue.Score != 7 {
		t.Fatalf("synced state = %+v, want stored JSON value", parseState.parseValue)
	}
	parseCleanup()

	parseStorage.parseValues["profile"] = "{"
	parseState.parseValue = fakePersistedItem{}
	parseEvents.parseHandler(interop.BrowserEvent{Type: "storage"})
	if parseState.parseValue != (fakePersistedItem{}) {
		t.Fatalf("corrupt storage event changed state: %+v", parseState.parseValue)
	}

	parseStorage.parseGetErr = errors.New("read failed")
	parseEvents.parseHandler(interop.BrowserEvent{Type: "storage"})
	if parseState.parseValue != (fakePersistedItem{}) {
		t.Fatalf("read-error storage event changed state: %+v", parseState.parseValue)
	}

	withFakePersistStorage(t, nil, errors.New("storage unavailable"))
	if parseCleanup := subscribePersistedStorageSync("profile", PersistLocal, parseState); parseCleanup == nil {
		t.Fatal("unavailable storage cleanup = nil, want no-op cleanup")
	}

	withFakePersistStorage(t, parseStorage, nil)
	withFakePersistWindowEvents(t, nil, errors.New("events unavailable"))
	if parseCleanup := subscribePersistedStorageSync("profile", PersistLocal, parseState); parseCleanup == nil {
		t.Fatal("unavailable events cleanup = nil, want no-op cleanup")
	}

	parseListenErrTarget := &fakePersistEventTarget{parseListenErr: errors.New("listen failed")}
	withFakePersistWindowEvents(t, parseListenErrTarget, nil)
	if parseCleanup := subscribePersistedStorageSync("profile", PersistLocal, parseState); parseCleanup == nil {
		t.Fatal("listen error cleanup = nil, want no-op cleanup")
	}
}

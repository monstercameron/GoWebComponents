//go:build js && wasm

package app

import (
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

type threadScrollMemory struct {
	shouldAutoScroll     func() bool
	showScrollToBottom   func() bool
	resetToBottomMode    func()
	followStream         func()
	scrollToBottom       func()
	cancelPendingPersist func()
	persistNow           func(int64)
	prepareRestore       func(int64)
}

func (parseM threadScrollMemory) ShouldAutoScroll() bool {
	if parseM.shouldAutoScroll == nil {
		return true
	}
	return parseM.shouldAutoScroll()
}

func (parseM threadScrollMemory) ResetToBottomMode() {
	if parseM.resetToBottomMode != nil {
		parseM.resetToBottomMode()
	}
}

func (parseM threadScrollMemory) ParseShowScrollToBottom() bool {
	if parseM.showScrollToBottom == nil {
		return false
	}
	return parseM.showScrollToBottom()
}

func (parseM threadScrollMemory) ParseFollowStream() {
	if parseM.followStream != nil {
		parseM.followStream()
	}
}

func (parseM threadScrollMemory) ParseScrollToBottom() {
	if parseM.scrollToBottom != nil {
		parseM.scrollToBottom()
	}
}

func (parseM threadScrollMemory) CancelPendingPersist() {
	if parseM.cancelPendingPersist != nil {
		parseM.cancelPendingPersist()
	}
}

func (parseM threadScrollMemory) ParsePersistNow(parseConvID int64) {
	if parseM.persistNow != nil {
		parseM.persistNow(parseConvID)
	}
}

func (parseM threadScrollMemory) ParsePrepareRestore(parseConvID int64) {
	if parseM.prepareRestore != nil {
		parseM.prepareRestore(parseConvID)
	}
}

func parseUseThreadScrollMemory(parseActiveConvID int64, parseMessageCount int) threadScrollMemory {
	parseUserHasScrolledRef := ui.UseRef(false)
	parseManualScrollIntentRef := ui.UseRef(false)
	parseAutoScrollInFlightRef := ui.UseRef(false)
	parseLastScrollTopRef := ui.UseRef(float64(0))
	parseScrollCacheRef := ui.UseRef(map[int64]float64{})
	parseScrollPersistTimerRef := ui.UseRef(interop.Timer{})
	parseStreamFollowTimerRef := ui.UseRef(interop.Timer{})
	parseStreamFollowHoldTimerRef := ui.UseRef(interop.Timer{})
	parseScrollToBottomSettleTimerRef := ui.UseRef(interop.Timer{})
	parseScrollVisibilityPollTimerRef := ui.UseRef(interop.Timer{})
	parsePendingScrollRestoreRef := ui.UseRef((*float64)(nil))
	parseShowScrollToBottomState := ui.UseState(false)

	parseSyncScrollToBottomVisibility := func() {
		parseHasScrollBelow := parseMessageListHasScrollBelow()
		if parseShowScrollToBottomState.Get() == parseHasScrollBelow {
			return
		}
		parseShowScrollToBottomState.Set(parseHasScrollBelow)
	}

	parseCancelPendingScrollPersist := func() {
		parseTimer := parseScrollPersistTimerRef.Get()
		if parseErr := parseTimer.Cancel(); parseErr == nil {
			parseScrollPersistTimerRef.Set(interop.Timer{})
		}
	}

	parseCancelPendingStreamFollow := func() {
		parseTimer2 := parseStreamFollowTimerRef.Get()
		if parseErr2 := parseTimer2.Cancel(); parseErr2 == nil {
			parseStreamFollowTimerRef.Set(interop.Timer{})
		}
	}

	parseCancelStreamFollowHold := func() {
		parseTimer3 := parseStreamFollowHoldTimerRef.Get()
		if parseErr3 := parseTimer3.Cancel(); parseErr3 == nil {
			parseStreamFollowHoldTimerRef.Set(interop.Timer{})
		}
	}

	parseCancelPendingScrollToBottomSettle := func() {
		parseTimer := parseScrollToBottomSettleTimerRef.Get()
		if parseErr := parseTimer.Cancel(); parseErr == nil {
			parseScrollToBottomSettleTimerRef.Set(interop.Timer{})
		}
	}

	parseCancelScrollVisibilityPoll := func() {
		parseTimer := parseScrollVisibilityPollTimerRef.Get()
		if parseErr := parseTimer.Cancel(); parseErr == nil {
			parseScrollVisibilityPollTimerRef.Set(interop.Timer{})
		}
	}

	parseMarkManualScrollIntent := func() {
		parseManualScrollIntentRef.Set(true)
		parseUserHasScrolledRef.Set(true)
		parseCancelPendingStreamFollow()
		parseCancelStreamFollowHold()
		parseCancelPendingScrollToBottomSettle()
		parseAutoScrollInFlightRef.Set(false)
	}

	parsePersistThreadScrollNow := func(parseConvID int64) {
		parseScrollTop, parseOk := parseMessageListScrollTop()
		if !parseOk {
			return
		}
		parseScrollCache := parseScrollCacheRef.Get()
		parseScrollCache[parseConvID] = parseScrollTop
		parseScrollCacheRef.Set(parseScrollCache)
	}

	parseScheduleThreadScrollPersist := func(parseConvID2 int64) {
		parseCancelPendingScrollPersist()
		parseTimer4, parseErr4 := interop.ScheduleTimeout(scrollSettleDelay, func() {
			parsePersistThreadScrollNow(parseConvID2)
			parseScrollPersistTimerRef.Set(interop.Timer{})
		})
		if parseErr4 != nil {
			parsePersistThreadScrollNow(parseConvID2)
			return
		}
		parseScrollPersistTimerRef.Set(parseTimer4)
	}

	parsePrepareThreadScrollRestore := func(parseConvID3 int64) {
		parseCancelPendingStreamFollow()
		parseCancelStreamFollowHold()
		parseCancelPendingScrollToBottomSettle()
		parseAutoScrollInFlightRef.Set(false)
		parseManualScrollIntentRef.Set(false)
		if parseScrollTop2, parseOk2 := parseScrollCacheRef.Get()[parseConvID3]; parseOk2 {
			parseValue := parseScrollTop2
			parsePendingScrollRestoreRef.Set(&parseValue)
			return
		}
		parsePendingScrollRestoreRef.Set(nil)
		parseUserHasScrolledRef.Set(false)
	}

	parseScheduleStreamFollow := func() {
		if parseUserHasScrolledRef.Get() {
			return
		}
		parseCancelPendingStreamFollow()
		parseTimer5, parseErr5 := interop.ScheduleTimeout(streamFollowDelay, func() {
			parseStreamFollowTimerRef.Set(interop.Timer{})
			if parseUserHasScrolledRef.Get() {
				return
			}
			parseCancelStreamFollowHold()
			parseAutoScrollInFlightRef.Set(true)
			parseManualScrollIntentRef.Set(false)
			parseScrollStreamingAssistantBubbleIntoView(scrollBehaviorSmooth)
			parseHoldTimer, parseHoldErr := interop.ScheduleTimeout(streamFollowHold, func() {
				parseStreamFollowHoldTimerRef.Set(interop.Timer{})
				parseAutoScrollInFlightRef.Set(false)
			})
			if parseHoldErr != nil {
				parseAutoScrollInFlightRef.Set(false)
				return
			}
			parseStreamFollowHoldTimerRef.Set(parseHoldTimer)
		})
		if parseErr5 != nil {
			parseAutoScrollInFlightRef.Set(true)
			parseScrollStreamingAssistantBubbleIntoView(scrollBehaviorSmooth)
			parseAutoScrollInFlightRef.Set(false)
			return
		}
		parseStreamFollowTimerRef.Set(parseTimer5)
	}

	ui.UseEffect(func() func() {
		if !parseUserHasScrolledRef.Get() {
			parseScrollMessageListToBottom(scrollBehaviorSmooth)
		}
		parseSyncScrollToBottomVisibility()
		return nil
	}, parseMessageCount)

	ui.UseEffect(func() func() {
		parsePendingScrollTop := parsePendingScrollRestoreRef.Get()
		if parsePendingScrollTop == nil {
			return nil
		}
		if setMessageListScrollTop(*parsePendingScrollTop) {
			parseUserHasScrolledRef.Set(!isMessageListAtScrollBottom())
			parseLastScrollTopRef.Set(*parsePendingScrollTop)
		}
		parsePendingScrollRestoreRef.Set(nil)
		parseSyncScrollToBottomVisibility()
		return nil
	}, parseActiveConvID, parseMessageCount)

	ui.UseEffect(func() func() {
		parseDoc, parseErr6 := interop.GetDocument()
		if parseErr6 != nil {
			return nil
		}
		parseEl, parseOk3, parseErr6 := parseDoc.ElementByID(idMessageList)
		if parseErr6 != nil || !parseOk3 {
			return nil
		}
		parseSub, parseErr6 := parseEl.Listen("scroll", func(_ interop.BrowserEvent) {
			parseScrollTop3, hasScrollTop := parseMessageListScrollTop()
			parsePreviousTop := parseLastScrollTopRef.Get()
			if hasScrollTop {
				parseLastScrollTopRef.Set(parseScrollTop3)
			}
			if parseAutoScrollInFlightRef.Get() && !parseManualScrollIntentRef.Get() && hasScrollTop && parseScrollTop3+1 < parsePreviousTop {
				// User scrolled upward while smooth auto-follow was animating.
				parseMarkManualScrollIntent()
			}
			if parseAutoScrollInFlightRef.Get() && !parseManualScrollIntentRef.Get() {
				if isMessageListAtScrollBottom() {
					parseUserHasScrolledRef.Set(false)
				}
				parseSyncScrollToBottomVisibility()
				return
			}
			parseManualScrollIntentRef.Set(false)
			if isMessageListAtScrollBottom() {
				parseUserHasScrolledRef.Set(false)
			} else {
				parseUserHasScrolledRef.Set(true)
				parseCancelPendingStreamFollow()
				parseCancelStreamFollowHold()
				parseAutoScrollInFlightRef.Set(false)
			}
			parseSyncScrollToBottomVisibility()
			parseScheduleThreadScrollPersist(parseActiveConvID)
		})
		if parseErr6 != nil {
			return nil
		}
		parseWheelSub, parseWheelErr := parseEl.Listen("wheel", func(_ interop.BrowserEvent) {
			parseMarkManualScrollIntent()
		})
		if parseWheelErr != nil {
			return func() { parseSub.Cancel() }
		}
		parseTouchSub, parseTouchErr := parseEl.Listen("touchmove", func(_ interop.BrowserEvent) {
			parseMarkManualScrollIntent()
		})
		if parseTouchErr != nil {
			return func() {
				parseWheelSub.Cancel()
				parseSub.Cancel()
			}
		}
		parsePointerSub, parsePointerErr := parseEl.Listen("pointerdown", func(_ interop.BrowserEvent) {
			if !parseAutoScrollInFlightRef.Get() {
				return
			}
			parseMarkManualScrollIntent()
		})
		if parsePointerErr != nil {
			return func() {
				parseTouchSub.Cancel()
				parseWheelSub.Cancel()
				parseSub.Cancel()
			}
		}
		return func() {
			parsePointerSub.Cancel()
			parseTouchSub.Cancel()
			parseWheelSub.Cancel()
			parseSub.Cancel()
		}
	}, true, parseActiveConvID, parseMessageCount)

	ui.UseEffect(func() func() {
		parseCancelScrollVisibilityPoll()
		var parseScheduleVisibilityPoll func()
		parseScheduleVisibilityPoll = func() {
			parseTimer, parseErr := interop.ScheduleTimeout(scrollSettleDelay, func() {
				parseScrollVisibilityPollTimerRef.Set(interop.Timer{})
				parseSyncScrollToBottomVisibility()
				parseScheduleVisibilityPoll()
			})
			if parseErr != nil {
				return
			}
			parseScrollVisibilityPollTimerRef.Set(parseTimer)
		}
		parseSyncScrollToBottomVisibility()
		parseScheduleVisibilityPoll()
		return func() {
			parseCancelScrollVisibilityPoll()
		}
	}, true, parseActiveConvID)

	ui.UseEffect(func() func() {
		return func() {
			parseCancelPendingScrollPersist()
			parseCancelPendingStreamFollow()
			parseCancelStreamFollowHold()
			parseCancelPendingScrollToBottomSettle()
			parseCancelScrollVisibilityPoll()
		}
	}, true)

	return threadScrollMemory{
		shouldAutoScroll: func() bool {
			return !parseUserHasScrolledRef.Get()
		},
		showScrollToBottom: func() bool {
			return parseShowScrollToBottomState.Get()
		},
		resetToBottomMode: func() {
			parseCancelPendingStreamFollow()
			parseCancelStreamFollowHold()
			parseCancelPendingScrollToBottomSettle()
			parseAutoScrollInFlightRef.Set(false)
			parseManualScrollIntentRef.Set(false)
			parseUserHasScrolledRef.Set(false)
			parseShowScrollToBottomState.Set(false)
		},
		followStream: parseScheduleStreamFollow,
		scrollToBottom: func() {
			parseCancelPendingStreamFollow()
			parseCancelStreamFollowHold()
			parseCancelPendingScrollToBottomSettle()
			parseAutoScrollInFlightRef.Set(false)
			parseManualScrollIntentRef.Set(false)
			parseUserHasScrolledRef.Set(false)
			parseScrollMessageListToBottom(scrollBehaviorSmooth)
			parseSettleTimer, parseSettleErr := interop.ScheduleTimeout(scrollSettleDelay, func() {
				parseScrollToBottomSettleTimerRef.Set(interop.Timer{})
				parseScrollMessageListToBottom()
				parseSyncScrollToBottomVisibility()
			})
			if parseSettleErr != nil {
				parseScrollMessageListToBottom()
				parseSyncScrollToBottomVisibility()
				return
			}
			parseScrollToBottomSettleTimerRef.Set(parseSettleTimer)
			parseShowScrollToBottomState.Set(false)
		},
		cancelPendingPersist: parseCancelPendingScrollPersist,
		persistNow:           parsePersistThreadScrollNow,
		prepareRestore:       parsePrepareThreadScrollRestore,
	}
}

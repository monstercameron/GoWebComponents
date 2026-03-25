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

func (m threadScrollMemory) ShouldAutoScroll() bool {
	if m.shouldAutoScroll == nil {
		return true
	}
	return m.shouldAutoScroll()
}

func (m threadScrollMemory) ResetToBottomMode() {
	if m.resetToBottomMode != nil {
		m.resetToBottomMode()
	}
}

func (m threadScrollMemory) ShowScrollToBottom() bool {
	if m.showScrollToBottom == nil {
		return false
	}
	return m.showScrollToBottom()
}

func (m threadScrollMemory) FollowStream() {
	if m.followStream != nil {
		m.followStream()
	}
}

func (m threadScrollMemory) ScrollToBottom() {
	if m.scrollToBottom != nil {
		m.scrollToBottom()
	}
}

func (m threadScrollMemory) CancelPendingPersist() {
	if m.cancelPendingPersist != nil {
		m.cancelPendingPersist()
	}
}

func (m threadScrollMemory) PersistNow(convID int64) {
	if m.persistNow != nil {
		m.persistNow(convID)
	}
}

func (m threadScrollMemory) PrepareRestore(convID int64) {
	if m.prepareRestore != nil {
		m.prepareRestore(convID)
	}
}

func useThreadScrollMemory(activeConvID int64, messageCount int) threadScrollMemory {
	userHasScrolledRef := ui.UseRef(false)
	manualScrollIntentRef := ui.UseRef(false)
	autoScrollInFlightRef := ui.UseRef(false)
	lastScrollTopRef := ui.UseRef(float64(0))
	scrollCacheRef := ui.UseRef(map[int64]float64{})
	scrollPersistTimerRef := ui.UseRef(interop.Timer{})
	streamFollowTimerRef := ui.UseRef(interop.Timer{})
	streamFollowHoldTimerRef := ui.UseRef(interop.Timer{})
	pendingScrollRestoreRef := ui.UseRef((*float64)(nil))
	showScrollToBottomState := ui.UseState(false)

	syncScrollToBottomVisibility := func() {
		showScrollToBottomState.Set(messageListHasScrollBelow())
	}

	cancelPendingScrollPersist := func() {
		timer := scrollPersistTimerRef.Get()
		if err := timer.Cancel(); err == nil {
			scrollPersistTimerRef.Set(interop.Timer{})
		}
	}

	cancelPendingStreamFollow := func() {
		timer := streamFollowTimerRef.Get()
		if err := timer.Cancel(); err == nil {
			streamFollowTimerRef.Set(interop.Timer{})
		}
	}

	cancelStreamFollowHold := func() {
		timer := streamFollowHoldTimerRef.Get()
		if err := timer.Cancel(); err == nil {
			streamFollowHoldTimerRef.Set(interop.Timer{})
		}
	}

	markManualScrollIntent := func() {
		manualScrollIntentRef.Set(true)
		userHasScrolledRef.Set(true)
		cancelPendingStreamFollow()
		cancelStreamFollowHold()
		autoScrollInFlightRef.Set(false)
	}

	persistThreadScrollNow := func(convID int64) {
		scrollTop, ok := messageListScrollTop()
		if !ok {
			return
		}
		scrollCache := scrollCacheRef.Get()
		scrollCache[convID] = scrollTop
		scrollCacheRef.Set(scrollCache)
	}

	scheduleThreadScrollPersist := func(convID int64) {
		cancelPendingScrollPersist()
		timer, err := interop.ScheduleTimeout(scrollSettleDelay, func() {
			persistThreadScrollNow(convID)
			scrollPersistTimerRef.Set(interop.Timer{})
		})
		if err != nil {
			persistThreadScrollNow(convID)
			return
		}
		scrollPersistTimerRef.Set(timer)
	}

	prepareThreadScrollRestore := func(convID int64) {
		cancelPendingStreamFollow()
		cancelStreamFollowHold()
		autoScrollInFlightRef.Set(false)
		manualScrollIntentRef.Set(false)
		if scrollTop, ok := scrollCacheRef.Get()[convID]; ok {
			value := scrollTop
			pendingScrollRestoreRef.Set(&value)
			return
		}
		pendingScrollRestoreRef.Set(nil)
		userHasScrolledRef.Set(false)
	}

	scheduleStreamFollow := func() {
		if userHasScrolledRef.Get() {
			return
		}
		cancelPendingStreamFollow()
		timer, err := interop.ScheduleTimeout(streamFollowDelay, func() {
			streamFollowTimerRef.Set(interop.Timer{})
			if userHasScrolledRef.Get() {
				return
			}
			cancelStreamFollowHold()
			autoScrollInFlightRef.Set(true)
			manualScrollIntentRef.Set(false)
			scrollStreamingAssistantBubbleIntoView(scrollBehaviorSmooth)
			holdTimer, holdErr := interop.ScheduleTimeout(streamFollowHold, func() {
				streamFollowHoldTimerRef.Set(interop.Timer{})
				autoScrollInFlightRef.Set(false)
			})
			if holdErr != nil {
				autoScrollInFlightRef.Set(false)
				return
			}
			streamFollowHoldTimerRef.Set(holdTimer)
		})
		if err != nil {
			autoScrollInFlightRef.Set(true)
			scrollStreamingAssistantBubbleIntoView(scrollBehaviorSmooth)
			autoScrollInFlightRef.Set(false)
			return
		}
		streamFollowTimerRef.Set(timer)
	}

	ui.UseEffect(func() func() {
		if !userHasScrolledRef.Get() {
			scrollMessageListToBottom(scrollBehaviorSmooth)
		}
		syncScrollToBottomVisibility()
		return nil
	}, messageCount)

	ui.UseEffect(func() func() {
		pendingScrollTop := pendingScrollRestoreRef.Get()
		if pendingScrollTop == nil {
			return nil
		}
		if setMessageListScrollTop(*pendingScrollTop) {
			userHasScrolledRef.Set(!isMessageListAtScrollBottom())
			lastScrollTopRef.Set(*pendingScrollTop)
		}
		pendingScrollRestoreRef.Set(nil)
		syncScrollToBottomVisibility()
		return nil
	}, activeConvID, messageCount)

	ui.UseEffect(func() func() {
		doc, err := interop.CurrentDocument()
		if err != nil {
			return nil
		}
		el, ok, err := doc.ElementByID(idMessageList)
		if err != nil || !ok {
			return nil
		}
		sub, err := el.Listen("scroll", func(_ interop.BrowserEvent) {
			scrollTop, hasScrollTop := messageListScrollTop()
			previousTop := lastScrollTopRef.Get()
			if hasScrollTop {
				lastScrollTopRef.Set(scrollTop)
			}
			if autoScrollInFlightRef.Get() && !manualScrollIntentRef.Get() && hasScrollTop && scrollTop+1 < previousTop {
				// User scrolled upward while smooth auto-follow was animating.
				markManualScrollIntent()
			}
			if autoScrollInFlightRef.Get() && !manualScrollIntentRef.Get() {
				if isMessageListAtScrollBottom() {
					userHasScrolledRef.Set(false)
				}
				syncScrollToBottomVisibility()
				return
			}
			manualScrollIntentRef.Set(false)
			if isMessageListAtScrollBottom() {
				userHasScrolledRef.Set(false)
			} else {
				userHasScrolledRef.Set(true)
				cancelPendingStreamFollow()
				cancelStreamFollowHold()
				autoScrollInFlightRef.Set(false)
			}
			syncScrollToBottomVisibility()
			scheduleThreadScrollPersist(activeConvID)
		})
		if err != nil {
			return nil
		}
		wheelSub, wheelErr := el.Listen("wheel", func(_ interop.BrowserEvent) {
			markManualScrollIntent()
		})
		if wheelErr != nil {
			return func() { sub.Cancel() }
		}
		touchSub, touchErr := el.Listen("touchmove", func(_ interop.BrowserEvent) {
			markManualScrollIntent()
		})
		if touchErr != nil {
			return func() {
				wheelSub.Cancel()
				sub.Cancel()
			}
		}
		pointerSub, pointerErr := el.Listen("pointerdown", func(_ interop.BrowserEvent) {
			if !autoScrollInFlightRef.Get() {
				return
			}
			markManualScrollIntent()
		})
		if pointerErr != nil {
			return func() {
				touchSub.Cancel()
				wheelSub.Cancel()
				sub.Cancel()
			}
		}
		return func() {
			pointerSub.Cancel()
			touchSub.Cancel()
			wheelSub.Cancel()
			sub.Cancel()
		}
	}, true, activeConvID)

	ui.UseEffect(func() func() {
		return func() {
			cancelPendingScrollPersist()
			cancelPendingStreamFollow()
			cancelStreamFollowHold()
		}
	}, true)

	return threadScrollMemory{
		shouldAutoScroll: func() bool {
			return !userHasScrolledRef.Get()
		},
		showScrollToBottom: func() bool {
			return showScrollToBottomState.Get()
		},
		resetToBottomMode: func() {
			cancelPendingStreamFollow()
			cancelStreamFollowHold()
			autoScrollInFlightRef.Set(false)
			manualScrollIntentRef.Set(false)
			userHasScrolledRef.Set(false)
			showScrollToBottomState.Set(false)
		},
		followStream: scheduleStreamFollow,
		scrollToBottom: func() {
			cancelPendingStreamFollow()
			cancelStreamFollowHold()
			autoScrollInFlightRef.Set(false)
			manualScrollIntentRef.Set(false)
			userHasScrolledRef.Set(false)
			scrollMessageListToBottom(scrollBehaviorSmooth)
			showScrollToBottomState.Set(false)
		},
		cancelPendingPersist: cancelPendingScrollPersist,
		persistNow:           persistThreadScrollNow,
		prepareRestore:       prepareThreadScrollRestore,
	}
}

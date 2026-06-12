(function () {
    const storeBenchmarkScenarioConfigs = [
        {
            id: "core-render",
            label: "Core Render",
            category: "Initial Render",
            requestedWork: "Render 40 visible core-list rows from empty state.",
            correctnessCheck: "40 .benchmark-core-item nodes must exist.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-clear");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 0, 5000, "clear core items");
            },
            async run() {
                handleSubjectClick("#btn-core-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 40, 5000, "render core items");
            }
        },
        {
            id: "core-update",
            label: "Core Update",
            category: "Targeted Update",
            requestedWork: "Update the visible text content of the existing 40 core-list rows.",
            correctnessCheck: "All .benchmark-core-item nodes must include '(Updated)' and item count must stay at 40.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 40, 5000, "prepare core items");
            },
            async run() {
                handleSubjectClick("#btn-core-update");
                await handleSubjectWait(() => {
                    const getItems = Array.from(document.querySelectorAll(".benchmark-core-item"));
                    return getItems.length === 40 && getItems.every((parseNode) => parseNode.textContent.includes("(Updated)"));
                }, 5000, "update core items");
            }
        },
        {
            id: "core-stress-update",
            label: "Core Stress Update",
            category: "Targeted Update",
            requestedWork: "Update the visible text content of the existing 240-row stress core list.",
            correctnessCheck: "All 240 .benchmark-core-item nodes must include '(Updated)' and row count must stay at 240.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-stress-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 240, 5000, "prepare core stress update");
            },
            async run() {
                handleSubjectClick("#btn-core-update");
                await handleSubjectWait(() => {
                    const getItems = Array.from(document.querySelectorAll(".benchmark-core-item"));
                    return getItems.length === 240 && getItems.every((parseNode) => parseNode.textContent.includes("(Updated)"));
                }, 5000, "update core stress items");
            }
        },
        {
            id: "core-refresh",
            label: "Core Refresh",
            category: "Refresh",
            requestedWork: "Refresh the current core-list view without changing row count.",
            correctnessCheck: "40 .benchmark-core-item nodes must remain rendered and the active core-list refresh token must change after the refresh action.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 40, 5000, "prepare core refresh");
                return {
                    getRefreshToken: getSubjectContainerRefreshToken()
                };
            },
            async run(parseContext) {
                handleSubjectClick("#btn-refresh");
                await handleSubjectWait(() => {
                    return document.querySelectorAll(".benchmark-core-item").length === 40 &&
                        getSubjectContainerRefreshToken() !== parseContext.getRefreshToken;
                }, 5000, "refresh current view");
            }
        },
        {
            id: "core-append",
            label: "Core Append",
            category: "Structural Churn",
            requestedWork: "Append 100 rows after a 240-row stress core list.",
            correctnessCheck: "340 .benchmark-core-item nodes must exist and the last row-id must become 1100.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-stress-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 240, 5000, "prepare core append");
            },
            async run() {
                handleSubjectClick("#btn-core-append");
                await handleSubjectWait(() => {
                    const getRowIDs = buildSubjectRowIDs(".benchmark-core-item");
                    return getRowIDs.length === 340 && getRowIDs[getRowIDs.length - 1] === 1100;
                }, 5000, "append core rows");
            }
        },
        {
            id: "core-filter",
            label: "Core Filter",
            category: "Structural Churn",
            requestedWork: "Filter the 240-row stress core list down to rows whose stable ID is divisible by 3.",
            correctnessCheck: "80 .benchmark-core-item nodes must remain and every row-id must be divisible by 3.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-stress-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 240, 5000, "prepare core filter");
            },
            async run() {
                handleSubjectClick("#btn-core-filter");
                await handleSubjectWait(() => {
                    const getRowIDs = buildSubjectRowIDs(".benchmark-core-item");
                    return getRowIDs.length === 80 && getRowIDs.every((parseRowID) => parseRowID % 3 === 0);
                }, 5000, "filter core rows");
            }
        },
        {
            id: "content-render",
            label: "Content Render",
            category: "Initial Render",
            requestedWork: "Render 12 nested content cards from empty state.",
            correctnessCheck: "12 .benchmark-content-card nodes must exist.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-content-clear");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-content-card").length === 0, 5000, "clear content cards");
            },
            async run() {
                handleSubjectClick("#btn-content-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-content-card").length === 12, 5000, "render content cards");
            }
        },
        {
            id: "content-update",
            label: "Content Update",
            category: "Targeted Update",
            requestedWork: "Update the text and status fields of the 12 rendered content cards.",
            correctnessCheck: "All .benchmark-content-status values must become 'live' and titles must include '(Updated)'.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-content-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-content-card").length === 12, 5000, "prepare content cards");
            },
            async run() {
                handleSubjectClick("#btn-content-update");
                await handleSubjectWait(() => {
                    const getStatuses = Array.from(document.querySelectorAll(".benchmark-content-status"));
                    const getTitles = Array.from(document.querySelectorAll(".benchmark-content-title"));
                    return getStatuses.length === 12 &&
                        getTitles.length === 12 &&
                        getStatuses.every((parseNode) => parseNode.textContent.trim() === "live") &&
                        getTitles.every((parseNode) => parseNode.textContent.includes("(Updated)"));
                }, 5000, "update content cards");
            }
        },
        {
            id: "content-refresh",
            label: "Content Refresh",
            category: "Refresh",
            requestedWork: "Refresh the current content-card view without changing card count.",
            correctnessCheck: "12 .benchmark-content-card nodes must remain rendered and the active content refresh token must change after the refresh action.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-content-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-content-card").length === 12, 5000, "prepare content refresh");
                return {
                    getRefreshToken: getSubjectContainerRefreshToken()
                };
            },
            async run(parseContext) {
                handleSubjectClick("#btn-refresh");
                await handleSubjectWait(() => {
                    return document.querySelectorAll(".benchmark-content-card").length === 12 &&
                        getSubjectContainerRefreshToken() !== parseContext.getRefreshToken;
                }, 5000, "refresh content view");
            }
        },
        {
            id: "primitive-render",
            label: "Primitive Render",
            category: "Primitive Render",
            requestedWork: "Render 200 flat primitive host nodes from empty state.",
            correctnessCheck: "200 .benchmark-primitive-row nodes must exist after the render action.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-primitive-clear");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-primitive-row").length === 0, 5000, "clear primitive rows");
            },
            async run() {
                handleSubjectClick("#btn-primitive-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-primitive-row").length === 200, 5000, "render primitive rows");
            }
        },
        {
            id: "primitive-text-update",
            label: "Primitive Text Update",
            category: "Primitive Update",
            requestedWork: "Update only the text content of the existing 200 primitive host nodes.",
            correctnessCheck: "All .benchmark-primitive-label nodes must include '(Live)' while row count stays at 200.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-primitive-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-primitive-row").length === 200, 5000, "prepare primitive text update");
            },
            async run() {
                handleSubjectClick("#btn-primitive-text-update");
                await handleSubjectWait(() => {
                    const getLabels = Array.from(document.querySelectorAll(".benchmark-primitive-label"));
                    return getLabels.length === 200 && getLabels.every((parseNode) => parseNode.textContent.includes("(Live)"));
                }, 5000, "update primitive text");
            }
        },
        {
            id: "primitive-attribute-update",
            label: "Primitive Attribute Update",
            category: "Primitive Update",
            requestedWork: "Update only the attributes and classes of the existing 200 primitive host nodes.",
            correctnessCheck: "All .benchmark-primitive-row nodes must keep count 200 and switch to data-primitive-state='active'.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-primitive-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-primitive-row").length === 200, 5000, "prepare primitive attribute update");
            },
            async run() {
                handleSubjectClick("#btn-primitive-attr-update");
                await handleSubjectWait(() => {
                    const getRows = Array.from(document.querySelectorAll(".benchmark-primitive-row"));
                    return getRows.length === 200 && getRows.every((parseNode) => (parseNode.getAttribute("data-primitive-state") || "").trim() === "active");
                }, 5000, "update primitive attributes");
            }
        },
        {
            id: "primitive-append",
            label: "Primitive Append",
            category: "Primitive Churn",
            requestedWork: "Append 100 primitive host nodes after the initial 200-row primitive grid.",
            correctnessCheck: "300 .benchmark-primitive-row nodes must exist and the last data-primitive-id must become 300.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-primitive-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-primitive-row").length === 200, 5000, "prepare primitive append");
            },
            async run() {
                handleSubjectClick("#btn-primitive-append");
                await handleSubjectWait(() => {
                    const getRows = Array.from(document.querySelectorAll(".benchmark-primitive-row"));
                    const getLastRowID = Number.parseInt(getRows[getRows.length - 1]?.getAttribute("data-primitive-id") || "0", 10) || 0;
                    return getRows.length === 300 && getLastRowID === 300;
                }, 5000, "append primitive rows");
            }
        },
        {
            id: "primitive-remove",
            label: "Primitive Remove",
            category: "Primitive Churn",
            requestedWork: "Remove 100 primitive host nodes from the trailing edge of the 200-row primitive grid.",
            correctnessCheck: "100 .benchmark-primitive-row nodes must remain and the last data-primitive-id must become 100.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-primitive-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-primitive-row").length === 200, 5000, "prepare primitive remove");
            },
            async run() {
                handleSubjectClick("#btn-primitive-remove");
                await handleSubjectWait(() => {
                    const getRows = Array.from(document.querySelectorAll(".benchmark-primitive-row"));
                    const getLastRowID = Number.parseInt(getRows[getRows.length - 1]?.getAttribute("data-primitive-id") || "0", 10) || 0;
                    return getRows.length === 100 && getLastRowID === 100;
                }, 5000, "remove primitive rows");
            }
        },
        {
            id: "deep-render",
            label: "Deep Tree Render",
            category: "Initial Render",
            requestedWork: "Render the recursive deep-tree benchmark view until the benchmark leaf becomes visible.",
            correctnessCheck: "#benchmark-deep-leaf must exist after the render action.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-clear");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 0, 5000, "prepare deep tree");
            },
            async run() {
                handleSubjectClick("#btn-deep-render");
                await handleSubjectWait(() => !!document.querySelector("#benchmark-deep-leaf"), 5000, "render deep tree");
            }
        },
        {
            id: "deep-update",
            label: "Deep Tree Update",
            category: "Targeted Update",
            requestedWork: "Update the 60-level compliance tree in place so every nested level carries the new revision marker.",
            correctnessCheck: "The deep-tree root and every nested level must keep the same shape while the deep-tree revision marker changes.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-deep-render");
                await handleSubjectWait(() => !!document.querySelector("#benchmark-deep-root") && !!document.querySelector("#benchmark-deep-leaf"), 5000, "prepare deep update");
                return {
                    getRefreshToken: getSubjectDeepTreeRefreshToken()
                };
            },
            async run(parseContext) {
                handleSubjectClick("#btn-deep-update");
                await handleSubjectWait(() => {
                    const getNodes = Array.from(document.querySelectorAll(".benchmark-deep-node"));
                    const getRootNode = document.querySelector("#benchmark-deep-root");
                    const getLeafNode = document.querySelector("#benchmark-deep-leaf");
                    return !!getRootNode &&
                        !!getLeafNode &&
                        getNodes.length === 60 &&
                        getSubjectDeepTreeRefreshToken() !== parseContext.getRefreshToken;
                }, 5000, "update deep tree");
            }
        },
        {
            id: "deep-refresh",
            label: "Deep Tree Refresh",
            category: "Refresh",
            requestedWork: "Refresh the current deep-tree view without changing depth while the deep-tree refresh marker changes.",
            correctnessCheck: "The deep-tree root and leaf must remain rendered and the deep-tree refresh token must change after refresh.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-deep-render");
                await handleSubjectWait(() => !!document.querySelector("#benchmark-deep-root") && !!document.querySelector("#benchmark-deep-leaf"), 5000, "prepare deep refresh");
                return {
                    getRefreshToken: getSubjectDeepTreeRefreshToken()
                };
            },
            async run(parseContext) {
                handleSubjectClick("#btn-refresh");
                await handleSubjectWait(() => {
                    return !!document.querySelector("#benchmark-deep-root") &&
                        !!document.querySelector("#benchmark-deep-leaf") &&
                        document.querySelectorAll(".benchmark-deep-node").length === 60 &&
                        getSubjectDeepTreeRefreshToken() !== parseContext.getRefreshToken;
                }, 5000, "refresh deep tree");
            }
        },
        {
            id: "enterprise-subtree-update",
            label: "Enterprise Subtree Update",
            category: "Targeted Update",
            requestedWork: "Update one nested enterprise workspace section in place while preserving sibling sections and record count.",
            correctnessCheck: "6 enterprise sections and 30 records must remain rendered while the target section revision and statuses change in place.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-enterprise-render");
                await handleSubjectWait(() => {
                    return document.querySelectorAll(".benchmark-enterprise-section").length === 6 &&
                        document.querySelectorAll(".benchmark-enterprise-record").length === 30;
                }, 5000, "prepare enterprise subtree update");
                return {
                    getRefreshToken: getSubjectEnterpriseRefreshToken()
                };
            },
            async run(parseContext) {
                handleSubjectClick("#btn-enterprise-update");
                await handleSubjectWait(() => {
                    const getTargetRecords = Array.from(document.querySelectorAll('.benchmark-enterprise-record[data-section-id="section-3"]'));
                    return document.querySelectorAll(".benchmark-enterprise-section").length === 6 &&
                        document.querySelectorAll(".benchmark-enterprise-record").length === 30 &&
                        getTargetRecords.length === 5 &&
                        getSubjectEnterpriseRefreshToken() !== parseContext.getRefreshToken;
                }, 5000, "update enterprise subtree");
            }
        },
        {
            id: "hooks-render",
            label: "Hook Grid Render",
            category: "Initial Render",
            requestedWork: "Render the 40-cell hook-heavy grid view.",
            correctnessCheck: "Exactly 40 .benchmark-hook-node elements must exist after the render action.",
            finishLine: "DOM-ready plus next requestAnimationFrame paint proxy.",
            async prepare() {
                handleSubjectClick("#btn-core-clear");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-core-item").length === 0, 5000, "prepare hook grid");
            },
            async run() {
                handleSubjectClick("#btn-hooks-render");
                await handleSubjectWait(() => document.querySelectorAll(".benchmark-hook-node").length === 40, 5000, "render hook grid");
            }
        }
    ];

    function getSubjectNumber(parseSelector) {
        const getNode = document.querySelector(parseSelector);
        if (!getNode) {
            return 0;
        }
        const getMatch = getNode.textContent.match(/-?\d+(?:\.\d+)?/);
        if (!getMatch) {
            return 0;
        }
        return Number.parseFloat(getMatch[0]);
    }

    function handleSubjectClick(parseSelector) {
        const getNode = document.querySelector(parseSelector);
        if (!getNode) {
            throw new Error("missing benchmark control " + parseSelector);
        }
        getNode.click();
    }

    function buildSubjectRowIDs(parseSelector) {
        return Array.from(document.querySelectorAll(parseSelector)).map((parseNode) => Number.parseInt(parseNode.getAttribute("data-row-id") || "0", 10) || 0);
    }

    function getSubjectContainerRefreshToken() {
        const getNode = document.querySelector("#core-list-container, #content-container, #hooks-container, #benchmark-deep-leaf");
        if (!getNode) {
            return "";
        }
        return (getNode.getAttribute("data-refresh-token") || "").trim();
    }

    function getSubjectDeepTreeVersion() {
        const getNode = document.querySelector("#benchmark-deep-root");
        if (!getNode) {
            return "";
        }
        return (getNode.getAttribute("data-tree-version") || "").trim();
    }

    function getSubjectDeepTreeRefreshToken() {
        const getNode = document.querySelector("#benchmark-deep-root");
        if (!getNode) {
            return "";
        }
        return (getNode.getAttribute("data-refresh-token") || "").trim();
    }

    function getSubjectEnterpriseRevision(parseSectionID) {
        const getNode = document.querySelector(`.benchmark-enterprise-section[data-section-id="${parseSectionID}"]`);
        if (!getNode) {
            return "";
        }
        return (getNode.getAttribute("data-enterprise-revision") || "").trim();
    }

    function getSubjectEnterpriseRefreshToken() {
        const getNode = document.querySelector("#enterprise-container");
        if (!getNode) {
            return "";
        }
        return (getNode.getAttribute("data-refresh-token") || "").trim();
    }

    function hasSubjectAscendingRowIDs(parseRowIDs) {
        for (let parseIndex = 1; parseIndex < parseRowIDs.length; parseIndex++) {
            if (parseRowIDs[parseIndex - 1] >= parseRowIDs[parseIndex]) {
                return false;
            }
        }
        return true;
    }

    function hasSubjectDescendingRowIDs(parseRowIDs) {
        for (let parseIndex = 1; parseIndex < parseRowIDs.length; parseIndex++) {
            if (parseRowIDs[parseIndex - 1] <= parseRowIDs[parseIndex]) {
                return false;
            }
        }
        return true;
    }

    function buildSubjectOrderingBreak(parseRowIDs, parseDirection) {
        const getDirection = String(parseDirection || "").trim().toLowerCase();
        for (let parseIndex = 1; parseIndex < parseRowIDs.length; parseIndex++) {
            const getPreviousRowID = parseRowIDs[parseIndex - 1];
            const getCurrentRowID = parseRowIDs[parseIndex];
            if (getDirection === "ascending" && getPreviousRowID < getCurrentRowID) {
                continue;
            }
            if (getDirection === "descending" && getPreviousRowID > getCurrentRowID) {
                continue;
            }
            return {
                getIndex: parseIndex,
                getPreviousRowID: getPreviousRowID,
                getCurrentRowID: getCurrentRowID,
                getWindow: parseRowIDs.slice(Math.max(0, parseIndex - 3), Math.min(parseRowIDs.length, parseIndex + 3))
            };
        }
        return null;
    }

    function handleSubjectDelay(parseDurationMs) {
        return new Promise((parseResolve) => {
            window.setTimeout(parseResolve, parseDurationMs);
        });
    }

    function handleSubjectTaskTick() {
        return new Promise((parseResolve) => {
            if (typeof MessageChannel === "function") {
                const getChannel = new MessageChannel();
                getChannel.port1.onmessage = () => {
                    getChannel.port1.close();
                    getChannel.port2.close();
                    parseResolve();
                };
                getChannel.port2.postMessage(0);
                return;
            }
            window.setTimeout(parseResolve, 0);
        });
    }

    function handleSubjectAnimationFrames(parseCount) {
        return new Promise((parseResolve) => {
            function handleSubjectNext(parseRemaining) {
                if (parseRemaining <= 0) {
                    parseResolve();
                    return;
                }
                window.requestAnimationFrame(() => {
                    handleSubjectNext(parseRemaining - 1);
                });
            }
            handleSubjectNext(parseCount);
        });
    }

    async function handleSubjectWaitForDOMSettle(parseStableFrames, parseMaxFrames) {
        const getRequiredStableFrames = Math.max(1, Number.parseInt(String(parseStableFrames ?? "2"), 10) || 2);
        const getMaxFrames = Math.max(getRequiredStableFrames, Number.parseInt(String(parseMaxFrames ?? "30"), 10) || 30);
        const getContainerNode = document.querySelector("#benchmark-container") || document.querySelector("#benchmark-app") || document.body;
        let hasSubjectMutation = false;
        const getObserver = new MutationObserver(() => {
            hasSubjectMutation = true;
        });
        getObserver.observe(getContainerNode, {
            childList: true,
            subtree: true,
            attributes: true,
            characterData: true
        });
        try {
            let getStableFrameCount = 0;
            let getObservedFrameCount = 0;
            while (getStableFrameCount < getRequiredStableFrames && getObservedFrameCount < getMaxFrames) {
                hasSubjectMutation = false;
                await handleSubjectAnimationFrames(1);
                getObservedFrameCount += 1;
                if (hasSubjectMutation) {
                    getStableFrameCount = 0;
                    continue;
                }
                getStableFrameCount += 1;
            }
        } finally {
            getObserver.disconnect();
        }
    }

    async function handleSubjectWait(parsePredicate, parseTimeoutMs, parseLabel) {
        const getStartedAt = performance.now();
        while ((performance.now() - getStartedAt) < parseTimeoutMs) {
            let isReady = false;
            try {
                isReady = !!parsePredicate();
            } catch (parseErr) {
                window.__example201SubjectError = String(parseErr);
                throw parseErr;
            }
            if (isReady) {
                return;
            }
            await handleSubjectTaskTick();
        }
        const getFramework = (document.body.dataset.framework || "unknown").trim() || "unknown";
        const getWorkerStatus = (document.querySelector("#metric-worker-status")?.textContent || "").trim().toLowerCase();
        const getWorkerBatch = (document.querySelector("#metric-worker-batch-ms")?.textContent || "").trim().toLowerCase();
        const getLastAction = (document.querySelector("#metric-last-action")?.textContent || "").trim().toLowerCase();
        const getMetricCoreCount = getSubjectNumber("#metric-core-count");
        const getCoreRowIDs = buildSubjectRowIDs(".benchmark-core-item");
        const hasCoreRowsAscending = hasSubjectAscendingRowIDs(getCoreRowIDs);
        const hasCoreRowsDescending = hasSubjectDescendingRowIDs(getCoreRowIDs);
        const getDescendingBreak = buildSubjectOrderingBreak(getCoreRowIDs, "descending");
        const getCoreRowHead = getCoreRowIDs.slice(0, 6).join(",");
        const getCoreRowTail = getCoreRowIDs.slice(Math.max(0, getCoreRowIDs.length - 6)).join(",");
        throw new Error(
            "timed out waiting for " + parseLabel +
            "; framework=" + getFramework +
            "; worker-status=" + getWorkerStatus +
            "; worker-batch=" + getWorkerBatch +
            "; last-action=" + getLastAction +
            "; metric-core-count=" + String(getMetricCoreCount) +
            "; core-count=" + getCoreRowIDs.length +
            "; core-ascending=" + String(hasCoreRowsAscending) +
            "; core-descending=" + String(hasCoreRowsDescending) +
            "; core-desc-break=" + (getDescendingBreak ? `${getDescendingBreak.getIndex - 1}:${getDescendingBreak.getPreviousRowID}<=${getDescendingBreak.getCurrentRowID}:[${getDescendingBreak.getWindow.join(",")}]` : "none") +
            "; core-head=[" + getCoreRowHead + "]" +
            "; core-tail=[" + getCoreRowTail + "]"
        );
    }

    function buildSubjectScenarioOrder(parseScenarioIDs, parseSeed) {
        const getScenarioIDs = Array.isArray(parseScenarioIDs) && parseScenarioIDs.length > 0
            ? parseScenarioIDs.slice()
            : storeBenchmarkScenarioConfigs.map((parseConfig) => parseConfig.id);
        let getState = (Number.parseInt(String(parseSeed || "1"), 10) || 1) >>> 0;
        function buildSubjectNextRandom() {
            getState = (getState * 1664525 + 1013904223) >>> 0;
            return getState / 0x100000000;
        }
        for (let parseIndex = getScenarioIDs.length - 1; parseIndex > 0; parseIndex--) {
            const getSwapIndex = Math.floor(buildSubjectNextRandom() * (parseIndex + 1));
            const getTemp = getScenarioIDs[parseIndex];
            getScenarioIDs[parseIndex] = getScenarioIDs[getSwapIndex];
            getScenarioIDs[getSwapIndex] = getTemp;
        }
        return getScenarioIDs;
    }

    function buildSubjectHeapBytes() {
        if (typeof performance.memory?.usedJSHeapSize !== "number") {
            return 0;
        }
        return performance.memory.usedJSHeapSize;
    }

    function buildSubjectWorkerSnapshot() {
        const getWorkerCountNode = document.querySelector("#metric-worker-count");
        if (!getWorkerCountNode) {
            return null;
        }
        return {
            getWorkerCount: getSubjectNumber("#metric-worker-count"),
            getPreparedBatchCount: getSubjectNumber("#metric-worker-batch-count"),
            getPreparedItems: getSubjectNumber("#metric-worker-items"),
            getLastBatchMs: getSubjectNumber("#metric-worker-batch-ms")
        };
    }

    function buildSubjectLongTaskProbe() {
        if (typeof PerformanceObserver !== "function") {
            return {
                getSupported: false,
                handleStop() {
                    return {
                        getLongTaskCount: 0,
                        getLongTaskDurationMs: 0
                    };
                }
            };
        }
        if (!Array.isArray(PerformanceObserver.supportedEntryTypes) || !PerformanceObserver.supportedEntryTypes.includes("longtask")) {
            return {
                getSupported: false,
                handleStop() {
                    return {
                        getLongTaskCount: 0,
                        getLongTaskDurationMs: 0
                    };
                }
            };
        }
        const getEntries = [];
        const getObserver = new PerformanceObserver((parseList) => {
            getEntries.push(...parseList.getEntries());
        });
        try {
            getObserver.observe({ entryTypes: ["longtask"] });
        } catch (parseErr) {
            return {
                getSupported: false,
                handleStop() {
                    return {
                        getLongTaskCount: 0,
                        getLongTaskDurationMs: 0
                    };
                }
            };
        }
        return {
            getSupported: true,
            handleStop() {
                getObserver.disconnect();
                const getLongTaskDurationMs = getEntries.reduce((parseTotal, parseEntry) => parseTotal + parseEntry.duration, 0);
                return {
                    getLongTaskCount: getEntries.length,
                    getLongTaskDurationMs: Number(getLongTaskDurationMs.toFixed(3))
                };
            }
        };
    }

    function buildSubjectMutationProbe() {
        const getContainerNode = document.querySelector("#benchmark-container") || document.querySelector("#benchmark-app") || document.body;
        const getMutationStats = {
            getMutationRecordCount: 0,
            getChildListMutationCount: 0,
            getAttributeMutationCount: 0,
            getCharacterDataMutationCount: 0,
            getAddedNodeCount: 0,
            getRemovedNodeCount: 0
        };
        const getObserver = new MutationObserver((parseRecords) => {
            getMutationStats.getMutationRecordCount += parseRecords.length;
            for (const parseRecord of parseRecords) {
                if (parseRecord.type === "childList") {
                    getMutationStats.getChildListMutationCount += 1;
                    getMutationStats.getAddedNodeCount += parseRecord.addedNodes.length;
                    getMutationStats.getRemovedNodeCount += parseRecord.removedNodes.length;
                    continue;
                }
                if (parseRecord.type === "attributes") {
                    getMutationStats.getAttributeMutationCount += 1;
                    continue;
                }
                if (parseRecord.type === "characterData") {
                    getMutationStats.getCharacterDataMutationCount += 1;
                }
            }
        });
        getObserver.observe(getContainerNode, {
            childList: true,
            subtree: true,
            attributes: true,
            characterData: true
        });
        return {
            handleStop() {
                getObserver.disconnect();
                return getMutationStats;
            }
        };
    }

    function buildSubjectRoundedNumber(parseValue) {
        if (!Number.isFinite(parseValue)) {
            return 0;
        }
        return Number(parseValue.toFixed(3));
    }

    function buildSubjectMean(parseValues) {
        if (!Array.isArray(parseValues) || parseValues.length === 0) {
            return 0;
        }
        return parseValues.reduce((parseTotal, parseValue) => parseTotal + parseValue, 0) / parseValues.length;
    }

    function buildSubjectMetricSummary(parseValues) {
        const getSortedValues = parseValues
            .map((parseValue) => Number(parseValue))
            .filter((parseValue) => Number.isFinite(parseValue))
            .sort((parseLeft, parseRight) => parseLeft - parseRight);
        if (getSortedValues.length === 0) {
            return {
                getMean: 0,
                getMedian: 0,
                getMin: 0,
                getMax: 0,
                getP95: 0,
                getStdDev: 0,
                getCoefficientOfVariation: 0,
                getSpreadRatio: 0,
                getRepresentative: 0,
                getRepresentativeSource: "empty",
                hasBimodalSamples: false,
                getBimodalGap: 0,
                getBimodalLowMean: 0,
                getBimodalHighMean: 0,
                getBimodalLowCount: 0,
                getBimodalHighCount: 0,
                getSamples: []
            };
        }
        const getMedianIndex = Math.floor(getSortedValues.length / 2);
        const getP95Index = Math.min(getSortedValues.length - 1, Math.floor(getSortedValues.length * 0.95));
        const getMean = buildSubjectMean(getSortedValues);
        const getVariance = buildSubjectMean(getSortedValues.map((parseValue) => {
            const getDistance = parseValue - getMean;
            return getDistance * getDistance;
        }));
        const getStdDev = Math.sqrt(getVariance);
        const getCoefficientOfVariation = Math.abs(getMean) > 0 ? getStdDev / Math.abs(getMean) : 0;
        const getSpreadRatio = getSortedValues[0] > 0 ? getSortedValues[getSortedValues.length - 1] / getSortedValues[0] : 0;
        let getBestSplit = null;
        if (getSortedValues.length >= 5) {
            for (let parseIndex = 2; parseIndex <= getSortedValues.length - 2; parseIndex++) {
                const getLowValues = getSortedValues.slice(0, parseIndex);
                const getHighValues = getSortedValues.slice(parseIndex);
                const getGap = getSortedValues[parseIndex] - getSortedValues[parseIndex - 1];
                const getLowSpread = getLowValues[getLowValues.length - 1] - getLowValues[0];
                const getHighSpread = getHighValues[getHighValues.length - 1] - getHighValues[0];
                if (getBestSplit && getBestSplit.getGap >= getGap) {
                    continue;
                }
                getBestSplit = {
                    getGap: getGap,
                    getLowMean: buildSubjectMean(getLowValues),
                    getHighMean: buildSubjectMean(getHighValues),
                    getLowCount: getLowValues.length,
                    getHighCount: getHighValues.length,
                    getLowSpread: getLowSpread,
                    getHighSpread: getHighSpread
                };
            }
        }
        const getBimodalGapThreshold = Math.max(0.25, Math.abs(getMean) * 0.2);
        const hasBimodalSamples = !!getBestSplit &&
            getCoefficientOfVariation >= 0.18 &&
            getBestSplit.getGap >= getBimodalGapThreshold &&
            getBestSplit.getGap >= Math.max(getBestSplit.getLowSpread, getBestSplit.getHighSpread, 0.001) * 1.5;
        let getRepresentative = getMean;
        let getRepresentativeSource = "mean";
        if (hasBimodalSamples) {
            if (getBestSplit.getLowCount > getBestSplit.getHighCount) {
                getRepresentative = getBestSplit.getLowMean;
                getRepresentativeSource = "dominant-low-cluster";
            } else if (getBestSplit.getHighCount > getBestSplit.getLowCount) {
                getRepresentative = getBestSplit.getHighMean;
                getRepresentativeSource = "dominant-high-cluster";
            } else {
                getRepresentative = getBestSplit.getHighMean;
                getRepresentativeSource = "conservative-high-cluster";
            }
        }
        return {
            getMean: buildSubjectRoundedNumber(getMean),
            getMedian: buildSubjectRoundedNumber(getSortedValues[getMedianIndex]),
            getMin: buildSubjectRoundedNumber(getSortedValues[0]),
            getMax: buildSubjectRoundedNumber(getSortedValues[getSortedValues.length - 1]),
            getP95: buildSubjectRoundedNumber(getSortedValues[getP95Index]),
            getStdDev: buildSubjectRoundedNumber(getStdDev),
            getCoefficientOfVariation: buildSubjectRoundedNumber(getCoefficientOfVariation),
            getSpreadRatio: buildSubjectRoundedNumber(getSpreadRatio),
            getRepresentative: buildSubjectRoundedNumber(getRepresentative),
            getRepresentativeSource: getRepresentativeSource,
            hasBimodalSamples: hasBimodalSamples,
            getBimodalGap: hasBimodalSamples ? buildSubjectRoundedNumber(getBestSplit.getGap) : 0,
            getBimodalLowMean: hasBimodalSamples ? buildSubjectRoundedNumber(getBestSplit.getLowMean) : 0,
            getBimodalHighMean: hasBimodalSamples ? buildSubjectRoundedNumber(getBestSplit.getHighMean) : 0,
            getBimodalLowCount: hasBimodalSamples ? getBestSplit.getLowCount : 0,
            getBimodalHighCount: hasBimodalSamples ? getBestSplit.getHighCount : 0,
            getSamples: getSortedValues.map((parseValue) => buildSubjectRoundedNumber(parseValue))
        };
    }

    function buildSubjectSummary(parseSamples, parseScenarioConfig, parseFramework) {
        const getDomReadyStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getDomReadyMs));
        const getPaintVisibleStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getPaintVisibleMs));
        const getPaintAfterDomStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getPaintAfterDomMs));
        const getMutationRecordStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getMutationRecordCount));
        const getChildListStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getChildListMutationCount));
        const getAttributeStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getAttributeMutationCount));
        const getCharacterDataStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getCharacterDataMutationCount));
        const getAddedNodeStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getAddedNodeCount));
        const getRemovedNodeStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getRemovedNodeCount));
        const getLongTaskCountStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getLongTaskCount));
        const getLongTaskDurationStats = buildSubjectMetricSummary(parseSamples.map((parseSample) => parseSample.getLongTaskDurationMs));
        const getHeapDeltaValues = parseSamples
            .map((parseSample) => parseSample.getHeapDeltaBytes)
            .filter((parseValue) => typeof parseValue === "number");
        const getHeapDeltaStats = getHeapDeltaValues.length > 0 ? buildSubjectMetricSummary(getHeapDeltaValues) : null;
        const getWorkerSamples = parseSamples.filter((parseSample) => parseSample.hasWorkerMetrics);
        const getWorkerBatchCountStats = getWorkerSamples.length > 0
            ? buildSubjectMetricSummary(getWorkerSamples.map((parseSample) => parseSample.getWorkerBatchCountDelta))
            : null;
        const getWorkerBatchStats = getWorkerSamples.length > 0
            ? buildSubjectMetricSummary(getWorkerSamples.map((parseSample) => parseSample.getWorkerBatchMs))
            : null;
        const getWorkerPreparedItemStats = getWorkerSamples.length > 0
            ? buildSubjectMetricSummary(getWorkerSamples.map((parseSample) => parseSample.getWorkerPreparedItems))
            : null;
        const getWorkerCountStats = getWorkerSamples.length > 0
            ? buildSubjectMetricSummary(getWorkerSamples.map((parseSample) => parseSample.getWorkerCount))
            : null;
        return {
            getFramework: parseFramework,
            getScenarioID: parseScenarioConfig.id,
            getScenarioLabel: parseScenarioConfig.label,
            getCategory: parseScenarioConfig.category,
            getRequestedWork: parseScenarioConfig.requestedWork,
            getCorrectnessCheck: parseScenarioConfig.correctnessCheck,
            getFinishLine: parseScenarioConfig.finishLine,
            getIterationCount: parseSamples.length,
            getDomReadyMeanMs: getDomReadyStats.getMean,
            getDomReadyMedianMs: getDomReadyStats.getMedian,
            getDomReadyMinMs: getDomReadyStats.getMin,
            getDomReadyMaxMs: getDomReadyStats.getMax,
            getDomReadyP95Ms: getDomReadyStats.getP95,
            getDomReadySamplesMs: getDomReadyStats.getSamples,
            getDomReadyStdDevMs: getDomReadyStats.getStdDev,
            getDomReadyCoefficientOfVariation: getDomReadyStats.getCoefficientOfVariation,
            getDomReadySpreadRatio: getDomReadyStats.getSpreadRatio,
            getDomReadyRepresentativeMs: getDomReadyStats.getRepresentative,
            getDomReadyRepresentativeSource: getDomReadyStats.getRepresentativeSource,
            hasDomReadyBimodalSamples: getDomReadyStats.hasBimodalSamples,
            getDomReadyBimodalGapMs: getDomReadyStats.getBimodalGap,
            getDomReadyBimodalLowMeanMs: getDomReadyStats.getBimodalLowMean,
            getDomReadyBimodalHighMeanMs: getDomReadyStats.getBimodalHighMean,
            getDomReadyBimodalLowCount: getDomReadyStats.getBimodalLowCount,
            getDomReadyBimodalHighCount: getDomReadyStats.getBimodalHighCount,
            getPaintVisibleMeanMs: getPaintVisibleStats.getMean,
            getPaintVisibleMedianMs: getPaintVisibleStats.getMedian,
            getPaintVisibleMinMs: getPaintVisibleStats.getMin,
            getPaintVisibleMaxMs: getPaintVisibleStats.getMax,
            getPaintVisibleP95Ms: getPaintVisibleStats.getP95,
            getPaintVisibleSamplesMs: getPaintVisibleStats.getSamples,
            getPaintAfterDomMeanMs: getPaintAfterDomStats.getMean,
            getPaintAfterDomMedianMs: getPaintAfterDomStats.getMedian,
            getPaintAfterDomP95Ms: getPaintAfterDomStats.getP95,
            getMutationRecordMean: getMutationRecordStats.getMean,
            getChildListMutationMean: getChildListStats.getMean,
            getAttributeMutationMean: getAttributeStats.getMean,
            getCharacterDataMutationMean: getCharacterDataStats.getMean,
            getAddedNodeMean: getAddedNodeStats.getMean,
            getRemovedNodeMean: getRemovedNodeStats.getMean,
            getLongTaskCountMean: getLongTaskCountStats.getMean,
            getLongTaskDurationMeanMs: getLongTaskDurationStats.getMean,
            hasHeapDelta: !!getHeapDeltaStats,
            getHeapDeltaMeanBytes: getHeapDeltaStats ? getHeapDeltaStats.getMean : 0,
            getHeapDeltaMedianBytes: getHeapDeltaStats ? getHeapDeltaStats.getMedian : 0,
            getHeapDeltaP95Bytes: getHeapDeltaStats ? getHeapDeltaStats.getP95 : 0,
            hasWorkerMetrics: !!getWorkerBatchStats,
            getWorkerCountMean: getWorkerCountStats ? getWorkerCountStats.getMean : 0,
            getWorkerBatchCountMean: getWorkerBatchCountStats ? getWorkerBatchCountStats.getMean : 0,
            getWorkerBatchMeanMs: getWorkerBatchStats ? getWorkerBatchStats.getMean : 0,
            getWorkerPreparedItemsMean: getWorkerPreparedItemStats ? getWorkerPreparedItemStats.getMean : 0
        };
    }

    async function handleSubjectMeasureScenario(parseScenarioID, parseOptions) {
        const getScenarioConfig = storeBenchmarkScenarioConfigs.find((parseConfig) => parseConfig.id === parseScenarioID);
        if (!getScenarioConfig) {
            throw new Error("unknown benchmark scenario " + parseScenarioID);
        }
        const getIterations = Math.max(1, Number.parseInt(parseOptions?.iterations ?? "7", 10) || 7);
        const getWarmups = Math.max(0, Number.parseInt(parseOptions?.warmups ?? "2", 10) || 2);
        for (let parseIndex = 0; parseIndex < getWarmups; parseIndex++) {
            const getWarmupContext = getScenarioConfig.prepare ? await getScenarioConfig.prepare() : undefined;
            await getScenarioConfig.run(getWarmupContext);
            await handleSubjectAnimationFrames(1);
        }
        const getSamples = [];
        for (let parseIndex = 0; parseIndex < getIterations; parseIndex++) {
            const getScenarioContext = getScenarioConfig.prepare ? await getScenarioConfig.prepare() : undefined;
            await handleSubjectWaitForDOMSettle(2, 30);
            const getWorkerBefore = buildSubjectWorkerSnapshot();
            const getLongTaskProbe = buildSubjectLongTaskProbe();
            const getMutationProbe = buildSubjectMutationProbe();
            const getHeapBeforeBytes = buildSubjectHeapBytes();
            const getStartedAt = performance.now();
            await getScenarioConfig.run(getScenarioContext);
            const getDomReadyAt = performance.now();
            await handleSubjectAnimationFrames(1);
            const getPaintVisibleAt = performance.now();
            if (getWorkerBefore) {
                await handleSubjectTaskTick();
                await handleSubjectAnimationFrames(1);
            }
            const getWorkerAfter = buildSubjectWorkerSnapshot();
            const getMutationStats = getMutationProbe.handleStop();
            const getLongTaskStats = getLongTaskProbe.handleStop();
            const getHeapAfterBytes = buildSubjectHeapBytes();
            const hasWorkerMetrics = !!getWorkerBefore && !!getWorkerAfter;
            getSamples.push({
                getDomReadyMs: getDomReadyAt - getStartedAt,
                getPaintVisibleMs: getPaintVisibleAt - getStartedAt,
                getPaintAfterDomMs: getPaintVisibleAt - getDomReadyAt,
                getMutationRecordCount: getMutationStats.getMutationRecordCount,
                getChildListMutationCount: getMutationStats.getChildListMutationCount,
                getAttributeMutationCount: getMutationStats.getAttributeMutationCount,
                getCharacterDataMutationCount: getMutationStats.getCharacterDataMutationCount,
                getAddedNodeCount: getMutationStats.getAddedNodeCount,
                getRemovedNodeCount: getMutationStats.getRemovedNodeCount,
                getLongTaskCount: getLongTaskStats.getLongTaskCount,
                getLongTaskDurationMs: getLongTaskStats.getLongTaskDurationMs,
                getHeapDeltaBytes: getHeapBeforeBytes > 0 && getHeapAfterBytes > 0 ? getHeapAfterBytes - getHeapBeforeBytes : null,
                hasWorkerMetrics: hasWorkerMetrics,
                getWorkerCount: hasWorkerMetrics ? getWorkerAfter.getWorkerCount : 0,
                getWorkerBatchCountDelta: hasWorkerMetrics ? Math.max(0, getWorkerAfter.getPreparedBatchCount - getWorkerBefore.getPreparedBatchCount) : 0,
                getWorkerBatchMs: hasWorkerMetrics && getWorkerAfter.getPreparedBatchCount > getWorkerBefore.getPreparedBatchCount ? getWorkerAfter.getLastBatchMs : 0,
                getWorkerPreparedItems: hasWorkerMetrics && getWorkerAfter.getPreparedBatchCount > getWorkerBefore.getPreparedBatchCount ? getWorkerAfter.getPreparedItems : 0
            });
        }
        return buildSubjectSummary(getSamples, getScenarioConfig, document.body.dataset.framework || "unknown");
    }

    async function handleSubjectMeasureAllScenarios(parseOptions) {
        const getScenarioOrder = buildSubjectScenarioOrder(parseOptions?.scenarioIDs, parseOptions?.seed);
        const getScenarioResults = [];
        for (const getScenarioID of getScenarioOrder) {
            getScenarioResults.push(await handleSubjectMeasureScenario(getScenarioID, parseOptions));
        }
        return {
            getScenarioOrder: getScenarioOrder,
            getScenarioResults: getScenarioResults
        };
    }

    async function handleSubjectRegister(parseFramework) {
        await handleSubjectWait(() => !!document.querySelector("#benchmark-app") && !!document.querySelector("#btn-core-render"), 20000, "benchmark subject ready");
        const getWorkerStatusNode = document.querySelector("#metric-worker-status");
        if (getWorkerStatusNode) {
            await handleSubjectWait(() => {
                const getWorkerStatusText = (document.querySelector("#metric-worker-status")?.textContent || "").trim().toLowerCase();
                return getWorkerStatusText === "ready";
            }, 20000, "benchmark worker ready");
        }
        document.body.dataset.framework = parseFramework;
        window.__example201Subject = {
            getFramework: parseFramework,
            getScenarioIDs: storeBenchmarkScenarioConfigs.map((parseConfig) => parseConfig.id),
            getScenarioLabels: storeBenchmarkScenarioConfigs.map((parseConfig) => ({
                getScenarioID: parseConfig.id,
                getScenarioLabel: parseConfig.label,
                getCategory: parseConfig.category,
                getRequestedWork: parseConfig.requestedWork,
                getCorrectnessCheck: parseConfig.correctnessCheck,
                getFinishLine: parseConfig.finishLine
            })),
            isReady: true,
            measureScenario: handleSubjectMeasureScenario,
            measureAllScenarios: handleSubjectMeasureAllScenarios
        };
    }

    window.__example201RegisterSubject = function (parseFramework) {
        return handleSubjectRegister(parseFramework).catch((parseErr) => {
            window.__example201SubjectError = String(parseErr);
            throw parseErr;
        });
    };
})();

(function () {
    const storeBenchmarkSubjectConfigs = [
        {
            getFramework: "react",
            getLabel: "React 18",
            getURL: "./react/"
        },
        {
            getFramework: "runtime1",
            getLabel: "Runtime 1",
            getURL: "./runtime/"
        },
        {
            getFramework: "runtime2",
            getLabel: "Runtime 2",
            getURL: "./runtime2/"
        },
        {
            getFramework: "runtime2-workers4",
            getLabel: "Runtime 2 (4 Workers)",
            getURL: "./runtime2-workers4/"
        }
    ];

    function buildRunnerConfig() {
        const getQuery = new URLSearchParams(window.location.search);
        return {
            getIterations: Math.max(1, Number.parseInt(getQuery.get("iterations") || "7", 10) || 7),
            getWarmups: Math.max(0, Number.parseInt(getQuery.get("warmups") || "2", 10) || 2),
            getSeed: Math.max(1, Number.parseInt(getQuery.get("seed") || String(Date.now()), 10) || Date.now())
        };
    }

    function buildRunnerShuffledList(parseItems, parseSeed) {
        const getItems = parseItems.slice();
        let getState = (Number.parseInt(String(parseSeed || "1"), 10) || 1) >>> 0;
        function buildRunnerNextRandom() {
            getState = (getState * 1664525 + 1013904223) >>> 0;
            return getState / 0x100000000;
        }
        for (let parseIndex = getItems.length - 1; parseIndex > 0; parseIndex--) {
            const getSwapIndex = Math.floor(buildRunnerNextRandom() * (parseIndex + 1));
            const getTemp = getItems[parseIndex];
            getItems[parseIndex] = getItems[getSwapIndex];
            getItems[getSwapIndex] = getTemp;
        }
        return getItems;
    }

    function handleRunnerDelay(parseDurationMs) {
        return new Promise((parseResolve) => {
            window.setTimeout(parseResolve, parseDurationMs);
        });
    }

    function getRunnerFrame(parseFramework) {
        return document.querySelector(`[data-benchmark-frame="${parseFramework}"]`);
    }

    async function handleRunnerLoadSubject(parseSubjectConfig) {
        const getFrame = getRunnerFrame(parseSubjectConfig.getFramework);
        if (!getFrame) {
            throw new Error("missing frame for " + parseSubjectConfig.getFramework);
        }
        getFrame.src = `${parseSubjectConfig.getURL}?framework=${encodeURIComponent(parseSubjectConfig.getFramework)}&v=${Date.now()}`;
        const getStartedAt = performance.now();
        while ((performance.now() - getStartedAt) < 45000) {
            if (getFrame.contentWindow) {
                if (getFrame.contentWindow.__example201SubjectError) {
                    throw new Error(`${parseSubjectConfig.getFramework} subject error: ${getFrame.contentWindow.__example201SubjectError}`);
                }
                if (getFrame.contentWindow.__example201Subject && getFrame.contentWindow.__example201Subject.isReady) {
                    return getFrame;
                }
            }
            await handleRunnerDelay(20);
        }
        const getDebugURL = getFrame.contentWindow ? getFrame.contentWindow.location.href : "(no contentWindow)";
        const getDebugBody = getFrame.contentDocument && getFrame.contentDocument.body
            ? getFrame.contentDocument.body.textContent.slice(0, 240)
            : "(no body)";
        throw new Error(`timed out waiting for ${parseSubjectConfig.getFramework} subject load; url=${getDebugURL}; body=${getDebugBody}`);
    }

    function buildRunnerScenarioRows(parseReport) {
        const getScenarioRows = [];
        const getScenarioLabels = parseReport.getScenarioLabels || [];
        for (const getScenarioLabel of getScenarioLabels) {
            const getRow = {
                getScenarioID: getScenarioLabel.getScenarioID,
                getScenarioLabel: getScenarioLabel.getScenarioLabel,
                getCategory: getScenarioLabel.getCategory,
                getRequestedWork: getScenarioLabel.getRequestedWork,
                getCorrectnessCheck: getScenarioLabel.getCorrectnessCheck,
                getFinishLine: getScenarioLabel.getFinishLine,
                getFrameworks: []
            };
            let getFastestPaintVisible = Number.POSITIVE_INFINITY;
            for (const getFramework of parseReport.getFrameworks) {
                const getScenarioResult = getFramework.getScenarioResults.find((parseResult) => parseResult.getScenarioID === getScenarioLabel.getScenarioID);
                if (!getScenarioResult) {
                    continue;
                }
                getFastestPaintVisible = Math.min(getFastestPaintVisible, getScenarioResult.getPaintVisibleMeanMs);
                getRow.getFrameworks.push({
                    getFramework: getFramework.getFramework,
                    getLabel: getFramework.getLabel,
                    getDomReadyMeanMs: getScenarioResult.getDomReadyMeanMs,
                    getDomReadyMedianMs: getScenarioResult.getDomReadyMedianMs,
                    getPaintVisibleMeanMs: getScenarioResult.getPaintVisibleMeanMs,
                    getPaintVisibleMedianMs: getScenarioResult.getPaintVisibleMedianMs,
                    getPaintAfterDomMeanMs: getScenarioResult.getPaintAfterDomMeanMs,
                    getChildListMutationMean: getScenarioResult.getChildListMutationMean,
                    getAttributeMutationMean: getScenarioResult.getAttributeMutationMean,
                    getCharacterDataMutationMean: getScenarioResult.getCharacterDataMutationMean,
                    getAddedNodeMean: getScenarioResult.getAddedNodeMean,
                    getRemovedNodeMean: getScenarioResult.getRemovedNodeMean,
                    getLongTaskCountMean: getScenarioResult.getLongTaskCountMean,
                    getLongTaskDurationMeanMs: getScenarioResult.getLongTaskDurationMeanMs
                });
            }
            for (const getFramework of getRow.getFrameworks) {
                getFramework.getRelativePaintVisible = Number((getFramework.getPaintVisibleMeanMs / getFastestPaintVisible).toFixed(3));
            }
            getScenarioRows.push(getRow);
        }
        return getScenarioRows;
    }

    function buildRunnerCategoryRows(parseScenarioRows) {
        const getCategoryMap = new Map();
        for (const getScenarioRow of parseScenarioRows) {
            if (!getCategoryMap.has(getScenarioRow.getCategory)) {
                getCategoryMap.set(getScenarioRow.getCategory, []);
            }
            getCategoryMap.get(getScenarioRow.getCategory).push(getScenarioRow);
        }
        return Array.from(getCategoryMap.entries()).map(([parseCategory, parseRows]) => {
            const getFrameworkMap = new Map();
            for (const getScenarioRow of parseRows) {
                for (let parseIndex = 0; parseIndex < getScenarioRow.getFrameworks.length; parseIndex++) {
                    const getFramework = getScenarioRow.getFrameworks[parseIndex];
                    if (!getFrameworkMap.has(getFramework.getFramework)) {
                        getFrameworkMap.set(getFramework.getFramework, {
                            getFramework: getFramework.getFramework,
                            getLabel: getFramework.getLabel,
                            getPaintVisibleRelativeProduct: 1,
                            getDomReadySumMs: 0,
                            getPaintVisibleSumMs: 0,
                            getScenarioCount: 0,
                            getScenarioWins: 0
                        });
                    }
                    const getCategoryFramework = getFrameworkMap.get(getFramework.getFramework);
                    getCategoryFramework.getPaintVisibleRelativeProduct *= Math.max(getFramework.getRelativePaintVisible, 0.0001);
                    getCategoryFramework.getDomReadySumMs += getFramework.getDomReadyMeanMs;
                    getCategoryFramework.getPaintVisibleSumMs += getFramework.getPaintVisibleMeanMs;
                    getCategoryFramework.getScenarioCount += 1;
                    if (parseIndex === 0 || getFramework.getRelativePaintVisible === 1) {
                        getCategoryFramework.getScenarioWins += 1;
                    }
                }
            }
            const getFrameworks = Array.from(getFrameworkMap.values()).map((parseFramework) => ({
                getFramework: parseFramework.getFramework,
                getLabel: parseFramework.getLabel,
                getScenarioCount: parseFramework.getScenarioCount,
                getScenarioWins: parseFramework.getScenarioWins,
                getDomReadyMeanMs: Number((parseFramework.getDomReadySumMs / parseFramework.getScenarioCount).toFixed(3)),
                getPaintVisibleMeanMs: Number((parseFramework.getPaintVisibleSumMs / parseFramework.getScenarioCount).toFixed(3)),
                getPaintVisibleGeometricRelative: Number(Math.pow(parseFramework.getPaintVisibleRelativeProduct, 1 / parseFramework.getScenarioCount).toFixed(3))
            })).sort((parseLeft, parseRight) => parseLeft.getPaintVisibleGeometricRelative - parseRight.getPaintVisibleGeometricRelative);
            return {
                getCategory: parseCategory,
                getScenarioCount: parseRows.length,
                getFrameworks: getFrameworks
            };
        });
    }

    function renderRunnerSummary(parseReport) {
        const getResultsNode = document.querySelector("#benchmark-results");
        const getRawNode = document.querySelector("#benchmark-report-json");
        const getCategoryNode = document.querySelector("#benchmark-category-results");
        if (!getResultsNode || !getRawNode || !getCategoryNode) {
            return;
        }
        const getScenarioRows = buildRunnerScenarioRows(parseReport);
        const getCategoryRows = buildRunnerCategoryRows(getScenarioRows);
        getCategoryNode.innerHTML = getCategoryRows.map((parseCategoryRow) => {
            const getFrameworkRows = parseCategoryRow.getFrameworks.map((parseFramework) => `
                <tr>
                    <td>${parseFramework.getLabel}</td>
                    <td>${parseFramework.getDomReadyMeanMs.toFixed(3)} ms</td>
                    <td>${parseFramework.getPaintVisibleMeanMs.toFixed(3)} ms</td>
                    <td>${parseFramework.getPaintVisibleGeometricRelative.toFixed(3)}x</td>
                    <td>${parseFramework.getScenarioWins}</td>
                </tr>
            `).join("");
            return `
                <section class="benchmark-card benchmark-stack" style="padding: 18px;">
                    <div class="benchmark-inline" style="justify-content: space-between;">
                        <h3 style="margin: 0; font-size: 18px;">${parseCategoryRow.getCategory}</h3>
                        <span class="benchmark-muted">${parseCategoryRow.getScenarioCount} scenarios</span>
                    </div>
                    <table class="benchmark-table" style="margin-top: 12px;">
                        <thead>
                            <tr>
                                <th>Framework</th>
                                <th>Avg DOM Ready</th>
                                <th>Avg Paint Proxy</th>
                                <th>Geom. Relative</th>
                                <th>Wins</th>
                            </tr>
                        </thead>
                        <tbody>${getFrameworkRows}</tbody>
                    </table>
                </section>
            `;
        }).join("");
        getResultsNode.innerHTML = getScenarioRows.map((parseRow) => {
            const getFrameworkRows = parseRow.getFrameworks.map((parseFramework) => {
                const isFastest = parseFramework.getRelativePaintVisible === 1;
                return `
                    <tr>
                        <td>${parseFramework.getLabel}</td>
                        <td>${parseFramework.getDomReadyMeanMs.toFixed(3)} ms</td>
                        <td>${parseFramework.getPaintVisibleMeanMs.toFixed(3)} ms</td>
                        <td>${parseFramework.getPaintAfterDomMeanMs.toFixed(3)} ms</td>
                        <td>${parseFramework.getChildListMutationMean.toFixed(1)} / ${parseFramework.getAttributeMutationMean.toFixed(1)} / ${parseFramework.getCharacterDataMutationMean.toFixed(1)}</td>
                        <td>${parseFramework.getAddedNodeMean.toFixed(1)} / ${parseFramework.getRemovedNodeMean.toFixed(1)}</td>
                        <td>${parseFramework.getLongTaskCountMean.toFixed(1)} / ${parseFramework.getLongTaskDurationMeanMs.toFixed(3)} ms</td>
                        <td class="${isFastest ? "benchmark-rank-fastest" : "benchmark-rank-slower"}">${parseFramework.getRelativePaintVisible.toFixed(3)}x</td>
                    </tr>
                `;
            }).join("");
            return `
                <section class="benchmark-card benchmark-stack" style="padding: 18px;">
                    <div class="benchmark-inline" style="justify-content: space-between;">
                        <h3 style="margin: 0; font-size: 18px;">${parseRow.getScenarioLabel}</h3>
                        <span class="benchmark-muted">${parseRow.getScenarioID} / ${parseRow.getCategory}</span>
                    </div>
                    <p class="benchmark-muted" style="margin: 0;">Requested work: ${parseRow.getRequestedWork}</p>
                    <p class="benchmark-muted" style="margin: 0;">Correctness check: ${parseRow.getCorrectnessCheck}</p>
                    <p class="benchmark-muted" style="margin: 0;">Finish line: ${parseRow.getFinishLine}</p>
                    <table class="benchmark-table" style="margin-top: 12px;">
                        <thead>
                            <tr>
                                <th>Framework</th>
                                <th>DOM Ready</th>
                                <th>Paint Proxy</th>
                                <th>Paint-After-DOM</th>
                                <th>Mutations C/A/T</th>
                                <th>Nodes + / -</th>
                                <th>Long Tasks</th>
                                <th>Relative Paint</th>
                            </tr>
                        </thead>
                        <tbody>${getFrameworkRows}</tbody>
                    </table>
                </section>
            `;
        }).join("");
        getRawNode.textContent = JSON.stringify(parseReport, null, 2);
    }

    async function handleRunnerStart() {
        const getStatusNode = document.querySelector("#benchmark-status");
        const getConfig = buildRunnerConfig();
        const getFrameworkOrder = buildRunnerShuffledList(storeBenchmarkSubjectConfigs, getConfig.getSeed);
        if (getStatusNode) {
            getStatusNode.textContent = "Loading benchmark subjects...";
        }
        const getFrameworks = [];
        let getScenarioLabels = [];
        let getScenarioOrder = [];
        for (const getSubjectConfig of getFrameworkOrder) {
            const getFrame = await handleRunnerLoadSubject(getSubjectConfig);
            if (getStatusNode) {
                getStatusNode.textContent = "Running " + getSubjectConfig.getLabel + "...";
            }
            const getMeasureResult = await getFrame.contentWindow.__example201Subject.measureAllScenarios({
                iterations: getConfig.getIterations,
                warmups: getConfig.getWarmups,
                seed: getConfig.getSeed,
                scenarioIDs: getScenarioOrder
            });
            if (getScenarioLabels.length === 0) {
                getScenarioLabels = getFrame.contentWindow.__example201Subject.getScenarioLabels || [];
            }
            if (getScenarioOrder.length === 0) {
                getScenarioOrder = getMeasureResult.getScenarioOrder || [];
            }
            getFrameworks.push({
                getFramework: getSubjectConfig.getFramework,
                getLabel: getSubjectConfig.getLabel,
                getScenarioResults: getMeasureResult.getScenarioResults || []
            });
        }
        const getReport = {
            generatedAt: new Date().toISOString(),
            getIterations: getConfig.getIterations,
            getWarmups: getConfig.getWarmups,
            getSeed: getConfig.getSeed,
            getScenarioOrder: getScenarioOrder,
            getFrameworkOrder: getFrameworkOrder.map((parseConfig) => parseConfig.getFramework),
            getScenarioLabels: getScenarioLabels,
            getFrameworks: getFrameworks
        };
        window.__example201Report = getReport;
        renderRunnerSummary(getReport);
        if (getStatusNode) {
            getStatusNode.textContent = "Benchmark run complete.";
        }
        return getReport;
    }

    function handleRunnerBind() {
        const getButton = document.querySelector("#btn-run-benchmark");
        if (!getButton) {
            return;
        }
        getButton.addEventListener("click", () => {
            handleRunnerStart().catch((parseErr) => {
                const getStatusNode = document.querySelector("#benchmark-status");
                if (getStatusNode) {
                    getStatusNode.textContent = "Benchmark failed: " + String(parseErr);
                }
                window.__example201ReportError = String(parseErr);
                throw parseErr;
            });
        });
        window.__example201Runner = {
            runBenchmarks: handleRunnerStart
        };
    }

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", handleRunnerBind);
    } else {
        handleRunnerBind();
    }
})();

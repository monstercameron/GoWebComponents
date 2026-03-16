(() => {
  if (window.GWCExampleLogger) {
    return;
  }

  const noop = () => undefined;
  window.GWCExampleLogger = { log: noop, error: noop };
})();

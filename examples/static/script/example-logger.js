(function(){
  if (window.__gwcExampleLogger) return;
  const copyPrefix = '[gwc example]';
  window.__gwcExampleLogger = {
    log: (...copyArgs) => console.log(copyPrefix, ...copyArgs),
    info: (...copyArgs) => console.info(copyPrefix, ...copyArgs),
    warn: (...copyArgs) => console.warn(copyPrefix, ...copyArgs),
    error: (...copyArgs) => console.error(copyPrefix, ...copyArgs)
  };
})();

export const bundleName = 'interop-demo-module';

export function formatLabel(input) {
  return `lazy-module:${input}`;
}

export default function defaultLabel(input) {
  return `default-module:${input}`;
}

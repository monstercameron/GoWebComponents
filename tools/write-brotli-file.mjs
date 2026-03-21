import { readFile, writeFile } from 'node:fs/promises';
import { brotliCompressSync, constants as zlibConstants } from 'node:zlib';

const [, , sourcePath, targetPath] = process.argv;

if (!sourcePath || !targetPath) {
  console.error('usage: node tools/write-brotli-file.mjs <source> <target>');
  process.exit(2);
}

const input = await readFile(sourcePath);
const output = brotliCompressSync(input, {
  params: {
    [zlibConstants.BROTLI_PARAM_QUALITY]: 11,
    [zlibConstants.BROTLI_PARAM_MODE]: zlibConstants.BROTLI_MODE_GENERIC,
  },
});

await writeFile(targetPath, output);
// 仅在镜像构建中执行：安装已固定且通过 SHA512 核验的官方 npm 发布包。
const { createHash, timingSafeEqual } = require('node:crypto');
const { writeFile, rm, mkdir, readFile } = require('node:fs/promises');
const { execFileSync } = require('node:child_process');
const manifest = require('./npm-runtime.json');

async function main() {
  const expectedUrl = `https://registry.npmjs.org/npm/-/npm-${manifest.version}.tgz`;
  if (manifest.tarball !== expectedUrl || !manifest.integrity.startsWith('sha512-')) {
    throw new Error('npm 运行时清单来源或完整性字段无效');
  }
  const response = await fetch(manifest.tarball, { signal: AbortSignal.timeout(60000) });
  if (!response.ok) throw new Error(`npm 下载失败：HTTP ${response.status}`);
  const chunks = [];
  let size = 0;
  for await (const chunk of response.body) {
    size += chunk.length;
    if (size > 20 * 1024 * 1024) throw new Error('npm 发布包超过 20 MiB');
    chunks.push(chunk);
  }
  const archive = Buffer.concat(chunks);
  const actual = createHash('sha512').update(archive).digest();
  const expected = Buffer.from(manifest.integrity.slice(7), 'base64');
  if (actual.length !== expected.length || !timingSafeEqual(actual, expected)) {
    throw new Error('npm 发布包 SHA512 不匹配');
  }
  const archivePath = '/tmp/kexra-npm/runtime.tgz';
  const destination = '/usr/local/lib/node_modules/npm';
  await writeFile(archivePath, archive, { mode: 0o600 });
  await rm(destination, { recursive: true, force: true });
  await mkdir(destination, { recursive: true });
  execFileSync('tar', ['-xzf', archivePath, '-C', destination, '--strip-components=1']);
  const installed = JSON.parse(await readFile(`${destination}/package.json`, 'utf8'));
  if (installed.name !== 'npm' || installed.version !== manifest.version) {
    throw new Error('安装后的 npm 版本与清单不同');
  }
  console.log(`已核验并安装 npm ${installed.version}`);
}

main().catch(error => { console.error(error); process.exitCode = 1; });

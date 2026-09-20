// Release metadata must agree before any native installer is built.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
const read = path => readFileSync(new URL(`../../${path}`, import.meta.url), 'utf8');
const version = read('VERSION').trim();
assert.match(version, /^\d+\.\d+\.\d+$/);
const check = (label, actual, expected = version) => assert.equal(actual, expected, label);
check('Go application', read('main.go').match(/appVersion\s*=\s*"([^"]+)"/)[1]);
check('Frontend', JSON.parse(read('frontend/package.json')).version);
check('Demo diagnostics', read('frontend/src/lib/backend.ts').match(/appVersion:\s*"([^"]+)"/)[1], `${version}-dev`);
check('Wails build configuration', read('build/config.yml').match(/version:\s*"([^"]+)"/)[1]);
const windows = JSON.parse(read('build/windows/info.json'));
check('Windows file version', windows.fixed.file_version);
check('Windows product version', windows.info['0000'].ProductVersion);
check('NSIS', read('build/windows/nsis/wails_tools.nsh').match(/!define INFO_PRODUCTVERSION "([^"]+)"/)[1]);
check('Windows application manifest', read('build/windows/wails.exe.manifest').match(/assemblyIdentity[^>]*version="([^"]+)"/)[1], `${version}.0`);
for (const path of ['build/windows/msix/app_manifest.xml', 'build/windows/msix/template.xml']) {
  check(path, read(path).match(/Version="([^"]+)"/)[1], `${version}.0`);
}
check('Linux package', read('build/linux/nfpm/nfpm.yaml').match(/^version: "([^"]+)"/m)[1]);
for (const path of ['build/darwin/Info.plist', 'build/darwin/Info.dev.plist', 'build/ios/Info.plist', 'build/ios/Info.dev.plist']) {
  const plist = read(path);
  for (const key of ['CFBundleShortVersionString', 'CFBundleVersion']) {
    const actual = plist.match(new RegExp(`<key>${key}</key>\\s*<string>([^<]+)</string>`))[1];
    check(`${path} ${key}`, actual, path === 'build/ios/Info.dev.plist' && key === 'CFBundleShortVersionString' ? `${version}-dev` : version);
  }
}
console.log(`Application and installer metadata agree: ${version}`);

import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';

const template = readFileSync(new URL('../../build/darwin/installer/Distribution.xml', import.meta.url), 'utf8');
const script = template.match(/<!\[CDATA\[([\s\S]*?)\]\]>/)[1];
const managed = '/Library/LaunchDaemons/io.headscaleclient.tailscaled.plist';
const official = '/Applications/Tailscale.app';
const brew = '/Library/LaunchDaemons/homebrew.mxcl.tailscale.plist';
function visibleChoices(paths) {
  const context = { system: { files: { fileExistsAtPath: path => paths.includes(path) } } };
  runInNewContext(script, context);
  return [...template.matchAll(/<choice\s+id="([^"]+)"([\s\S]*?)(?:\/>|>)/g)]
    .filter(([, , attributes]) => {
      const expression = attributes.match(/visible="([^"]+)"/);
      return !expression || runInNewContext(expression[1].replaceAll('&amp;', '&'), context);
    }).map(([, id]) => id);
}
test('Installer enables JavaScript expressions and nests OS constraints correctly', () => {
  assert.match(template, /require-scripts="true"/);
  assert.match(template, /<volume-check><allowed-os-versions>/);
  assert.match(template, /hostArchitectures="@ARCH@"/);
  assert.match(template, /enable_currentUserHome="false"/);
});
for (const [name, paths, hint] of [
  ['fresh', [], 'automatic'], ['official', [official], 'reuse'],
  ['Homebrew', [brew], 'reuse'], ['managed', [managed], 'upgrade'],
  ['conflict', [managed, official], 'conflict'],
  ['manual socket', ['/var/run/tailscaled.socket'], 'reuse'],
]) {
  test(`Installer shows the ${name} policy without a service-selection checkbox`, () => {
    assert.deepEqual(visibleChoices(paths), ['client', hint]);
    assert.match(template, /id="client"[^>]*enabled="false" selected="true"/);
  });
}

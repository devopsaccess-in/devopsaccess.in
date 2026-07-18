// Generates site/public/og.png (1200x630) — the default social share image.
// Pure Node (zlib only, no deps) so it runs anywhere, including CI as a prebuild
// step. On-brand: dark navy base with the teal/amber radial glows from the site
// background, plus the "devopsaccess.in" wordmark in a blocky terminal font.
import { deflateSync } from "node:zlib";
import { writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const W = 1200,
  H = 630;

// Brand palette (matches src/styles/global.css).
const BASE = [11, 17, 32]; // #0b1120
const TEAL = [45, 212, 191];
const AMBER = [245, 166, 35];
const WHITE = [230, 235, 240];

// 5x7 lowercase glyphs for the characters in "devopsaccess.in".
const FONT = {
  a: ["00000", "00000", "01110", "00001", "01111", "10001", "01111"],
  c: ["00000", "00000", "01110", "10001", "10000", "10001", "01110"],
  d: ["00001", "00001", "01101", "10011", "10001", "10011", "01101"],
  e: ["00000", "00000", "01110", "10001", "11111", "10000", "01110"],
  i: ["00100", "00000", "01100", "00100", "00100", "00100", "01110"],
  n: ["00000", "00000", "10110", "11001", "10001", "10001", "10001"],
  o: ["00000", "00000", "01110", "10001", "10001", "10001", "01110"],
  p: ["00000", "00000", "11110", "10001", "11110", "10000", "10000"],
  s: ["00000", "00000", "01111", "10000", "01110", "00001", "11110"],
  v: ["00000", "00000", "10001", "10001", "10001", "01010", "00100"],
  ".": ["00000", "00000", "00000", "00000", "00000", "01100", "01100"],
  " ": ["00000", "00000", "00000", "00000", "00000", "00000", "00000"],
};

const px = new Uint8Array(W * H * 4);

function set(x, y, [r, g, b]) {
  if (x < 0 || x >= W || y < 0 || y >= H) return;
  const i = (y * W + x) * 4;
  px[i] = r;
  px[i + 1] = g;
  px[i + 2] = b;
  px[i + 3] = 255;
}

// Background: base + two additive radial glows.
function glow(x, y, cx, cy, radius, intensity, color, out) {
  const d = Math.hypot(x - cx, y - cy);
  const f = Math.max(0, 1 - d / radius) * intensity;
  for (let k = 0; k < 3; k++) out[k] = Math.min(255, out[k] + color[k] * f);
}
for (let y = 0; y < H; y++) {
  for (let x = 0; x < W; x++) {
    const c = [...BASE];
    glow(x, y, 0.8 * W, -0.1 * H, 0.75 * W, 0.55, TEAL, c);
    glow(x, y, 0.0 * W, 0.2 * H, 0.6 * W, 0.4, AMBER, c);
    set(x, y, c);
  }
}

// Draw a string of glyphs at (x,y) scaled by s, in the given color.
function text(str, x, y, s, color) {
  let cx = x;
  for (const ch of str) {
    const g = FONT[ch] ?? FONT[" "];
    for (let r = 0; r < 7; r++) {
      for (let cN = 0; cN < 5; cN++) {
        if (g[r][cN] === "1") {
          for (let dy = 0; dy < s; dy++)
            for (let dx = 0; dx < s; dx++) set(cx + cN * s + dx, y + r * s + dy, color);
        }
      }
    }
    cx += 6 * s; // 5 cols + 1 spacing
  }
}

const word = "devopsaccess";
const tld = ".in";
const S = 11;
const glyphW = 6 * S;
const totalW = (word.length + tld.length) * glyphW - S;
const startX = Math.round((W - totalW) / 2);
const startY = Math.round((H - 7 * S) / 2) - 10;

// teal prompt block before the wordmark
for (let dy = 0; dy < 7 * S; dy++)
  for (let dx = 0; dx < Math.round(S * 0.8); dx++) set(startX - 3 * S + dx, startY + dy, TEAL);

text(word, startX, startY, S, WHITE);
text(tld, startX + word.length * glyphW, startY, S, TEAL);

// accent underline
for (let dx = 0; dx < totalW; dx++)
  for (let dy = 0; dy < 4; dy++) set(startX + dx, startY + 7 * S + 22 + dy, [...TEAL]);

// Encode PNG (truecolor + alpha).
function crc32(buf) {
  let c = ~0;
  for (let i = 0; i < buf.length; i++) {
    c ^= buf[i];
    for (let k = 0; k < 8; k++) c = (c >>> 1) ^ (0xedb88320 & -(c & 1));
  }
  return ~c >>> 0;
}
function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const td = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(td));
  return Buffer.concat([len, td, crc]);
}

const ihdr = Buffer.alloc(13);
ihdr.writeUInt32BE(W, 0);
ihdr.writeUInt32BE(H, 4);
ihdr[8] = 8; // bit depth
ihdr[9] = 6; // RGBA
// 10,11,12 = 0 (compression, filter, interlace)

// Raw scanlines, each prefixed with filter byte 0.
const raw = Buffer.alloc(H * (1 + W * 4));
for (let y = 0; y < H; y++) {
  const ro = y * (1 + W * 4);
  raw[ro] = 0;
  px.subarray(y * W * 4, (y + 1) * W * 4).forEach((b, i) => (raw[ro + 1 + i] = b));
}

const png = Buffer.concat([
  Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
  chunk("IHDR", ihdr),
  chunk("IDAT", deflateSync(raw, { level: 9 })),
  chunk("IEND", Buffer.alloc(0)),
]);

const out = fileURLToPath(new URL("../public/og.png", import.meta.url));
writeFileSync(out, png);
console.log(`[gen-og] wrote ${out} (${png.length} bytes)`);

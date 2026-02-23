import * as esbuild from 'esbuild';
import { cpSync } from 'fs';

const isDev = process.argv.includes('--dev');
const isProd = process.argv.includes('--production');

// Production builds go to server/static/ for Docker packaging
const outdir = isProd ? '../server/static' : 'dist';

const ctx = await esbuild.context({
  entryPoints: ['src/main.ts'],
  bundle: true,
  outfile: `${outdir}/bundle.js`,
  minify: !isDev,
  sourcemap: isDev,
  target: 'es2020',
  format: 'iife',
});

if (isDev) {
  await ctx.watch();
  console.log('Watching for changes...');
} else {
  await ctx.rebuild();
  await ctx.dispose();

  // Copy index.html next to bundle
  cpSync('index.html', `${outdir}/index.html`);

  console.log(`Build complete → ${outdir}/`);
}

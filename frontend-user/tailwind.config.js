export default {
  content: ['./index.html', './src/**/*.{vue,js}'],
  theme: {
    extend: {
      colors: {
        bg: '#0c0e12',
        elev: '#141820',
        sunken: '#090b0e',
        line: '#2a3140',
        brass: '#c9a227',
        'brass-dim': '#8a7018',
        cyan: '#7ec8c4',
        ink: '#e8e4d9',
        muted: '#8b907c',
        danger: '#d45d4c',
        ok: '#7aaf6a',
      },
      fontFamily: {
        display: ['Syne', 'sans-serif'],
        serif: ['"Source Serif 4"', 'serif'],
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'monospace'],
      },
    },
  },
  plugins: [],
}

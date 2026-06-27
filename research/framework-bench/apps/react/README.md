# Bill Splitter — React (TypeScript + Vite)

Implements [../../SPEC.md](../../SPEC.md). Shared design from
[`../../shared/styles.css`](../../shared/styles.css).

```bash
npm install
npm run dev      # http://localhost:5173
```

| Dimension | Value |
|---|---|
| Language | TypeScript |
| Shared state | React Context (`SettingsContext`) for `theme` + `roundUp` |
| Local state | `useState` for `bill` / `tipPercent` / `people` |
| Reactivity | Virtual DOM diff on `setState` |
| Build | Vite + `@vitejs/plugin-react` |
| Styling | imports the canonical `shared/styles.css` |

`node_modules/` and `dist/` are git-ignored.
